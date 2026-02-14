package ui

import (
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

var appLogger = slog.With("component", "ui.app")

func Run(dep RuntimeDependencies) error {
	fyApp := fyneapp.NewWithID("meshgo")
	initialVariant := fyApp.Settings().ThemeVariant()
	fyApp.SetIcon(resources.AppIconResource(initialVariant))
	appLogger.Info(
		"starting UI runtime",
		"start_hidden", dep.Launch.StartHidden,
		"initial_theme", initialVariant,
	)

	initialStatus := resolveInitialConnStatus(dep)

	initialUpdateSnapshot, initialUpdateSnapshotKnown := currentUpdateSnapshot(dep)

	window := fyApp.NewWindow("")
	window.Resize(fyne.NewSize(1000, 700))
	view := buildMainView(
		dep,
		fyApp,
		window,
		initialVariant,
		initialStatus,
		initialUpdateSnapshot,
		initialUpdateSnapshotKnown,
	)

	themeRuntime := newThemeRuntime(fyApp, view.sidebar, view.updateIndicator, view.applyMapTheme, view.connStatusPresenter)
	themeRuntime.BindSettings()

	stopNotifications := startNotificationService(dep, fyApp, dep.Launch.StartHidden)

	stopUIListeners, stopUpdateSnapshots := bindPresentationListeners(
		dep,
		fyApp,
		view.connStatusPresenter,
		view.updateIndicator,
		view.left.Refresh,
	)

	content := container.NewBorder(nil, nil, view.left, nil, view.rightStack)
	window.SetContent(content)

	uiRuntime := newUIRuntime(
		fyApp,
		window,
		stopNotifications,
		stopUIListeners,
		stopUpdateSnapshots,
		dep.Actions.OnQuit,
	)
	uiRuntime.BindCloseIntercept()

	setTrayIcon := configureSystemTray(fyApp, window, initialVariant, uiRuntime.Quit)
	themeRuntime.SetTrayIconSetter(setTrayIcon)
	themeRuntime.Apply(initialVariant)

	uiRuntime.Run(dep.Launch.StartHidden)

	return nil
}

func initialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
	return meshapp.ConnectionStatusFromConfig(dep.Data.Config.Connection)
}

func resolveInitialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
	fallback := initialConnStatus(dep)
	status, ok := currentConnStatus(dep)
	if !ok || status.State == "" {
		return fallback
	}
	if strings.TrimSpace(status.TransportName) == "" {
		status.TransportName = fallback.TransportName
	}
	if strings.TrimSpace(status.Target) == "" {
		status.Target = fallback.Target
	}

	return status
}

func currentConnStatus(dep RuntimeDependencies) (connectors.ConnectionStatus, bool) {
	if dep.Data.CurrentConnStatus == nil {
		return connectors.ConnectionStatus{}, false
	}

	return dep.Data.CurrentConnStatus()
}

func resolveNodeDisplayName(store *domain.NodeStore) func(string) string {
	if store == nil {
		return nil
	}

	return func(nodeID string) string {
		return domain.NodeDisplayNameByID(store, nodeID)
	}
}

func nodeChanges(store *domain.NodeStore) <-chan struct{} {
	if store == nil {
		return nil
	}

	return store.Changes()
}

func currentUpdateSnapshot(dep RuntimeDependencies) (meshapp.UpdateSnapshot, bool) {
	if dep.Data.CurrentUpdateSnapshot == nil {
		return meshapp.UpdateSnapshot{}, false
	}

	return dep.Data.CurrentUpdateSnapshot()
}
