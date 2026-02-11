package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

func TestResolveNodeDisplayName_Priority(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!11111111", LongName: "Long", ShortName: "Short"})
	store.Upsert(domain.Node{NodeID: "!22222222", ShortName: "ShortOnly"})
	store.Upsert(domain.Node{NodeID: "!33333333"})

	resolve := resolveNodeDisplayName(store)
	if resolve == nil {
		t.Fatalf("expected non-nil resolver")
	}

	if got := resolve("!11111111"); got != "Long" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := resolve("!22222222"); got != "ShortOnly" {
		t.Fatalf("expected short name, got %q", got)
	}
	if got := resolve("!33333333"); got != "" {
		t.Fatalf("expected empty fallback for id-only node, got %q", got)
	}
}

func TestFormatWindowTitle(t *testing.T) {
	got := formatWindowTitle(connectors.ConnStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
	})
	want := "MeshGo - connected via ip"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInitialConnStatusBluetooth(t *testing.T) {
	status := initialConnStatus(Dependencies{
		Config: config.AppConfig{
			Connection: config.ConnectionConfig{
				Connector:        config.ConnectorBluetooth,
				BluetoothAddress: "AA:BB:CC:DD:EE:FF",
			},
		},
	})

	if status.TransportName != "bluetooth" {
		t.Fatalf("expected bluetooth transport, got %q", status.TransportName)
	}
	if status.State != connectors.ConnectionStateConnecting {
		t.Fatalf("expected connecting state, got %q", status.State)
	}
}

func TestSidebarStatusIcon(t *testing.T) {
	connected := sidebarStatusIcon(connectors.ConnStatus{
		State: connectors.ConnectionStateConnected,
	})
	if connected != resources.UIIconConnected {
		t.Fatalf("expected connected icon, got %q", connected)
	}

	disconnected := sidebarStatusIcon(connectors.ConnStatus{
		State: connectors.ConnectionStateDisconnected,
	})
	if disconnected != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon, got %q", disconnected)
	}

	connecting := sidebarStatusIcon(connectors.ConnStatus{
		State: connectors.ConnectionStateConnecting,
	})
	if connecting != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon for connecting state, got %q", connecting)
	}
}
