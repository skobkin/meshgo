package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/traceroute"
)

type stubTracerouteSender struct {
	send func(to uint32, channel uint32) (string, error)
}

func (s stubTracerouteSender) SendTraceroute(to uint32, channel uint32) (string, error) {
	return s.send(to, channel)
}

func TestTracerouteServiceStartTraceroute_EnforcesGlobalCooldown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	nodeStore := domain.NewNodeStore()
	channel := uint32(2)
	nodeStore.Upsert(domain.Node{NodeID: "!0000002a", Channel: &channel})

	sendCalls := 0
	service := NewTracerouteService(
		messageBus,
		stubTracerouteSender{
			send: func(to uint32, ch uint32) (string, error) {
				sendCalls++
				if to != 0x2a {
					t.Fatalf("unexpected destination: %d", to)
				}
				if ch != 2 {
					t.Fatalf("unexpected channel: %d", ch)
				}

				return "42", nil
			},
		},
		nodeStore,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
		0,
	)

	first, err := service.StartTraceroute(context.Background(), TracerouteTarget{NodeID: "!0000002a"})
	if err != nil {
		t.Fatalf("start traceroute: %v", err)
	}
	if first.RequestID != 42 {
		t.Fatalf("unexpected request id: %d", first.RequestID)
	}
	if first.Status != traceroute.StatusStarted {
		t.Fatalf("unexpected status: %s", first.Status)
	}

	_, err = service.StartTraceroute(context.Background(), TracerouteTarget{NodeID: "!0000002b"})
	if err == nil {
		t.Fatalf("expected cooldown error")
	}
	var cooldownErr *TracerouteCooldownError
	if !errors.As(err, &cooldownErr) {
		t.Fatalf("expected cooldown error, got %T (%v)", err, err)
	}
	if sendCalls != 1 {
		t.Fatalf("unexpected send calls: %d", sendCalls)
	}
}

func TestTracerouteServiceProgressAndCompletion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	service := NewTracerouteService(
		messageBus,
		stubTracerouteSender{
			send: func(_ uint32, _ uint32) (string, error) {
				return "100", nil
			},
		},
		nil,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
		0,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	sub := messageBus.Subscribe(connectors.TopicTracerouteUpdate)
	defer messageBus.Unsubscribe(sub, connectors.TopicTracerouteUpdate)

	if _, err := service.StartTraceroute(context.Background(), TracerouteTarget{NodeID: "!0000002a"}); err != nil {
		t.Fatalf("start traceroute: %v", err)
	}

	waitTracerouteStatus(t, sub, traceroute.StatusStarted)

	messageBus.Publish(connectors.TopicTraceroute, connectors.TracerouteEvent{
		RequestID: 100,
		Route:     []uint32{0x2a, 0x10},
		SnrTowards: []int32{
			24,
		},
	})
	progress := waitTracerouteStatus(t, sub, traceroute.StatusProgress)
	if len(progress.ForwardRoute) != 2 {
		t.Fatalf("unexpected forward route length: %d", len(progress.ForwardRoute))
	}
	if len(progress.ReturnRoute) != 0 {
		t.Fatalf("unexpected return route length: %d", len(progress.ReturnRoute))
	}

	messageBus.Publish(connectors.TopicTraceroute, connectors.TracerouteEvent{
		RequestID:  100,
		Route:      []uint32{0x2a, 0x10},
		SnrTowards: []int32{24},
		RouteBack:  []uint32{0x10, 0x2a},
		SnrBack:    []int32{20},
		IsComplete: true,
	})
	completed := waitTracerouteStatus(t, sub, traceroute.StatusCompleted)
	if len(completed.ReturnRoute) != 2 {
		t.Fatalf("unexpected return route length: %d", len(completed.ReturnRoute))
	}
	if completed.DurationMS < 0 {
		t.Fatalf("unexpected duration: %d", completed.DurationMS)
	}
}

func TestTracerouteServiceTimesOutPendingRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	service := NewTracerouteService(
		messageBus,
		stubTracerouteSender{
			send: func(_ uint32, _ uint32) (string, error) {
				return "7", nil
			},
		},
		nil,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
		50*time.Millisecond,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	sub := messageBus.Subscribe(connectors.TopicTracerouteUpdate)
	defer messageBus.Unsubscribe(sub, connectors.TopicTracerouteUpdate)

	if _, err := service.StartTraceroute(context.Background(), TracerouteTarget{NodeID: "!00000007"}); err != nil {
		t.Fatalf("start traceroute: %v", err)
	}

	waitTracerouteStatus(t, sub, traceroute.StatusStarted)
	timedOut := waitTracerouteStatus(t, sub, traceroute.StatusTimedOut)
	if timedOut.RequestID != 7 {
		t.Fatalf("unexpected request id: %d", timedOut.RequestID)
	}
	if timedOut.Error == "" {
		t.Fatalf("expected timeout error text")
	}
}

func waitTracerouteStatus(t *testing.T, sub bus.Subscription, status traceroute.Status) connectors.TracerouteUpdate {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for traceroute status %q", status)
		case raw, ok := <-sub:
			if !ok {
				t.Fatalf("traceroute update subscription closed")
			}
			update, ok := raw.(connectors.TracerouteUpdate)
			if !ok {
				continue
			}
			if update.Status == status {
				return update
			}
		}
	}
}
