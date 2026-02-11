package main

import (
	"testing"

	"github.com/skobkin/meshgo/internal/config"
)

func TestConnectionTarget(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.ConnectionConfig
		want string
	}{
		{name: "ip", cfg: config.ConnectionConfig{Connector: config.ConnectorIP, Host: "192.168.1.10"}, want: "192.168.1.10"},
		{name: "serial", cfg: config.ConnectionConfig{Connector: config.ConnectorSerial, SerialPort: "/dev/ttyACM0", SerialBaud: 115200}, want: "/dev/ttyACM0@115200"},
		{name: "bluetooth", cfg: config.ConnectionConfig{Connector: config.ConnectorBluetooth, BluetoothAddress: "AA:BB:CC:DD:EE:FF"}, want: "AA:BB:CC:DD:EE:FF"},
		{name: "bluetooth with adapter", cfg: config.ConnectionConfig{Connector: config.ConnectorBluetooth, BluetoothAddress: "AA:BB:CC:DD:EE:FF", BluetoothAdapter: "hci1"}, want: "AA:BB:CC:DD:EE:FF (hci1)"},
	}

	for _, tc := range tests {
		if got := connectionTarget(tc.cfg); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}
