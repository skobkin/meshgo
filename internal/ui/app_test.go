package ui

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
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
	if got := resolve("!33333333"); got != "!33333333" {
		t.Fatalf("expected node id fallback for id-only node, got %q", got)
	}
}

func TestFormatWindowTitle(t *testing.T) {
	got := formatWindowTitle(connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
	}, "")
	want := "MeshGo - IP connected"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatConnStatus_WithTargetAndLocalNodeName(t *testing.T) {
	got := formatConnStatus(connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "serial",
		Target:        "/dev/ttyACM2",
	}, "ABCD")
	want := "Serial connected (/dev/ttyACM2, ABCD)"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestTransportDisplayName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ip", in: "ip", want: "IP"},
		{name: "serial", in: "serial", want: "Serial"},
		{name: "bluetooth", in: "bluetooth", want: "Bluetooth LE (unstable)"},
		{name: "fallback", in: "custom", want: "custom"},
		{name: "empty", in: " ", want: ""},
	}

	for _, tt := range tests {
		got := transportDisplayName(tt.in)
		if got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}

func TestInitialConnStatusBluetooth(t *testing.T) {
	status := initialConnStatus(RuntimeDependencies{
		Data: DataDependencies{
			Config: config.AppConfig{
				Connection: config.ConnectionConfig{
					Connector:        config.ConnectorBluetooth,
					BluetoothAddress: "AA:BB:CC:DD:EE:FF",
				},
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

func TestResolveInitialConnStatus_UsesCachedStatus(t *testing.T) {
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config: config.AppConfig{
				Connection: config.ConnectionConfig{
					Connector:        config.ConnectorSerial,
					SerialPort:       "/dev/ttyACM0",
					SerialBaud:       115200,
					Host:             "",
					BluetoothAddress: "",
				},
			},
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{
					State:         connectors.ConnectionStateConnected,
					TransportName: "serial",
					Target:        "",
				}, true
			},
		},
	}

	status := resolveInitialConnStatus(dep)
	if status.State != connectors.ConnectionStateConnected {
		t.Fatalf("expected cached connected status, got %q", status.State)
	}
	if status.TransportName != "serial" {
		t.Fatalf("expected serial transport, got %q", status.TransportName)
	}
	if status.Target != "/dev/ttyACM0" {
		t.Fatalf("expected serial target fallback, got %q", status.Target)
	}
}

func TestSidebarStatusIcon(t *testing.T) {
	connected := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateConnected,
	})
	if connected != resources.UIIconConnected {
		t.Fatalf("expected connected icon, got %q", connected)
	}

	disconnected := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateDisconnected,
	})
	if disconnected != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon, got %q", disconnected)
	}

	connecting := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateConnecting,
	})
	if connecting != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon for connecting state, got %q", connecting)
	}
}

func TestLocalNodeDisplayName(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!11111111", ShortName: "ABCD", LongName: "Alpha Bravo"})
	store.Upsert(domain.Node{NodeID: "!22222222", ShortName: "EFGH"})
	store.Upsert(domain.Node{NodeID: "!33333333"})

	if got := localNodeDisplayName(func() string { return "!11111111" }, store); got != "Alpha Bravo" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!22222222" }, store); got != "EFGH" {
		t.Fatalf("expected short name fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!33333333" }, store); got != "!33333333" {
		t.Fatalf("expected node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!44444444" }, store); got != "!44444444" {
		t.Fatalf("expected unknown node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "" }, store); got != "" {
		t.Fatalf("expected empty for empty node id, got %q", got)
	}
	if got := localNodeDisplayName(nil, store); got != "" {
		t.Fatalf("expected empty for nil localNodeID, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!11111111" }, nil); got != "!11111111" {
		t.Fatalf("expected node id fallback for nil store, got %q", got)
	}
}

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
