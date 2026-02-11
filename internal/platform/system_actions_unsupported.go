//go:build !linux && !windows

package platform

import "runtime"

type unsupportedSystemActions struct{}

func newSystemActions() SystemActions {
	return unsupportedSystemActions{}
}

func (unsupportedSystemActions) OpenBluetoothSettings() error {
	return openBluetoothSettingsForOS(runtime.GOOS, startCommandDetached)
}
