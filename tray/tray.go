package tray

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
}

// Noop implements Tray with no side effects.
type Noop struct {
	showHide func()
	toggle   func(bool)
	exit     func()
}

func (n *Noop) SetUnread(bool) {}

func (n *Noop) OnShowHide(fn func()) { n.showHide = fn }

func (n *Noop) OnToggleNotifications(fn func(bool)) { n.toggle = fn }

func (n *Noop) OnExit(fn func()) { n.exit = fn }

func (n *Noop) Run() {}
