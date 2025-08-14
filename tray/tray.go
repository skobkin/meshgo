package tray

import "log/slog"

// Tray controls the application tray icon state and routes user interactions.
type Tray interface {
	// SetUnread toggles the unread indicator.
	SetUnread(hasUnread bool)
	// OnShowHide registers a callback for show/hide actions.
	OnShowHide(fn func())
	// OnToggleNotifications registers a callback for enabling/disabling notifications.
	OnToggleNotifications(fn func(enabled bool))
	// OnExit registers a callback for exit requests.
	OnExit(fn func())
	// OnReady registers a callback invoked when the tray is ready.
	OnReady(fn func())
	// Run starts the tray event loop and blocks until the tray is closed.
	Run()
	// Quit requests the tray event loop to exit.
	Quit()
}

// Noop implements Tray with no side effects.
type Noop struct {
	showHide func()
	toggle   func(bool)
	exit     func()
	ready    func()
	quit     chan struct{}
}

func (n *Noop) SetUnread(bool) {}

func (n *Noop) OnShowHide(fn func()) { n.showHide = fn }

func (n *Noop) OnToggleNotifications(fn func(bool)) { n.toggle = fn }

func (n *Noop) OnExit(fn func()) { n.exit = fn }

func (n *Noop) OnReady(fn func()) { n.ready = fn }

// Run blocks until Quit is called.
func (n *Noop) Run() {
	if n.quit == nil {
		n.quit = make(chan struct{})
	}
	slog.Info("tray disabled; running without system tray")
	if n.ready != nil {
		slog.Info("tray ready")
		n.ready()
	}
	<-n.quit
}

// Quit unblocks Run and calls the exit callback if set.
func (n *Noop) Quit() {
	if n.quit != nil {
		select {
		case <-n.quit:
			// already closed
		default:
			close(n.quit)
		}
	}
	if n.exit != nil {
		n.exit()
	}
	slog.Info("tray quit")
}
