//go:build !linux

package bluetoothutil

import "tinygo.org/x/bluetooth"

func ResolveAdapter(_ string) *bluetooth.Adapter {
	// tinygo.org/x/bluetooth exposes custom adapter IDs via NewAdapter only on Linux.
	// On non-Linux platforms, use the default adapter.
	return bluetooth.DefaultAdapter
}
