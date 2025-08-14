//go:build !cgo

package tray

import "log/slog"

// NewSystray returns a no-op Tray when CGO is disabled.
func NewSystray(enabled bool) Tray {
	slog.Info("systray unavailable; running without GUI")
	return &Noop{quit: make(chan struct{}), ready: make(chan struct{})}
}
