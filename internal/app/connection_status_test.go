package app

import (
	"testing"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
)

func TestTransportNameFromConnector(t *testing.T) {
	tests := []struct {
		name      string
		connector config.ConnectorType
		want      string
	}{
		{name: "ip", connector: config.ConnectorIP, want: "ip"},
		{name: "serial", connector: config.ConnectorSerial, want: "serial"},
		{name: "bluetooth", connector: config.ConnectorBluetooth, want: "bluetooth"},
		{name: "unknown", connector: "custom", want: "custom"},
		{name: "empty", connector: "", want: "unknown"},
	}

	for _, tc := range tests {
		if got := TransportNameFromConnector(tc.connector); got != tc.want {
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
		{name: "ip", cfg: config.ConnectionConfig{Connector: config.ConnectorIP, Host: "192.168.1.10"}, want: "192.168.1.10"},
		{name: "serial", cfg: config.ConnectionConfig{Connector: config.ConnectorSerial, SerialPort: "/dev/ttyACM0", SerialBaud: 115200}, want: "/dev/ttyACM0"},
		{name: "bluetooth", cfg: config.ConnectionConfig{Connector: config.ConnectorBluetooth, BluetoothAddress: "AA:BB:CC:DD:EE:FF"}, want: "AA:BB:CC:DD:EE:FF"},
		{name: "unknown", cfg: config.ConnectionConfig{Connector: "custom"}, want: ""},
	}

	for _, tc := range tests {
		if got := ConnectionTarget(tc.cfg); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestConnectionStatusFromConfig(t *testing.T) {
	status := ConnectionStatusFromConfig(config.ConnectionConfig{
		Connector:  config.ConnectorSerial,
		SerialPort: "/dev/ttyACM2",
		SerialBaud: 115200,
	})

	if status.State != connectors.ConnectionStateConnecting {
		t.Fatalf("expected connecting state, got %q", status.State)
	}
	if status.TransportName != "serial" {
		t.Fatalf("expected serial transport name, got %q", status.TransportName)
	}
	if status.Target != "/dev/ttyACM2" {
		t.Fatalf("expected serial target, got %q", status.Target)
	}
}
