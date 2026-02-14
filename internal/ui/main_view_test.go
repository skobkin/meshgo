package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestBuildMainViewBuildsSidebarAndPresentationComponents(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	window := app.NewWindow("main")
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config:    config.Default(),
			Paths:     meshapp.Paths{MapTilesDir: t.TempDir()},
			ChatStore: domain.NewChatStore(),
			NodeStore: domain.NewNodeStore(),
			LocalNodeID: func() string {
				return ""
			},
		},
	}

	view := buildMainView(
		dep,
		app,
		window,
		app.Settings().ThemeVariant(),
		connectors.ConnectionStatus{
			State:         connectors.ConnectionStateConnecting,
			TransportName: "ip",
		},
	)

	if view.left == nil || view.rightStack == nil {
		t.Fatalf("expected main view containers to be initialized")
	}
	if view.updateIndicator == nil {
		t.Fatalf("expected update indicator to be initialized")
	}
	if view.connStatusPresenter == nil {
		t.Fatalf("expected connection status presenter to be initialized")
	}
	if view.applyMapTheme == nil {
		t.Fatalf("expected map theme callback to be initialized")
	}

	view.applyMapTheme(app.Settings().ThemeVariant())
}
