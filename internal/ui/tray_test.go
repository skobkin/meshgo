package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestConfigureSystemTrayDesktopApp(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	app := &trayAppSpy{App: base}
	window := &windowSpy{Window: base.NewWindow("tray")}
	var quitCalls int

	setTrayIcon := configureSystemTray(app, window, theme.VariantLight, func() {
		quitCalls++
	})
	if setTrayIcon == nil {
		t.Fatalf("expected tray icon setter")
	}
	if app.trayIcon == nil {
		t.Fatalf("expected initial tray icon to be set")
	}
	if app.trayMenu == nil {
		t.Fatalf("expected tray menu to be configured")
	}
	if len(app.trayMenu.Items) != 2 {
		t.Fatalf("expected two tray menu items, got %d", len(app.trayMenu.Items))
	}

	setTrayIcon(theme.VariantDark)
	if app.trayIcon == nil {
		t.Fatalf("expected tray icon after theme change")
	}

	app.trayMenu.Items[0].Action()
	if window.showCalls != 1 {
		t.Fatalf("expected show action to show window once, got %d", window.showCalls)
	}
	if window.focusCalls != 1 {
		t.Fatalf("expected show action to request focus once, got %d", window.focusCalls)
	}

	app.trayMenu.Items[1].Action()
	if quitCalls != 1 {
		t.Fatalf("expected quit action callback once, got %d", quitCalls)
	}
}

func TestConfigureSystemTrayNonDesktopAppReturnsNoopSetter(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	app := &basicAppWrapper{App: base}
	window := base.NewWindow("tray")
	setTrayIcon := configureSystemTray(app, window, theme.VariantLight, nil)
	if setTrayIcon == nil {
		t.Fatalf("expected non-nil setter for non-desktop app")
	}

	setTrayIcon(theme.VariantDark)
}
