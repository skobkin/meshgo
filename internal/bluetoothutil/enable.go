package bluetoothutil

import (
	"runtime"
	"strings"

	"tinygo.org/x/bluetooth"
)

func EnableAdapter(adapter *bluetooth.Adapter) error {
	if err := adapter.Enable(); err != nil {
		if isBenignEnableAdapterError(err) {
			return nil
		}
		return err
	}
	return nil
}

func isBenignEnableAdapterError(err error) bool {
	if err == nil || runtime.GOOS != "windows" {
		return false
	}

	// tinygo.org/x/bluetooth on Windows surfaces RoInitialize(S_FALSE=1) as
	// "Incorrect function.", even though this means COM is already initialized.
	msg := strings.TrimSpace(strings.ToLower(err.Error()))

	return msg == "incorrect function" || msg == "incorrect function."
}
