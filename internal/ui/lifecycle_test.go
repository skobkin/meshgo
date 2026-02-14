package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestStartNotificationServiceRegistersLifecycleHooks(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	lifecycle := &lifecycleSpy{}
	app := &lifecycleAppSpy{
		App:       base,
		lifecycle: lifecycle,
	}
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Bus:           nil,
			ChatStore:     domain.NewChatStore(),
			NodeStore:     domain.NewNodeStore(),
			CurrentConfig: config.Default,
		},
	}

	stop := startNotificationService(dep, app, true)
	if stop == nil {
		t.Fatalf("expected notification stop function")
	}
	if lifecycle.onEnteredForeground == nil {
		t.Fatalf("expected on-entered-foreground hook to be registered")
	}
	if lifecycle.onExitedForeground == nil {
		t.Fatalf("expected on-exited-foreground hook to be registered")
	}

	lifecycle.onEnteredForeground()
	lifecycle.onExitedForeground()
	stop()
	stop()
}
