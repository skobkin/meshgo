package platform

import (
	"errors"
	"testing"
)

func TestBluetoothSettingsCommandsForOS(t *testing.T) {
	windows, err := bluetoothSettingsCommandsForOS("windows")
	if err != nil {
		t.Fatalf("unexpected windows commands error: %v", err)
	}
	if len(windows) != 1 {
		t.Fatalf("unexpected windows command count: %d", len(windows))
	}
	if windows[0].name != "cmd" {
		t.Fatalf("unexpected windows command: %q", windows[0].name)
	}

	linux, err := bluetoothSettingsCommandsForOS("linux")
	if err != nil {
		t.Fatalf("unexpected linux commands error: %v", err)
	}
	if len(linux) == 0 {
		t.Fatalf("expected linux command fallbacks")
	}
	if linux[0].name != "systemsettings6" {
		t.Fatalf("unexpected first linux command: %q", linux[0].name)
	}
}

func TestBluetoothSettingsCommandsForOSUnsupported(t *testing.T) {
	if _, err := bluetoothSettingsCommandsForOS("darwin"); err == nil {
		t.Fatalf("expected unsupported os error")
	}
}

func TestOpenBluetoothSettingsForOSFallsBack(t *testing.T) {
	var attempts []string
	start := func(name string, args ...string) error {
		attempts = append(attempts, name)
		if len(attempts) == 1 {
			return errors.New("first command failed")
		}

		return nil
	}

	if err := openBluetoothSettingsForOS("linux", start); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) < 2 {
		t.Fatalf("expected fallback attempt, got %d", len(attempts))
	}
}

func TestOpenBluetoothSettingsForOSAllFail(t *testing.T) {
	start := func(_ string, _ ...string) error {
		return errors.New("fail")
	}

	if err := openBluetoothSettingsForOS("windows", start); err == nil {
		t.Fatalf("expected aggregate error")
	}
}
