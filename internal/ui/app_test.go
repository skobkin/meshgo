package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"

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

func TestRun_UsesInjectedAppFactoryAndCallsOnQuit(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	app := &appRunWindowSpy{App: base}
	previousFactory := newFyneApp
	newFyneApp = func() fyne.App {
		return app
	}
	t.Cleanup(func() {
		newFyneApp = previousFactory
	})

	var onQuitCalls int
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config:    config.Default(),
			Paths:     meshapp.Paths{MapTilesDir: t.TempDir()},
			ChatStore: domain.NewChatStore(),
			NodeStore: domain.NewNodeStore(),
		},
		Actions: ActionDependencies{
			OnQuit: func() {
				onQuitCalls++
			},
		},
	}

	if err := Run(dep); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if app.runCalls != 1 {
		t.Fatalf("expected app Run to be called once, got %d", app.runCalls)
	}
	if onQuitCalls != 1 {
		t.Fatalf("expected OnQuit to be called once, got %d", onQuitCalls)
	}
	if app.createdWindow == nil {
		t.Fatalf("expected UI window to be created")
	}
	if app.createdWindow.showCalls != 1 {
		t.Fatalf("expected main window to be shown once, got %d", app.createdWindow.showCalls)
	}
}

func TestRun_StartHiddenHidesWindow(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	app := &appRunWindowSpy{App: base}
	previousFactory := newFyneApp
	newFyneApp = func() fyne.App {
		return app
	}
	t.Cleanup(func() {
		newFyneApp = previousFactory
	})

	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config:    config.Default(),
			Paths:     meshapp.Paths{MapTilesDir: t.TempDir()},
			ChatStore: domain.NewChatStore(),
			NodeStore: domain.NewNodeStore(),
		},
		Launch: LaunchOptions{StartHidden: true},
	}

	if err := Run(dep); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if app.createdWindow == nil {
		t.Fatalf("expected UI window to be created")
	}
	if app.createdWindow.hideCalls < 1 {
		t.Fatalf("expected main window to be hidden for start-hidden mode")
	}
}
