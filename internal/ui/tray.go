package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/skobkin/meshgo/internal/resources"
)

func configureSystemTray(fyApp fyne.App, window fyne.Window, initialVariant fyne.ThemeVariant, quit func()) func(fyne.ThemeVariant) {
	setTrayIcon := func(_ fyne.ThemeVariant) {}

	desk, ok := fyApp.(desktop.App)
	if !ok {
		return setTrayIcon
	}

	setTrayIcon = func(variant fyne.ThemeVariant) {
		desk.SetSystemTrayIcon(resources.TrayIconResource(variant))
	}
	setTrayIcon(initialVariant)
	desk.SetSystemTrayMenu(fyne.NewMenu("meshgo",
		fyne.NewMenuItem("Show", func() {
			appLogger.Debug("system tray show action invoked")
			window.Show()
			window.RequestFocus()
		}),
		fyne.NewMenuItem("Quit", func() {
			appLogger.Debug("system tray quit action invoked")
			quit()
		}),
	))

	return setTrayIcon
}
