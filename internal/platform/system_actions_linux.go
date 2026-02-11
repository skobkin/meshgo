//go:build linux

package platform

type linuxSystemActions struct{}

func newSystemActions() SystemActions {
	return linuxSystemActions{}
}

func (linuxSystemActions) OpenBluetoothSettings() error {
	return openBluetoothSettingsForOS("linux", startCommandDetached)
}
