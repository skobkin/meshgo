package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func TestNodeDiscoveryProjection_EmitsAfterBootstrapForUnknownNodeInfoPacket(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	t.Cleanup(messageBus.Close)

	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!00000001"})

	proj := NewNodeDiscoveryProjection(store, logger)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	proj.Start(ctx, messageBus)

	sub := messageBus.Subscribe(connectors.TopicNodeDiscovered)
	t.Cleanup(func() {
		messageBus.Unsubscribe(sub, connectors.TopicNodeDiscovered)
	})

	// Before bootstrap completion, discovery must stay muted.
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000099",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	assertNoNodeDiscovered(t, sub)

	armBootstrap(messageBus)

	// Non-nodeinfo packets must not trigger discovery.
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000098",
		},
		Type: domain.NodeUpdateTypeTelemetryPacket,
	})
	assertNoNodeDiscovered(t, sub)

	// Already-known startup node should not emit.
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000001",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	assertNoNodeDiscovered(t, sub)

	// Unknown node discovered post-bootstrap emits once.
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID:    "!00000099",
			ShortName: "N99",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	event := waitNodeDiscovered(t, sub)
	if event.NodeID != "!00000099" {
		t.Fatalf("unexpected discovered node id: %q", event.NodeID)
	}
	if event.Source != nodeDiscoverySourceNodeInfoPacket {
		t.Fatalf("unexpected discovery source: %q", event.Source)
	}

	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000099",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	assertNoNodeDiscovered(t, sub)
}

func TestNodeDiscoveryProjection_ResetFromStoreClearsSessionState(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	t.Cleanup(messageBus.Close)

	store := domain.NewNodeStore()
	proj := NewNodeDiscoveryProjection(store, logger)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	proj.Start(ctx, messageBus)

	sub := messageBus.Subscribe(connectors.TopicNodeDiscovered)
	t.Cleanup(func() {
		messageBus.Unsubscribe(sub, connectors.TopicNodeDiscovered)
	})

	armBootstrap(messageBus)
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000042",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	_ = waitNodeDiscovered(t, sub)

	// Reset discovery state to simulate DB/store reset on transport change.
	store.Reset()
	proj.ResetFromStore(store)

	// Must remain muted again until bootstrap is ready.
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000042",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	assertNoNodeDiscovered(t, sub)

	// After re-arming bootstrap, the same node can be discovered again.
	armBootstrap(messageBus)
	messageBus.Publish(connectors.TopicNodeInfo, domain.NodeUpdate{
		Node: domain.Node{
			NodeID: "!00000042",
		},
		Type: domain.NodeUpdateTypeNodeInfoPacket,
	})
	event := waitNodeDiscovered(t, sub)
	if event.NodeID != "!00000042" {
		t.Fatalf("unexpected discovered node id after reset: %q", event.NodeID)
	}
}

func waitNodeDiscovered(t *testing.T, sub bus.Subscription) domain.NodeDiscovered {
	t.Helper()
	timeout := time.NewTimer(500 * time.Millisecond)
	defer timeout.Stop()

	for {
		select {
		case raw, ok := <-sub:
			if !ok {
				t.Fatalf("node discovery subscription closed")
			}
			event, ok := raw.(domain.NodeDiscovered)
			if !ok {
				continue
			}

			return event
		case <-timeout.C:
			t.Fatalf("timeout waiting for node discovery event")
		}
	}
}

func assertNoNodeDiscovered(t *testing.T, sub bus.Subscription) {
	t.Helper()
	timer := time.NewTimer(120 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case raw, ok := <-sub:
			if !ok {
				t.Fatalf("node discovery subscription closed")
			}
			if _, ok := raw.(domain.NodeDiscovered); ok {
				t.Fatalf("unexpected node discovery event: %#v", raw)
			}
		case <-timer.C:
			return
		}
	}
}

func armBootstrap(messageBus bus.MessageBus) {
	messageBus.Publish(connectors.TopicRadioFrom, radio.DecodedFrame{WantConfigReady: true})
	time.Sleep(20 * time.Millisecond)
}
