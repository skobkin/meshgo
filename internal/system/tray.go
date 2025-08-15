//go:build !no_systray

package system

import (
	"log/slog"

	"github.com/getlantern/systray"
)

type SystemTray struct {
	logger              *slog.Logger
	hasUnread           bool
	notificationsEnabled bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
	
	// Menu items
	menuShow            *systray.MenuItem
	menuNotifications   *systray.MenuItem
	menuSeparator       *systray.MenuItem
	menuExit            *systray.MenuItem
}

func NewSystemTray(logger *slog.Logger) *SystemTray {
	return &SystemTray{
		logger:              logger,
		notificationsEnabled: true,
	}
}

func (st *SystemTray) SetUnread(hasUnread bool) {
	if st.hasUnread == hasUnread {
		return
	}
	
	st.hasUnread = hasUnread
	st.updateIcon()
	st.logger.Debug("Tray unread status", "hasUnread", hasUnread)
}

func (st *SystemTray) OnShowHide(fn func()) {
	st.onShowHide = fn
}

func (st *SystemTray) OnToggleNotifications(fn func(bool)) {
	st.onToggleNotifs = fn
}

func (st *SystemTray) OnExit(fn func()) {
	st.onExit = fn
}

func (st *SystemTray) Run() {
	systray.Run(st.onReady, st.onExit)
}

func (st *SystemTray) Quit() {
	systray.Quit()
}

func (st *SystemTray) onReady() {
	st.logger.Info("System tray initialized")
	
	// Set initial icon
	st.updateIcon()
	systray.SetTitle("MeshGo")
	systray.SetTooltip("MeshGo - Meshtastic GUI")

	// Create menu items
	st.menuShow = systray.AddMenuItem("Show/Hide", "Show or hide the main window")
	st.menuSeparator = systray.AddSeparator()
	st.menuNotifications = systray.AddMenuItem("✓ Notifications", "Toggle notifications")
	systray.AddSeparator()
	st.menuExit = systray.AddMenuItem("Exit", "Exit MeshGo")

	// Handle menu clicks
	go st.handleMenuClicks()
}

func (st *SystemTray) handleMenuClicks() {
	for {
		select {
		case <-st.menuShow.ClickedCh:
			st.logger.Debug("Show/Hide clicked")
			if st.onShowHide != nil {
				st.onShowHide()
			}

		case <-st.menuNotifications.ClickedCh:
			st.notificationsEnabled = !st.notificationsEnabled
			st.updateNotificationsMenuItem()
			st.logger.Info("Notifications toggled", "enabled", st.notificationsEnabled)
			if st.onToggleNotifs != nil {
				st.onToggleNotifs(st.notificationsEnabled)
			}

		case <-st.menuExit.ClickedCh:
			st.logger.Info("Exit clicked")
			if st.onExit != nil {
				st.onExit()
			}
			return
		}
	}
}

func (st *SystemTray) updateIcon() {
	if st.hasUnread {
		// Icon with unread indicator (red dot overlay)
		systray.SetIcon(getUnreadIcon())
		systray.SetTooltip("MeshGo - New messages")
	} else {
		// Normal icon
		systray.SetIcon(getNormalIcon())
		systray.SetTooltip("MeshGo - Meshtastic GUI")
	}
}

func (st *SystemTray) updateNotificationsMenuItem() {
	if st.notificationsEnabled {
		st.menuNotifications.SetTitle("✓ Notifications")
	} else {
		st.menuNotifications.SetTitle("✗ Notifications")
	}
}

// Simple icons as byte arrays - in production would use proper icon files
func getNormalIcon() []byte {
	// This would be a proper icon file in production
	// For now, using a simple placeholder
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		// ... rest of icon data would go here
		// For simplicity, using minimal PNG data
	}
}

func getUnreadIcon() []byte {
	// This would be the icon with unread indicator overlay
	// For now, same as normal icon
	return getNormalIcon()
}

// Alternative implementation using embedded icons
// In production, you would embed actual icon files:

//go:embed icons/normal.png
var normalIconData []byte

//go:embed icons/unread.png  
var unreadIconData []byte

func getNormalIconEmbed() []byte {
	return normalIconData
}

func getUnreadIconEmbed() []byte {
	return unreadIconData
}