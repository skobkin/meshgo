package ui

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestStartUIEventListenersStopPreventsFurtherCallbacks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	var connEvents atomic.Int64
	var nodeEvents atomic.Int64
	stop := startUIEventListeners(
		messageBus,
		func(_ connectors.ConnectionStatus) {
			connEvents.Add(1)
		},
		func() {
			nodeEvents.Add(1)
		},
	)

	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{State: connectors.ConnectionStateConnected})
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{})

	waitForCondition(t, func() bool {
		return connEvents.Load() == 1 && nodeEvents.Load() == 1
	})

	stop()

	connBefore := connEvents.Load()
	nodeBefore := nodeEvents.Load()
	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{State: connectors.ConnectionStateDisconnected})
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{})
	time.Sleep(100 * time.Millisecond)

	if connEvents.Load() != connBefore {
		t.Fatalf("expected no new connection callbacks after stop: before=%d after=%d", connBefore, connEvents.Load())
	}
	if nodeEvents.Load() != nodeBefore {
		t.Fatalf("expected no new node callbacks after stop: before=%d after=%d", nodeBefore, nodeEvents.Load())
	}
}

func TestStartUIEventListenersNilBusReturnsNoopStop(t *testing.T) {
	stop := startUIEventListeners(nil, nil, nil)
	stop()
	stop()
}

func TestStartUpdateSnapshotListenerStopPreventsFurtherCallbacks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	var calls atomic.Int64
	stop := startUpdateSnapshotListener(messageBus, func(_ meshapp.UpdateSnapshot) {
		calls.Add(1)
	})

	messageBus.Publish(connectors.TopicUpdateSnapshot, meshapp.UpdateSnapshot{CurrentVersion: "0.6.0"})
	waitForCondition(t, func() bool {
		return calls.Load() == 1
	})

	stop()

	before := calls.Load()
	messageBus.Publish(connectors.TopicUpdateSnapshot, meshapp.UpdateSnapshot{CurrentVersion: "0.7.0"})
	time.Sleep(100 * time.Millisecond)

	if calls.Load() != before {
		t.Fatalf("expected no new update callbacks after stop: before=%d after=%d", before, calls.Load())
	}
}
