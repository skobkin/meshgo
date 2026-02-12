package config

import "testing"

func TestAppConfigApplyDefaults(t *testing.T) {
	cfg := AppConfig{}
	cfg.ApplyDefaults()

	if cfg.Connection.Connector != ConnectorIP {
		t.Fatalf("expected default connector %q, got %q", ConnectorIP, cfg.Connection.Connector)
	}
	if cfg.Connection.SerialBaud != DefaultSerialBaud {
		t.Fatalf("expected default serial baud %d, got %d", DefaultSerialBaud, cfg.Connection.SerialBaud)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("expected default log level info, got %q", cfg.Logging.Level)
	}
	if cfg.UI.Autostart.Enabled {
		t.Fatalf("expected autostart to be disabled by default")
	}
	if cfg.UI.Autostart.Mode != AutostartModeNormal {
		t.Fatalf("expected default autostart mode %q, got %q", AutostartModeNormal, cfg.UI.Autostart.Mode)
	}
}

func TestAppConfigApplyDefaultsNormalizesAutostartMode(t *testing.T) {
	cfg := AppConfig{
		UI: UIConfig{
			Autostart: AutostartConfig{
				Enabled: true,
				Mode:    AutostartMode("invalid"),
			},
		},
	}

	cfg.ApplyDefaults()
	if cfg.UI.Autostart.Mode != AutostartModeNormal {
		t.Fatalf("expected invalid autostart mode to normalize to %q, got %q", AutostartModeNormal, cfg.UI.Autostart.Mode)
	}
}

func TestAppConfigApplyDefaultsNormalizesMapViewport(t *testing.T) {
	cfg := AppConfig{
		UI: UIConfig{
			MapViewport: MapViewportConfig{
				Set:  true,
				Zoom: 55,
				X:    10,
				Y:    -4,
			},
		},
	}

	cfg.ApplyDefaults()
	if !cfg.UI.MapViewport.Set {
		t.Fatalf("expected map viewport to stay set")
	}
	if cfg.UI.MapViewport.Zoom != 19 {
		t.Fatalf("expected zoom to clamp to 19, got %d", cfg.UI.MapViewport.Zoom)
	}
	if cfg.UI.MapViewport.X != 10 || cfg.UI.MapViewport.Y != -4 {
		t.Fatalf("expected pan offsets to remain unchanged")
	}
}

func TestAppConfigApplyDefaultsClearsUnsetMapViewport(t *testing.T) {
	cfg := AppConfig{
		UI: UIConfig{
			MapViewport: MapViewportConfig{
				Set:  false,
				Zoom: 9,
				X:    1,
				Y:    2,
			},
		},
	}

	cfg.ApplyDefaults()
	if cfg.UI.MapViewport.Set {
		t.Fatalf("expected map viewport to remain unset")
	}
	if cfg.UI.MapViewport.Zoom != 0 || cfg.UI.MapViewport.X != 0 || cfg.UI.MapViewport.Y != 0 {
		t.Fatalf("expected unset viewport to normalize to zero values, got %+v", cfg.UI.MapViewport)
	}
}

func TestAppConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AppConfig
		wantErr bool
	}{
		{
			name: "valid ip",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector: ConnectorIP,
					Host:      "192.168.1.10",
				},
			},
		},
		{
			name: "invalid ip without host",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector: ConnectorIP,
				},
			},
			wantErr: true,
		},
		{
			name: "valid serial",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector:  ConnectorSerial,
					SerialPort: "/dev/ttyACM0",
					SerialBaud: 115200,
				},
			},
		},
		{
			name: "invalid serial without port",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector:  ConnectorSerial,
					SerialBaud: 115200,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid serial with non-positive baud",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector:  ConnectorSerial,
					SerialPort: "COM3",
					SerialBaud: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "valid bluetooth",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector:        ConnectorBluetooth,
					BluetoothAddress: "AA:BB:CC:DD:EE:FF",
				},
			},
		},
		{
			name: "invalid bluetooth without address",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector: ConnectorBluetooth,
				},
			},
			wantErr: true,
		},
		{
			name: "unknown connector",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Connector: ConnectorType("usb"),
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		err := tc.cfg.Validate()
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: expected no error, got %v", tc.name, err)
		}
	}
}
