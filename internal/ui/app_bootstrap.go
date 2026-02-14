package ui

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"

	"github.com/skobkin/meshgo/internal/resources"
)

var newFyneApp = func() fyne.App {
	return fyneapp.NewWithID("meshgo")
}

func runWithApp(dep RuntimeDependencies, fyApp fyne.App) error {
	initialVariant := fyApp.Settings().ThemeVariant()
	fyApp.SetIcon(resources.AppIconResource(initialVariant))
	appLogger.Info(
		"starting UI runtime",
		"start_hidden", dep.Launch.StartHidden,
		"initial_theme", initialVariant,
	)

	initialStatus := resolveInitialConnStatus(dep)

	window := fyApp.NewWindow("")
	window.Resize(fyne.NewSize(1000, 700))
	view := buildMainView(
		dep,
		fyApp,
		window,
		initialVariant,
		initialStatus,
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
	// Start checks only after listeners are attached so the first snapshot is not missed.
	if dep.Actions.OnStartUpdateChecker != nil {
		dep.Actions.OnStartUpdateChecker()
	}

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
