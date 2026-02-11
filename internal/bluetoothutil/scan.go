package bluetoothutil

import "tinygo.org/x/bluetooth"

func StopScan(adapter *bluetooth.Adapter) error {
	err := adapter.StopScan()
	if err != nil && !IsBenignStopScanError(err) {
		return err
	}

	return nil
}

func NormalizeScanError(err error) error {
	if err == nil || IsBenignStopScanError(err) {
		return nil
	}

	return err
}
