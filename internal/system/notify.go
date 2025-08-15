package system

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Notifier struct {
	enabled    bool
	logger     *slog.Logger
	lastNotify map[string]time.Time
	mu         sync.RWMutex
}

func NewNotifier(logger *slog.Logger) *Notifier {
	return &Notifier{
		enabled:    true,
		logger:     logger,
		lastNotify: make(map[string]time.Time),
	}
}

func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
	n.logger.Info("Notifications", "enabled", enabled)
}

func (n *Notifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}

func (n *Notifier) NotifyNewMessage(chatID, title, body string, timestamp time.Time) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.enabled {
		return nil
	}

	// Throttle notifications per chat (2 second cooldown)
	if lastTime, exists := n.lastNotify[chatID]; exists {
		if time.Since(lastTime) < 2*time.Second {
			n.logger.Debug("Notification throttled", "chat", chatID)
			return nil
		}
	}

	// Use beeep for cross-platform notifications
	err := beeep.Notify(title, body, "")
	if err != nil {
		n.logger.Error("Failed to send notification", "error", err, "chat", chatID)
		return fmt.Errorf("notification failed: %w", err)
	}

	n.lastNotify[chatID] = time.Now()
	n.logger.Debug("Notification sent", "chat", chatID, "title", title)
	return nil
}

func (n *Notifier) Alert(title, message string) error {
	n.mu.RLock()
	enabled := n.enabled
	n.mu.RUnlock()

	if !enabled {
		return nil
	}

	err := beeep.Alert(title, message, "")
	if err != nil {
		n.logger.Error("Failed to send alert", "error", err)
		return fmt.Errorf("alert failed: %w", err)
	}

	n.logger.Debug("Alert sent", "title", title)
	return nil
}

func (n *Notifier) Beep() error {
	n.mu.RLock()
	enabled := n.enabled
	n.mu.RUnlock()

	if !enabled {
		return nil
	}

	err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if err != nil {
		n.logger.Error("Failed to beep", "error", err)
		return fmt.Errorf("beep failed: %w", err)
	}

	return nil
}
