//go:build !cgo

package tray

// NewSystray returns a no-op Tray when CGO is disabled.
func NewSystray(enabled bool) Tray {
	return &Noop{}
}
