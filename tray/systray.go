//go:build cgo

package tray

import (
	"encoding/base64"
	"log/slog"

	"github.com/getlantern/systray"
)

var (
	// simple gray square 16x16 PNG
	iconDefault = mustDecodeIcon("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAHklEQVR4nGI5ceIEAymAiSTVoxpGNQwpDYAAAAD//2Z5Antq1nXTAAAAAElFTkSuQmCC")
	// same base with a red badge in the top-right corner for unread state
	iconUnread = mustDecodeIcon("iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAALklEQVR4nGI5ceIEA24gYmGBJsKERzVWMBw0sOCXfoMRhoPQD6MaiAGAAAAA//9gPwV9LZxsbwAAAABJRU5ErkJggg==")
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
	ready         chan struct{}
}

// NewSystray creates a new Systray. The enabled parameter sets the initial
// notifications state and checkbox.
func NewSystray(enabled bool) Tray {
	return &Systray{notifications: enabled, ready: make(chan struct{})}
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
	slog.Info("starting systray")
	systray.Run(s.onReady, s.onExit)
}

// Quit stops the tray event loop.
func (s *Systray) Quit() {
	systray.Quit()
}

func (s *Systray) onReady() {
	slog.Info("systray ready")
	close(s.ready)
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

func (s *Systray) onExit() {
	slog.Info("systray exit")
}

// Ready returns a channel that's closed when the systray icon is displayed.
func (s *Systray) Ready() <-chan struct{} { return s.ready }
