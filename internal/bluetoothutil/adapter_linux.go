//go:build linux

package bluetoothutil

import (
	"strings"

	"tinygo.org/x/bluetooth"
)

func ResolveAdapter(adapterID string) *bluetooth.Adapter {
	trimmed := strings.TrimSpace(adapterID)
	if trimmed == "" {
		return bluetooth.DefaultAdapter
	}
	return bluetooth.NewAdapter(trimmed)
}
