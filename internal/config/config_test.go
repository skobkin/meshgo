package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppConfigFillMissingDefaults(t *testing.T) {
	cfg := AppConfig{}
	cfg.FillMissingDefaults()

	if cfg.Connection.Transport != TransportIP {
		t.Fatalf("expected default transport %q, got %q", TransportIP, cfg.Connection.Transport)
	}
	if cfg.Connection.SerialBaud != DefaultSerialBaud {
		t.Fatalf("expected default serial baud %d, got %d", DefaultSerialBaud, cfg.Connection.SerialBaud)
	}
	if cfg.Connection.BluetoothTestingEnabled {
		t.Fatalf("expected bluetooth testing to be disabled by default")
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
	if !cfg.UI.Notifications.Events.UpdateAvailable {
		t.Fatalf("expected update available notification to be enabled by default")
	}
}

func TestLoadMissingNotificationsUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "transport": "ip",
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
	if !cfg.UI.Notifications.Events.IncomingMessage ||
		!cfg.UI.Notifications.Events.NodeDiscovered ||
		!cfg.UI.Notifications.Events.ConnectionStatus ||
		!cfg.UI.Notifications.Events.UpdateAvailable {
		t.Fatalf("expected notification types to default to enabled, got %+v", cfg.UI.Notifications)
	}
}

func TestLoadPreservesExplicitNotificationFalseValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "transport": "ip",
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
        "connection_status": false,
        "update_available": false
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
	if cfg.UI.Notifications.Events.UpdateAvailable {
		t.Fatalf("expected update_available=false to be preserved")
	}
}

func TestLoadLegacyFlatNotificationFieldsAreIgnored(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{
  "connection": {
    "transport": "ip",
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
	if !cfg.UI.Notifications.Events.UpdateAvailable {
		t.Fatalf("expected missing update_available to default to enabled")
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

func TestAppConfigFillMissingDefaultsEnablesBluetoothTestingForBluetoothTransport(t *testing.T) {
	cfg := AppConfig{
		Connection: ConnectionConfig{
			Transport:               TransportBluetooth,
			BluetoothAddress:        "AA:BB:CC:DD:EE:FF",
			BluetoothTestingEnabled: false,
		},
	}

	cfg.FillMissingDefaults()

	if !cfg.Connection.BluetoothTestingEnabled {
		t.Fatalf("expected bluetooth testing to be enabled when bluetooth transport is selected")
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
					Transport: TransportIP,
					Host:      "192.168.1.10",
				},
			},
		},
		{
			name: "invalid ip without host",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport: TransportIP,
				},
			},
			wantErr: true,
		},
		{
			name: "valid serial",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport:  TransportSerial,
					SerialPort: "/dev/ttyACM0",
					SerialBaud: 115200,
				},
			},
		},
		{
			name: "invalid serial without port",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport:  TransportSerial,
					SerialBaud: 115200,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid serial with non-positive baud",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport:  TransportSerial,
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
					Transport:        TransportBluetooth,
					BluetoothAddress: "AA:BB:CC:DD:EE:FF",
				},
			},
		},
		{
			name: "invalid bluetooth without address",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport: TransportBluetooth,
				},
			},
			wantErr: true,
		},
		{
			name: "unknown transport",
			cfg: AppConfig{
				Connection: ConnectionConfig{
					Transport: TransportType("usb"),
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

func TestLoadPreservesTransportField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with new "transport" field
	newConfig := `{
  "connection": {
    "transport": "bluetooth",
    "bluetooth_address": "AA:BB:CC:DD:EE:FF"
  }
}`
	if err := os.WriteFile(configPath, []byte(newConfig), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Connection.Transport != TransportBluetooth {
		t.Fatalf("expected transport %q, got %q", TransportBluetooth, cfg.Connection.Transport)
	}
}

func TestLoadWithEmptyConnectionSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with minimal content, no connection section
	minimalConfig := `{}`
	if err := os.WriteFile(configPath, []byte(minimalConfig), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should use defaults
	if cfg.Connection.Transport != TransportIP {
		t.Fatalf("expected default transport %q, got %q", TransportIP, cfg.Connection.Transport)
	}
}
