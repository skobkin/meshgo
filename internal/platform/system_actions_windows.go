//go:build windows

package platform

type windowsSystemActions struct{}

func newSystemActions() SystemActions {
	return windowsSystemActions{}
}

func (windowsSystemActions) OpenBluetoothSettings() error {
	return openBluetoothSettingsForOS("windows", startCommandDetached)
}
