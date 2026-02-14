package ui

import (
	"testing"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
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

func TestCurrentUpdateSnapshot(t *testing.T) {
	expected := meshapp.UpdateSnapshot{
		CurrentVersion:  "0.6.0",
		UpdateAvailable: true,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	}
	dep := RuntimeDependencies{
		Data: DataDependencies{
			CurrentUpdateSnapshot: func() (meshapp.UpdateSnapshot, bool) {
				return expected, true
			},
		},
	}

	got, ok := currentUpdateSnapshot(dep)
	if !ok {
		t.Fatalf("expected snapshot to be present")
	}
	if got.Latest.Version != "0.7.0" {
		t.Fatalf("expected latest version 0.7.0, got %q", got.Latest.Version)
	}
}
