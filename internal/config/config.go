package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConnectorType identifies which transport backend should be used.
type ConnectorType string

// AutostartMode controls how the app is launched by OS autostart.
type AutostartMode string

const (
	ConnectorIP        ConnectorType = "ip"
	ConnectorBluetooth ConnectorType = "bluetooth"
	ConnectorSerial    ConnectorType = "serial"
	DefaultSerialBaud                = 115200

	AutostartModeNormal     AutostartMode = "normal"
	AutostartModeBackground AutostartMode = "background"
)

// LoggingConfig defines runtime logging behavior.
type LoggingConfig struct {
	Level     string `json:"level"`
	LogToFile bool   `json:"log_to_file"`
}

// ConnectionConfig contains connector-specific connection parameters.
type ConnectionConfig struct {
	Connector        ConnectorType `json:"connector"`
	Host             string        `json:"host"`
	SerialPort       string        `json:"serial_port"`
	SerialBaud       int           `json:"serial_baud"`
	BluetoothAddress string        `json:"bluetooth_address"`
	BluetoothAdapter string        `json:"bluetooth_adapter"`
}

// UIConfig stores persistent UI preferences.
type UIConfig struct {
	LastSelectedChat string             `json:"last_selected_chat"`
	Autostart        AutostartConfig    `json:"autostart"`
	MapViewport      MapViewportConfig  `json:"map_viewport"`
	Notifications    NotificationConfig `json:"notifications"`
}

// AutostartConfig stores autostart preferences saved in user config.
type AutostartConfig struct {
	Enabled bool          `json:"enabled"`
	Mode    AutostartMode `json:"mode"`
}

// MapViewportConfig stores the latest map tab viewport selected by user.
type MapViewportConfig struct {
	Set  bool `json:"set"`
	Zoom int  `json:"zoom"`
	X    int  `json:"x"`
	Y    int  `json:"y"`
}

// NotificationConfig stores desktop notification preferences.
type NotificationConfig struct {
	NotifyWhenFocused bool                     `json:"notify_when_focused"`
	Events            NotificationEventsConfig `json:"events"`
}

// NotificationEventsConfig stores per-event notification toggles.
type NotificationEventsConfig struct {
	IncomingMessage  bool `json:"incoming_message"`
	NodeDiscovered   bool `json:"node_discovered"`
	ConnectionStatus bool `json:"connection_status"`
	UpdateAvailable  bool `json:"update_available"`
}

// AppConfig is the root persisted application configuration.
type AppConfig struct {
	Connection ConnectionConfig `json:"connection"`
	Logging    LoggingConfig    `json:"logging"`
	UI         UIConfig         `json:"ui"`
}

func Default() AppConfig {
	return AppConfig{
		Connection: ConnectionConfig{
			Connector:        ConnectorIP,
			Host:             "",
			SerialPort:       "",
			SerialBaud:       DefaultSerialBaud,
			BluetoothAddress: "",
			BluetoothAdapter: "",
		},
		Logging: LoggingConfig{
			Level:     "info",
			LogToFile: false,
		},
		UI: UIConfig{
			LastSelectedChat: "",
			Autostart: AutostartConfig{
				Enabled: false,
				Mode:    AutostartModeNormal,
			},
			MapViewport: MapViewportConfig{},
			Notifications: NotificationConfig{
				NotifyWhenFocused: false,
				Events: NotificationEventsConfig{
					IncomingMessage:  true,
					NodeDiscovered:   true,
					ConnectionStatus: true,
					UpdateAvailable:  true,
				},
			},
		},
	}
}

func Load(path string) (AppConfig, error) {
	cfg := Default()
	cleanPath := filepath.Clean(path)
	// #nosec G304 -- path is resolved by app runtime and points to user config dir.
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}

		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("decode config json: %w", err)
	}

	cfg.FillMissingDefaults()

	return cfg, nil
}

func (c *AppConfig) FillMissingDefaults() {
	if c.Connection.Connector == "" {
		c.Connection.Connector = ConnectorIP
	}
	if c.Connection.SerialBaud <= 0 {
		c.Connection.SerialBaud = DefaultSerialBaud
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	c.UI.Autostart.Mode = normalizeAutostartMode(c.UI.Autostart.Mode)
	c.UI.MapViewport = normalizeMapViewport(c.UI.MapViewport)
}

func normalizeAutostartMode(mode AutostartMode) AutostartMode {
	switch mode {
	case AutostartModeBackground:
		return AutostartModeBackground
	default:
		return AutostartModeNormal
	}
}

func normalizeMapViewport(viewport MapViewportConfig) MapViewportConfig {
	if !viewport.Set {
		return MapViewportConfig{}
	}
	if viewport.Zoom < 0 {
		viewport.Zoom = 0
	}
	if viewport.Zoom > 19 {
		viewport.Zoom = 19
	}

	return viewport
}

func (c AppConfig) Validate() error {
	switch c.Connection.Connector {
	case ConnectorIP:
		if strings.TrimSpace(c.Connection.Host) == "" {
			return errors.New("ip host is required")
		}
	case ConnectorSerial:
		if strings.TrimSpace(c.Connection.SerialPort) == "" {
			return errors.New("serial port is required")
		}
		if c.Connection.SerialBaud <= 0 {
			return errors.New("serial baud must be positive")
		}
	case ConnectorBluetooth:
		if strings.TrimSpace(c.Connection.BluetoothAddress) == "" {
			return errors.New("bluetooth address is required")
		}
	default:
		return fmt.Errorf("unknown connector: %s", c.Connection.Connector)
	}

	return nil
}

func Save(path string, cfg AppConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp config: %w", err)
	}

	return nil
}
