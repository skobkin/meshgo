package ui

import (
	"strings"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/notifications"
)

// FyneNotificationSender bridges app notifications to native Fyne notifications.
type FyneNotificationSender struct {
	app fyne.App
}

func NewFyneNotificationSender(app fyne.App) *FyneNotificationSender {
	return &FyneNotificationSender{app: app}
}

func (s *FyneNotificationSender) Send(notification notifications.Payload) {
	if s == nil || s.app == nil {
		return
	}

	title := strings.TrimSpace(notification.Title)
	content := strings.TrimSpace(notification.Content)
	if title == "" && content == "" {
		return
	}

	fyne.Do(func() {
		s.app.SendNotification(fyne.NewNotification(title, content))
	})
}
