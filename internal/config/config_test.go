package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppConfigFillMissingDefaults(t *testing.T) {
	cfg := AppConfig{}
	cfg.FillMissingDefaults()

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

func TestDefaultEnablesNotificationTypes(t *testing.T) {
	cfg := Default()
	if cfg.UI.Notifications.NotifyWhenFocused {
		t.Fatalf("expected notify_when_focused to be disabled by default")
	}
	if !cfg.UI.Notifications.Events.IncomingMessage {
		t.Fatalf("expected incoming message notification to be enabled by default")
	}
	if !cfg.UI.Notifications.Events.NodeDiscovered {
		t.Fatalf("expected node discovered notification to be enabled by default")
	}
	if !cfg.UI.Notifications.Events.ConnectionStatus {
		t.Fatalf("expected connection status notification to be enabled by default")
	}
}

func TestLoadMissingNotificationsUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "connector": "ip",
    "host": "192.168.0.1"
  },
  "logging": {
    "level": "info",
    "log_to_file": false
  },
  "ui": {
    "last_selected_chat": "",
    "autostart": {
      "enabled": false,
      "mode": "normal"
    },
    "map_viewport": {
      "set": false,
      "zoom": 0,
      "x": 0,
      "y": 0
    }
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.UI.Notifications.NotifyWhenFocused {
		t.Fatalf("expected notify_when_focused to default to false")
	}
	if !cfg.UI.Notifications.Events.IncomingMessage || !cfg.UI.Notifications.Events.NodeDiscovered || !cfg.UI.Notifications.Events.ConnectionStatus {
		t.Fatalf("expected notification types to default to enabled, got %+v", cfg.UI.Notifications)
	}
}

func TestLoadPreservesExplicitNotificationFalseValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "connector": "ip",
    "host": "192.168.0.1"
  },
  "logging": {
    "level": "info",
    "log_to_file": false
  },
  "ui": {
    "notifications": {
      "notify_when_focused": false,
      "events": {
        "incoming_message": false,
        "node_discovered": false,
        "connection_status": false
      }
    }
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.UI.Notifications.NotifyWhenFocused {
		t.Fatalf("expected notify_when_focused=false to be preserved")
	}
	if cfg.UI.Notifications.Events.IncomingMessage {
		t.Fatalf("expected incoming_message=false to be preserved")
	}
	if cfg.UI.Notifications.Events.NodeDiscovered {
		t.Fatalf("expected node_discovered=false to be preserved")
	}
	if cfg.UI.Notifications.Events.ConnectionStatus {
		t.Fatalf("expected connection_status=false to be preserved")
	}
}

func TestLoadLegacyFlatNotificationFieldsAreIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "connector": "ip",
    "host": "192.168.0.1"
  },
  "ui": {
    "notifications": {
      "notify_when_focused": false,
      "incoming_message": false,
      "node_discovered": false,
      "connection_status": false
    }
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.UI.Notifications.NotifyWhenFocused {
		t.Fatalf("expected notify_when_focused=false to be preserved")
	}
	if !cfg.UI.Notifications.Events.IncomingMessage {
		t.Fatalf("expected legacy incoming_message field to be ignored")
	}
	if !cfg.UI.Notifications.Events.NodeDiscovered {
		t.Fatalf("expected legacy node_discovered field to be ignored")
	}
	if !cfg.UI.Notifications.Events.ConnectionStatus {
		t.Fatalf("expected legacy connection_status field to be ignored")
	}
}

func TestAppConfigFillMissingDefaultsNormalizesAutostartMode(t *testing.T) {
	cfg := AppConfig{
		UI: UIConfig{
			Autostart: AutostartConfig{
				Enabled: true,
				Mode:    AutostartMode("invalid"),
			},
		},
	}

	cfg.FillMissingDefaults()
	if cfg.UI.Autostart.Mode != AutostartModeNormal {
		t.Fatalf("expected invalid autostart mode to normalize to %q, got %q", AutostartModeNormal, cfg.UI.Autostart.Mode)
	}
}

func TestAppConfigFillMissingDefaultsNormalizesMapViewport(t *testing.T) {
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

	cfg.FillMissingDefaults()
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

func TestAppConfigFillMissingDefaultsClearsUnsetMapViewport(t *testing.T) {
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

	cfg.FillMissingDefaults()
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
