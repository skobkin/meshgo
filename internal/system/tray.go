//go:build !no_systray

package system

import (
	"log/slog"
)

type SystemTray struct {
	logger              *slog.Logger
	hasUnread           bool
	notificationsEnabled bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
	
	// Fallback tray (no actual system tray)
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
	st.logger.Debug("Fallback tray unread status", "hasUnread", hasUnread)
}

func (st *SystemTray) SetWindowVisible(visible bool) {
	st.logger.Debug("Fallback tray window visibility", "visible", visible)
	// No-op in fallback implementation
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
	st.logger.Info("Fallback system tray - no actual tray available")
	// Do nothing - this is a fallback when Fyne tray is not available
}

func (st *SystemTray) Quit() {
	st.logger.Debug("Fallback system tray quit")
	// Do nothing - no tray to quit
}