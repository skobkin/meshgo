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
	// Run starts the tray event loop and blocks until the tray is closed.
	Run()
	// Quit requests the tray event loop to exit.
	Quit()
	// Ready returns a channel that's closed when the tray is ready.
	Ready() <-chan struct{}
}

// Noop implements Tray with no side effects.
type Noop struct {
	showHide func()
	toggle   func(bool)
	exit     func()
	quit     chan struct{}
	ready    chan struct{}
}

func (n *Noop) SetUnread(bool) {}

func (n *Noop) OnShowHide(fn func()) { n.showHide = fn }

func (n *Noop) OnToggleNotifications(fn func(bool)) { n.toggle = fn }

func (n *Noop) OnExit(fn func()) { n.exit = fn }

// Run blocks until Quit is called.
func (n *Noop) Run() {
	if n.quit == nil {
		n.quit = make(chan struct{})
	}
	if n.ready == nil {
		n.ready = make(chan struct{})
	}
	select {
	case <-n.ready:
		// already closed
	default:
		close(n.ready)
	}
	slog.Info("tray disabled; running without system tray")
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

// Ready returns a channel that's closed immediately as the noop tray is ready at startup.
func (n *Noop) Ready() <-chan struct{} {
	if n.ready == nil {
		n.ready = make(chan struct{})
		close(n.ready)
	}
	return n.ready
}
