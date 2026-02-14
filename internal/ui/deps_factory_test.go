package ui

import (
	"testing"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func TestBuildRuntimeDependencies_MapsRuntimeAndLaunch(t *testing.T) {
	cfg := config.Default()
	cfg.UI.LastSelectedChat = "chat-1"

	rt := &meshapp.Runtime{
		Core: meshapp.RuntimeCore{
			Config: cfg,
			Paths: meshapp.Paths{
				RootDir:     "/tmp/meshgo",
				ConfigFile:  "/tmp/meshgo/config.json",
				DBFile:      "/tmp/meshgo/app.db",
				LogFile:     "/tmp/meshgo/app.log",
				CacheDir:    "/tmp/meshgo-cache",
				MapTilesDir: "/tmp/meshgo-cache/tiles",
			},
		},
		Domain: meshapp.RuntimeDomain{
			ChatStore: domain.NewChatStore(),
			NodeStore: domain.NewNodeStore(),
		},
		Connectivity: meshapp.RuntimeConnectivity{
			Radio:      &radio.Service{},
			Traceroute: &meshapp.TracerouteService{},
		},
	}

	quitCalled := false
	dep := BuildRuntimeDependencies(rt, LaunchOptions{StartHidden: true}, func() {
		quitCalled = true
	})

	if dep.Data.Config.UI.LastSelectedChat != "chat-1" {
		t.Fatalf("expected last selected chat to be mapped")
	}
	if dep.Data.ChatStore != rt.Domain.ChatStore {
		t.Fatalf("expected chat store to be mapped")
	}
	if dep.Data.Paths.MapTilesDir != "/tmp/meshgo-cache/tiles" {
		t.Fatalf("expected paths to be mapped")
	}
	if dep.Data.NodeStore != rt.Domain.NodeStore {
		t.Fatalf("expected node store to be mapped")
	}
	if dep.Data.LastSelectedChat != "chat-1" {
		t.Fatalf("expected data last selected chat to be mapped")
	}
	if dep.Data.CurrentConnStatus == nil {
		t.Fatalf("expected current conn status provider to be mapped")
	}
	if dep.Data.CurrentConfig == nil {
		t.Fatalf("expected current config provider to be mapped")
	}
	if dep.Data.LocalNodeID == nil {
		t.Fatalf("expected local node id provider to be mapped")
	}
	if got := dep.Data.LocalNodeID(); got != "" {
		t.Fatalf("expected empty local node id for zero radio service, got %q", got)
	}
	if dep.Actions.Sender == nil {
		t.Fatalf("expected sender to be mapped")
	}
	if dep.Actions.Traceroute == nil {
		t.Fatalf("expected traceroute action to be mapped")
	}
	if dep.Actions.OnSave == nil {
		t.Fatalf("expected save action to be mapped")
	}
	if dep.Actions.OnChatSelected == nil {
		t.Fatalf("expected chat selected action to be mapped")
	}
	if dep.Actions.OnMapViewportChanged == nil {
		t.Fatalf("expected map viewport action to be mapped")
	}
	if dep.Actions.OnClearDB == nil {
		t.Fatalf("expected clear db action to be mapped")
	}
	if dep.Actions.OnClearCache == nil {
		t.Fatalf("expected clear cache action to be mapped")
	}
	if dep.Actions.OnStartUpdateChecker == nil {
		t.Fatalf("expected update checker start action to be mapped")
	}
	if dep.Platform.BluetoothScanner == nil {
		t.Fatalf("expected bluetooth scanner to be initialized")
	}
	if dep.Platform.OpenBluetoothSettings == nil {
		t.Fatalf("expected bluetooth settings opener to be initialized")
	}
	if !dep.Launch.StartHidden {
		t.Fatalf("expected launch options to be mapped")
	}
	if dep.Actions.OnQuit == nil {
		t.Fatalf("expected quit action to be mapped")
	}

	dep.Actions.OnQuit()
	if !quitCalled {
		t.Fatalf("expected quit callback to be invoked")
	}
}

func TestBuildRuntimeDependencies_NilRuntimeStillMapsLaunchAndQuit(t *testing.T) {
	quitCalled := false
	dep := BuildRuntimeDependencies(nil, LaunchOptions{StartHidden: true}, func() {
		quitCalled = true
	})

	if !dep.Launch.StartHidden {
		t.Fatalf("expected launch options to be preserved")
	}
	if dep.Actions.OnQuit == nil {
		t.Fatalf("expected quit callback to be preserved")
	}
	if dep.Actions.Sender != nil {
		t.Fatalf("expected sender to stay nil for nil runtime")
	}
	if dep.Actions.Traceroute != nil {
		t.Fatalf("expected traceroute action to stay nil for nil runtime")
	}
	if dep.Actions.OnMapViewportChanged != nil {
		t.Fatalf("expected map viewport action to stay nil for nil runtime")
	}
	if dep.Actions.OnClearCache != nil {
		t.Fatalf("expected clear cache action to stay nil for nil runtime")
	}
	if dep.Actions.OnStartUpdateChecker != nil {
		t.Fatalf("expected update checker start action to stay nil for nil runtime")
	}
	if dep.Data.CurrentConnStatus != nil {
		t.Fatalf("expected status provider to stay nil for nil runtime")
	}
	if dep.Data.CurrentConfig != nil {
		t.Fatalf("expected current config provider to stay nil for nil runtime")
	}
	if dep.Platform.BluetoothScanner != nil {
		t.Fatalf("expected scanner to stay nil for nil runtime")
	}
	if dep.Platform.OpenBluetoothSettings == nil {
		t.Fatalf("expected bluetooth settings opener to be initialized for nil runtime")
	}

	dep.Actions.OnQuit()
	if !quitCalled {
		t.Fatalf("expected quit callback invocation")
	}
}
