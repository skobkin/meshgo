package app

import (
	"testing"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

func TestTransportNameFromType(t *testing.T) {
	tests := []struct {
		name      string
		transport config.TransportType
		want      string
	}{
		{name: "ip", transport: config.TransportIP, want: "ip"},
		{name: "serial", transport: config.TransportSerial, want: "serial"},
		{name: "bluetooth", transport: config.TransportBluetooth, want: "bluetooth"},
		{name: "unknown", transport: "custom", want: "custom"},
		{name: "empty", transport: "", want: "unknown"},
	}

	for _, tc := range tests {
		if got := TransportNameFromType(tc.transport); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestConnectionTarget(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ConnectionConfig
		want string
	}{
		{name: "ip", cfg: config.ConnectionConfig{Transport: config.TransportIP, Host: "192.168.1.10"}, want: "192.168.1.10"},
		{name: "serial", cfg: config.ConnectionConfig{Transport: config.TransportSerial, SerialPort: "/dev/ttyACM0", SerialBaud: 115200}, want: "/dev/ttyACM0"},
		{name: "bluetooth", cfg: config.ConnectionConfig{Transport: config.TransportBluetooth, BluetoothAddress: "AA:BB:CC:DD:EE:FF"}, want: "AA:BB:CC:DD:EE:FF"},
		{name: "unknown", cfg: config.ConnectionConfig{Transport: "custom"}, want: ""},
	}

	for _, tc := range tests {
		if got := ConnectionTarget(tc.cfg); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestConnectionStatusFromConfig(t *testing.T) {
	status := ConnectionStatusFromConfig(config.ConnectionConfig{
		Transport:  config.TransportSerial,
		SerialPort: "/dev/ttyACM2",
		SerialBaud: 115200,
	})

	if status.State != busmsg.ConnectionStateConnecting {
		t.Fatalf("expected connecting state, got %q", status.State)
	}
	if status.TransportName != "serial" {
		t.Fatalf("expected serial transport name, got %q", status.TransportName)
	}
	if status.Target != "/dev/ttyACM2" {
		t.Fatalf("expected serial target, got %q", status.Target)
	}
}
