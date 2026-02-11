package app

import (
	"testing"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/transport"
)

func TestNewTransportForConnection(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ConnectionConfig
		want    string
		wantErr bool
	}{
		{
			name: "ip",
			cfg: config.ConnectionConfig{
				Connector: config.ConnectorIP,
				Host:      "127.0.0.1",
			},
			want: "ip",
		},
		{
			name: "serial",
			cfg: config.ConnectionConfig{
				Connector:  config.ConnectorSerial,
				SerialPort: "/dev/ttyACM0",
				SerialBaud: 115200,
			},
			want: "serial",
		},
		{
			name: "bluetooth",
			cfg: config.ConnectionConfig{
				Connector:        config.ConnectorBluetooth,
				BluetoothAddress: "AA:BB:CC:DD:EE:FF",
			},
			want: "bluetooth",
		},
	}

	for _, tc := range tests {
		tr, err := NewTransportForConnection(tc.cfg)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if tr.Name() != tc.want {
			t.Fatalf("%s: expected transport %q, got %q", tc.name, tc.want, tr.Name())
		}
	}
}

func TestConnectionTransportApplySwitchesImplementation(t *testing.T) {
	initial := config.ConnectionConfig{
		Connector: config.ConnectorIP,
		Host:      "192.168.1.10",
	}
	connTr, err := NewConnectionTransport(initial)
	if err != nil {
		t.Fatalf("new connection transport: %v", err)
	}

	if connTr.Name() != "ip" {
		t.Fatalf("expected initial transport ip, got %q", connTr.Name())
	}

	next := config.ConnectionConfig{
		Connector:  config.ConnectorSerial,
		SerialPort: "COM3",
		SerialBaud: 115200,
	}
	if err := connTr.Apply(next); err != nil {
		t.Fatalf("apply serial config: %v", err)
	}
	if connTr.Name() != "serial" {
		t.Fatalf("expected switched transport serial, got %q", connTr.Name())
	}

	gotCfg := connTr.Config()
	if gotCfg.Connector != config.ConnectorSerial {
		t.Fatalf("expected serial config, got %q", gotCfg.Connector)
	}
}

func TestConnectionTransportApplyKeepsCurrentOnError(t *testing.T) {
	initial := config.ConnectionConfig{
		Connector: config.ConnectorIP,
		Host:      "192.168.1.10",
	}
	connTr, err := NewConnectionTransport(initial)
	if err != nil {
		t.Fatalf("new connection transport: %v", err)
	}

	err = connTr.Apply(config.ConnectionConfig{
		Connector: config.ConnectorType("usb"),
	})
	if err == nil {
		t.Fatalf("expected apply error for unknown connector")
	}
	if connTr.Name() != "ip" {
		t.Fatalf("expected transport to remain ip after failed apply, got %q", connTr.Name())
	}
}

func TestConnectionTransportImplementsTransportInterface(t *testing.T) {
	initial := config.ConnectionConfig{
		Connector: config.ConnectorIP,
		Host:      "192.168.1.10",
	}
	connTr, err := NewConnectionTransport(initial)
	if err != nil {
		t.Fatalf("new connection transport: %v", err)
	}

	var _ transport.Transport = connTr
}
