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
	windowVisible       bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
	shuttingDown        bool
	
	// Desktop integration
	desk    desktop.App
}

func NewFyneSystemTray(logger *slog.Logger, app fyne.App) *FyneSystemTray {
	tray := &FyneSystemTray{
		logger:              logger,
		app:                 app,
		notificationsEnabled: true,
		windowVisible:       true, // Window starts visible
	}
	
	// Icon is already set by the main app
	
	// Note: We don't use lifecycle hooks as they can cause circular shutdown calls
	// The system tray quit will be handled by the application's main quit sequence
	
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
	
	// Create a proper system tray icon
	st.logger.Debug("Setting up system tray with icon...")
	
	// Create a simple monochrome icon that works well in system tray
	iconResource := fyne.NewStaticResource("meshgo_tray", []byte{
		// 16x16 monochrome PNG (simple circle)
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x37, 0x6E, 0xF9, 0x24, 0x00, 0x00, 0x00,
		0x02, 0x50, 0x4C, 0x54, 0x45, 0x00, 0x00, 0x00, 0x55, 0xC2, 0xD3, 0x7E,
		0x00, 0x00, 0x00, 0x28, 0x49, 0x44, 0x41, 0x54, 0x08, 0x1D, 0x01, 0x1D,
		0x00, 0xE2, 0xFF, 0x00, 0x00, 0x03, 0xC0, 0x0F, 0xF0, 0x1F, 0xF8, 0x3F,
		0xFC, 0x3F, 0xFC, 0x7F, 0xFE, 0x7F, 0xFE, 0x7F, 0xFE, 0x7F, 0xFE, 0x3F,
		0xFC, 0x3F, 0xFC, 0x1F, 0xF8, 0x0F, 0xF0, 0x03, 0xC0, 0x00, 0x00, 0x37,
		0x7A, 0x19, 0x48, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	})
	
	// Try to set the system tray icon
	if iconSetter, ok := st.desk.(interface{ SetSystemTrayIcon(fyne.Resource) }); ok {
		st.logger.Debug("System tray supports icon setting")
		iconSetter.SetSystemTrayIcon(iconResource)
		st.logger.Debug("System tray icon set")
	}
	
	// Create initial menu
	st.updateMenu()
	
	st.logger.Info("Fyne system tray initialized")
}

func (st *FyneSystemTray) updateMenu() {
	if st.desk == nil {
		return
	}
	
	// Create menu items manually to avoid any automatic additions
	showHideItem := fyne.NewMenuItem(st.getShowHideMenuText(), func() {
		st.logger.Info("Tray Show/Hide menu item clicked (from updateMenu)")
		if st.onShowHide != nil {
			st.onShowHide()
		} else {
			st.logger.Warn("No onShowHide callback set (from updateMenu)")
		}
	})
	
	notifItem := fyne.NewMenuItem(st.getNotificationMenuText(), func() {
		st.notificationsEnabled = !st.notificationsEnabled
		if st.onToggleNotifs != nil {
			st.onToggleNotifs(st.notificationsEnabled)
		}
		st.updateMenu()
	})
	
	// Create quit menu item that properly handles shutdown
	exitItem := fyne.NewMenuItem("Quit", func() {
		st.logger.Info("Exit menu item clicked")
		if !st.shuttingDown && st.onExit != nil {
			st.shuttingDown = true
			// Use a goroutine to avoid blocking the UI thread
			go st.onExit()
		}
	})
	
	// Create menu with our custom quit item
	menu := fyne.NewMenu("",
		showHideItem,
		fyne.NewMenuItemSeparator(),
		notifItem,
		fyne.NewMenuItemSeparator(),
		exitItem,
	)
	
	st.desk.SetSystemTrayMenu(menu)
}

func (st *FyneSystemTray) getShowHideMenuText() string {
	if st.windowVisible {
		return "🔼 Hide Window"
	}
	return "🔽 Show Window"
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

func (st *FyneSystemTray) SetWindowVisible(visible bool) {
	if st.windowVisible != visible {
		st.windowVisible = visible
		st.updateMenu() // Refresh menu to show correct Show/Hide text
	}
}

func (st *FyneSystemTray) Quit() {
	st.logger.Debug("Fyne system tray quit requested")
	st.shuttingDown = true
	// The actual quit will be handled by the main app shutdown sequence
}