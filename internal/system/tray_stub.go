//go:build no_systray

package system

import (
	"log/slog"
)

// Stub implementation when systray is disabled
type SystemTray struct {
	logger              *slog.Logger
	hasUnread           bool
	notificationsEnabled bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
}

func NewSystemTray(logger *slog.Logger) *SystemTray {
	logger.Info("System tray disabled (built with no_systray tag)")
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
	st.logger.Debug("Tray unread status (stub)", "hasUnread", hasUnread)
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
	st.logger.Info("System tray stub - no GUI tray available")
	// Do nothing - tray functionality is disabled
}

func (st *SystemTray) Quit() {
	// Do nothing - no tray to quit
}