//go:build no_fyne

package system

import (
	"log/slog"

	"fyne.io/fyne/v2"
)

type FyneSystemTray struct {
	logger              *slog.Logger
	hasUnread           bool
	notificationsEnabled bool
	onShowHide          func()
	onToggleNotifs      func(bool)
	onExit              func()
}

func NewFyneSystemTray(logger *slog.Logger, app fyne.App) *FyneSystemTray {
	logger.Info("Fyne system tray disabled (built with no_fyne tag)")
	return &FyneSystemTray{
		logger:              logger,
		notificationsEnabled: true,
	}
}

func (st *FyneSystemTray) SetUnread(hasUnread bool) {
	if st.hasUnread == hasUnread {
		return
	}
	
	st.hasUnread = hasUnread
	st.logger.Debug("Fyne tray unread status (stub)", "hasUnread", hasUnread)
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
	st.logger.Info("Fyne system tray stub - no tray available")
	// Block forever to keep app running (signals will trigger shutdown)
	select {}
}

func (st *FyneSystemTray) Quit() {
	// Do nothing - no tray to quit
}