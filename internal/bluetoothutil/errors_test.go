package bluetoothutil

import (
	"fmt"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestIsDBusErrorName(t *testing.T) {
	err := dbus.NewError("org.bluez.Error.InProgress", nil)
	if !IsDBusErrorName(err, "org.bluez.Error.InProgress") {
		t.Fatalf("expected direct dbus error match")
	}
	if !IsDBusErrorName(fmt.Errorf("wrapped: %w", err), "org.bluez.Error.InProgress") {
		t.Fatalf("expected wrapped dbus error match")
	}
	if IsDBusErrorName(testErr("plain"), "org.bluez.Error.InProgress") {
		t.Fatalf("unexpected match for plain error")
	}
}

func TestIsBenignStopScanError(t *testing.T) {
	if !IsBenignStopScanError(nil) {
		t.Fatalf("nil error should be benign")
	}
	if !IsBenignStopScanError(dbus.NewError("org.bluez.Error.NotReady", nil)) {
		t.Fatalf("NotReady should be benign")
	}
	if !IsBenignStopScanError(dbus.NewError("org.bluez.Error.Failed", []interface{}{"No discovery started"})) {
		t.Fatalf("no discovery started should be benign")
	}
	if IsBenignStopScanError(testErr("some serious error")) {
		t.Fatalf("unexpected benign classification")
	}
}

func TestIsScanAlreadyInProgressError(t *testing.T) {
	if IsScanAlreadyInProgressError(nil) {
		t.Fatalf("nil should not match")
	}
	if !IsScanAlreadyInProgressError(dbus.NewError("org.bluez.Error.InProgress", nil)) {
		t.Fatalf("dbus in-progress should match")
	}
	if !IsScanAlreadyInProgressError(testErr("Operation already in progress")) {
		t.Fatalf("text fallback should match")
	}
	if IsScanAlreadyInProgressError(testErr("another error")) {
		t.Fatalf("unexpected positive match")
	}
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}
