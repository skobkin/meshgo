package bluetoothutil

import (
	"errors"
	"strings"

	"github.com/godbus/dbus/v5"
)

func IsDBusErrorName(err error, want string) bool {
	var dbusErrPtr *dbus.Error
	if errors.As(err, &dbusErrPtr) && dbusErrPtr != nil && dbusErrPtr.Name == want {
		return true
	}

	var dbusErr dbus.Error
	return errors.As(err, &dbusErr) && dbusErr.Name == want
}

func IsBenignStopScanError(err error) bool {
	if err == nil {
		return true
	}
	if IsDBusErrorName(err, "org.bluez.Error.NotReady") {
		return true
	}
	if IsDBusErrorName(err, "org.bluez.Error.Failed") && strings.Contains(strings.ToLower(err.Error()), "no discovery started") {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cancel") ||
		strings.Contains(msg, "stopped") ||
		strings.Contains(msg, "not scanning") ||
		strings.Contains(msg, "no scan in progress")
}

func IsScanAlreadyInProgressError(err error) bool {
	if err == nil {
		return false
	}
	if IsDBusErrorName(err, "org.bluez.Error.InProgress") {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "already in progress")
}
