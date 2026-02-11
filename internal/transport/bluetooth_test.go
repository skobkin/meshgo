package transport

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/skobkin/meshgo/internal/bluetoothutil"
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
	if got := bluetoothutil.ResolveAdapter(""); got == nil {
		t.Fatalf("expected default adapter, got nil")
	}
	if got := bluetoothutil.ResolveAdapter("   "); got == nil {
		t.Fatalf("expected default adapter for empty input, got nil")
	}
	if got := bluetoothutil.ResolveAdapter("hci1"); got == nil {
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

func TestBluetoothConnStateCloseAndError(t *testing.T) {
	state := &bluetoothConnState{
		closed: make(chan struct{}),
	}

	state.setAsyncError(testErr("drain failed"))
	state.markClosed()
	state.markClosed()

	select {
	case <-state.closed:
	default:
		t.Fatalf("expected closed channel to be closed")
	}

	if got := state.closeErr(); got == nil || got.Error() != "drain failed" {
		t.Fatalf("unexpected async error: %v", got)
	}
}

func TestBluetoothTransportReadFrameReturnsAsyncError(t *testing.T) {
	state := &bluetoothConnState{
		frameCh: make(chan []byte),
		closed:  make(chan struct{}),
	}
	state.setAsyncError(testErr("from-radio drain failed"))
	state.markClosed()

	tr := &BluetoothTransport{conn: state}
	_, err := tr.ReadFrame(context.Background())
	if err == nil {
		t.Fatalf("expected read error")
	}
	if err.Error() != "from-radio drain failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}
