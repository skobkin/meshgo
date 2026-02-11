package bluetoothutil

import (
	"errors"
	"runtime"
	"testing"
)

func TestIsBenignEnableAdapterError(t *testing.T) {
	if isBenignEnableAdapterError(nil) {
		t.Fatalf("nil error must not be benign")
	}

	if isBenignEnableAdapterError(errors.New("some other error")) {
		t.Fatalf("unexpected benign match for unrelated error")
	}

	got := isBenignEnableAdapterError(errors.New("Incorrect function."))
	want := runtime.GOOS == "windows"
	if got != want {
		t.Fatalf("unexpected benign match decision: got=%v want=%v", got, want)
	}
}
