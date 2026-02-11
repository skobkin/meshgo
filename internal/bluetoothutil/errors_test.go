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
	if IsBenignStopScanError(testErr("operation canceled by user")) {
		t.Fatalf("cancel should not be treated as benign stop-scan error")
	}
	if IsBenignStopScanError(testErr("adapter scan stopped unexpectedly")) {
		t.Fatalf("stopped should not be treated as benign stop-scan error")
	}
	if !IsBenignStopScanError(testErr("bluetooth: there is no scan in progress")) {
		t.Fatalf("known no-scan-in-progress message should be benign")
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

func TestNormalizeScanError(t *testing.T) {
	if got := NormalizeScanError(nil); got != nil {
		t.Fatalf("expected nil for nil error, got %v", got)
	}
	benign := dbus.NewError("org.bluez.Error.NotReady", nil)
	if got := NormalizeScanError(benign); got != nil {
		t.Fatalf("expected nil for benign error, got %v", got)
	}
	serious := testErr("serious scan error")
	if got := NormalizeScanError(serious); got == nil || got.Error() != serious.Error() {
		t.Fatalf("expected serious error passthrough, got %v", got)
	}
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}
