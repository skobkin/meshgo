package transport

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/skobkin/meshgo/internal/bluetoothutil"
	"tinygo.org/x/bluetooth"
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

func TestResolveMeshtasticCharacteristicIndices(t *testing.T) {
	toRadio := bluetoothutil.MeshtasticToRadioUUID()
	fromRadio := bluetoothutil.MeshtasticFromRadioUUID()
	fromNum := bluetoothutil.MeshtasticFromNumUUID()

	t.Run("out of order", func(t *testing.T) {
		toIdx, fromIdx, fromNumIdx, err := resolveMeshtasticCharacteristicIndices([]bluetooth.UUID{
			fromNum,
			toRadio,
			fromRadio,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toIdx != 1 || fromIdx != 2 || fromNumIdx != 0 {
			t.Fatalf("unexpected indices: to=%d from=%d fromNum=%d", toIdx, fromIdx, fromNumIdx)
		}
	})

	t.Run("missing characteristic", func(t *testing.T) {
		_, _, _, err := resolveMeshtasticCharacteristicIndices([]bluetooth.UUID{
			toRadio,
			fromNum,
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("duplicate characteristic", func(t *testing.T) {
		_, _, _, err := resolveMeshtasticCharacteristicIndices([]bluetooth.UUID{
			toRadio,
			fromRadio,
			fromNum,
			fromNum,
		})
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}

type testErr string

func (e testErr) Error() string {
	return string(e)
}
