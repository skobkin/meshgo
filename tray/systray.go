package tray

import (
	"encoding/base64"

	"github.com/getlantern/systray"
)

var (
	// 1x1 transparent PNG placeholder
	iconDefault = mustDecodeIcon("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=")
	// identical placeholder used for unread state; replace with real asset later
	iconUnread = mustDecodeIcon("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=")
)

func mustDecodeIcon(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// Systray implements Tray using getlantern/systray.
type Systray struct {
	showHide      func()
	toggle        func(bool)
	exit          func()
	notifications bool
}

// NewSystray creates a new Systray. The enabled parameter sets the initial
// notifications state and checkbox.
func NewSystray(enabled bool) *Systray {
	return &Systray{notifications: enabled}
}

// SetUnread switches the icon depending on unread status.
func (s *Systray) SetUnread(hasUnread bool) {
	if hasUnread {
		systray.SetIcon(iconUnread)
	} else {
		systray.SetIcon(iconDefault)
	}
}

// OnShowHide registers a callback for show/hide actions.
func (s *Systray) OnShowHide(fn func()) { s.showHide = fn }

// OnToggleNotifications registers a callback for enabling/disabling notifications.
func (s *Systray) OnToggleNotifications(fn func(bool)) { s.toggle = fn }

// OnExit registers a callback for exit requests.
func (s *Systray) OnExit(fn func()) { s.exit = fn }

// Run starts the tray event loop and blocks until the tray is closed.
func (s *Systray) Run() {
	systray.Run(s.onReady, s.onExit)
}

func (s *Systray) onReady() {
	systray.SetIcon(iconDefault)
	systray.SetTooltip("meshgo")

	mShow := systray.AddMenuItem("Show/Hide", "Show or hide the window")
	mToggle := systray.AddMenuItemCheckbox("Enable notifications", "Toggle notifications", s.notifications)
	if s.notifications {
		mToggle.Check()
	} else {
		mToggle.Uncheck()
	}
	mQuit := systray.AddMenuItem("Exit", "Quit meshgo")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if s.showHide != nil {
					s.showHide()
				}
			case <-mToggle.ClickedCh:
				s.notifications = !s.notifications
				if s.notifications {
					mToggle.Check()
				} else {
					mToggle.Uncheck()
				}
				if s.toggle != nil {
					s.toggle(s.notifications)
				}
			case <-mQuit.ClickedCh:
				if s.exit != nil {
					s.exit()
				}
				systray.Quit()
				return
			}
		}
	}()
}

func (s *Systray) onExit() {}
