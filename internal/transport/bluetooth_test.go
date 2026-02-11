package transport

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestParseBluetoothAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid upper", input: "AA:BB:CC:DD:EE:FF"},
		{name: "valid lower", input: "aa:bb:cc:dd:ee:ff"},
		{name: "empty", input: "   ", wantErr: true},
		{name: "invalid", input: "not-a-mac", wantErr: true},
	}

	for _, tc := range tests {
		_, err := parseBluetoothAddress(tc.input)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func TestResolveBluetoothAdapter(t *testing.T) {
	if got := resolveBluetoothAdapter(""); got == nil {
		t.Fatalf("expected default adapter, got nil")
	}
	if got := resolveBluetoothAdapter("   "); got == nil {
		t.Fatalf("expected default adapter for empty input, got nil")
	}
	if got := resolveBluetoothAdapter("hci1"); got == nil {
		t.Fatalf("expected adapter for explicit id, got nil")
	}
}

func TestShouldRetryBluetoothConnectWithDiscovery(t *testing.T) {
	err := dbus.NewError("org.freedesktop.DBus.Error.UnknownMethod", []interface{}{
		`Method "Get" with signature "ss" on interface "org.freedesktop.DBus.Properties" doesn't exist`,
	})
	got := shouldRetryBluetoothConnectWithDiscovery(fmt.Errorf("wrapped: %w", err))
	want := runtime.GOOS == "linux"
	if got != want {
		t.Fatalf("unexpected retry decision: got=%v want=%v", got, want)
	}
}
