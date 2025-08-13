package notify

import (
	"time"

	"github.com/gen2brain/beeep"
)

// Notifier delivers desktop notifications for new messages.
type Notifier interface {
	// NotifyNewMessage displays a notification for a new message in chatID
	// with the provided title and body at the given timestamp.
	NotifyNewMessage(chatID, title, body string, ts time.Time) error
	// SetEnabled toggles whether notifications are shown.
	SetEnabled(enabled bool)
}

// BeeepNotifier is a Notifier implementation backed by the beeep library.
type BeeepNotifier struct {
	enabled    bool
	notifyFunc func(title, message string, appIcon any) error
}

// NewBeeep creates a BeeepNotifier. When enabled is false, NotifyNewMessage
// becomes a no-op.
func NewBeeep(enabled bool) *BeeepNotifier {
	return &BeeepNotifier{enabled: enabled, notifyFunc: beeep.Notify}
}

// NotifyNewMessage shows a desktop notification using beeep when enabled.
func (b *BeeepNotifier) NotifyNewMessage(chatID, title, body string, ts time.Time) error {
	if !b.enabled {
		return nil
	}
	return b.notifyFunc(title, body, "")
}

// SetEnabled enables or disables desktop notifications.
func (b *BeeepNotifier) SetEnabled(enabled bool) {
	b.enabled = enabled
}
