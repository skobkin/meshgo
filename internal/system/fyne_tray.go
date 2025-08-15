//go:build !no_fyne

package system

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

type FyneSystemTray struct {
	logger              *slog.Logger
	app                 fyne.App
	hasUnread           bool
	notificationsEnabled bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
	
	// Desktop integration
	desk    desktop.App
}

func NewFyneSystemTray(logger *slog.Logger, app fyne.App) *FyneSystemTray {
	tray := &FyneSystemTray{
		logger:              logger,
		app:                 app,
		notificationsEnabled: true,
	}
	
	// Icon is already set by the main app
	
	// Try to get desktop app interface
	if desk, ok := app.(desktop.App); ok {
		tray.desk = desk
		tray.setupSystemTray()
	} else {
		logger.Warn("Desktop integration not available - system tray disabled")
	}
	
	return tray
}

func (st *FyneSystemTray) setupSystemTray() {
	if st.desk == nil {
		return
	}
	
	// Set up system tray menu
	menu := fyne.NewMenu("MeshGo",
		fyne.NewMenuItem("Show/Hide", func() {
			if st.onShowHide != nil {
				st.onShowHide()
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Notifications", func() {
			st.notificationsEnabled = !st.notificationsEnabled
			if st.onToggleNotifs != nil {
				st.onToggleNotifs(st.notificationsEnabled)
			}
			st.updateMenu()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			if st.onExit != nil {
				st.onExit()
			}
		}),
	)
	
	st.desk.SetSystemTrayMenu(menu)
	st.updateMenu()
	
	st.logger.Info("Fyne system tray initialized")
}

func (st *FyneSystemTray) updateMenu() {
	if st.desk == nil {
		return
	}
	
	// Update notifications menu item text
	menu := fyne.NewMenu("MeshGo",
		fyne.NewMenuItem("Show/Hide", func() {
			if st.onShowHide != nil {
				st.onShowHide()
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(st.getNotificationMenuText(), func() {
			st.notificationsEnabled = !st.notificationsEnabled
			if st.onToggleNotifs != nil {
				st.onToggleNotifs(st.notificationsEnabled)
			}
			st.updateMenu()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Exit", func() {
			if st.onExit != nil {
				st.onExit()
			}
		}),
	)
	
	st.desk.SetSystemTrayMenu(menu)
}

func (st *FyneSystemTray) getNotificationMenuText() string {
	if st.notificationsEnabled {
		return "✓ Notifications"
	}
	return "✗ Notifications"
}

func (st *FyneSystemTray) SetUnread(hasUnread bool) {
	if st.hasUnread == hasUnread {
		return
	}
	
	st.hasUnread = hasUnread
	st.logger.Debug("Tray unread status", "hasUnread", hasUnread)
	
	// Update system tray icon or appearance if possible
	// Note: Fyne's system tray API is more limited than standalone libraries
}

func (st *FyneSystemTray) OnShowHide(fn func()) {
	st.onShowHide = fn
}

func (st *FyneSystemTray) OnToggleNotifications(fn func(bool)) {
	st.onToggleNotifs = fn
}

func (st *FyneSystemTray) OnExit(fn func()) {
	st.onExit = fn
}

func (st *FyneSystemTray) Run() {
	// Fyne tray is integrated with the app, no separate run needed
	st.logger.Debug("Fyne system tray running with app")
}

func (st *FyneSystemTray) Quit() {
	st.logger.Debug("Fyne system tray quit requested")
	if st.app != nil {
		st.app.Quit()
	}
}