package ui

import (
	"testing"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func TestNewDependenciesFromRuntime_MapsRuntimeAndLaunch(t *testing.T) {
	cfg := config.Default()
	cfg.UI.LastSelectedChat = "chat-1"

	rt := &meshapp.Runtime{
		Config:    cfg,
		ChatStore: domain.NewChatStore(),
		NodeStore: domain.NewNodeStore(),
		Radio:     &radio.Service{},
	}

	quitCalled := false
	dep := NewDependenciesFromRuntime(rt, LaunchOptions{StartHidden: true}, func() {
		quitCalled = true
	})

	if dep.Data.Config.UI.LastSelectedChat != "chat-1" {
		t.Fatalf("expected last selected chat to be mapped")
	}
	if dep.Data.ChatStore != rt.ChatStore {
		t.Fatalf("expected chat store to be mapped")
	}
	if dep.Data.NodeStore != rt.NodeStore {
		t.Fatalf("expected node store to be mapped")
	}
	if dep.Data.LastSelectedChat != "chat-1" {
		t.Fatalf("expected data last selected chat to be mapped")
	}
	if dep.Data.CurrentConnStatus == nil {
		t.Fatalf("expected current conn status provider to be mapped")
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
	if dep.Actions.OnSave == nil {
		t.Fatalf("expected save action to be mapped")
	}
	if dep.Actions.OnChatSelected == nil {
		t.Fatalf("expected chat selected action to be mapped")
	}
	if dep.Actions.OnClearDB == nil {
		t.Fatalf("expected clear db action to be mapped")
	}
	if dep.Platform.BluetoothScanner == nil {
		t.Fatalf("expected bluetooth scanner to be initialized")
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

func TestNewDependenciesFromRuntime_NilRuntimeStillMapsLaunchAndQuit(t *testing.T) {
	quitCalled := false
	dep := NewDependenciesFromRuntime(nil, LaunchOptions{StartHidden: true}, func() {
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
	if dep.Data.CurrentConnStatus != nil {
		t.Fatalf("expected status provider to stay nil for nil runtime")
	}
	if dep.Platform.BluetoothScanner != nil {
		t.Fatalf("expected scanner to stay nil for nil runtime")
	}

	dep.Actions.OnQuit()
	if !quitCalled {
		t.Fatalf("expected quit callback invocation")
	}
}
