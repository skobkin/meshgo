package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TransportType identifies which transport backend should be used.
type TransportType string

// AutostartMode controls how the app is launched by OS autostart.
type AutostartMode string

// MapLinkProvider identifies which external map provider is used for location links.
type MapLinkProvider string

const (
	TransportIP        TransportType = "ip"
	TransportBluetooth TransportType = "bluetooth"
	TransportSerial    TransportType = "serial"
	DefaultSerialBaud                = 115200

	DefaultPositionHistoryLimit  = 100
	DefaultTelemetryHistoryLimit = 250
	DefaultIdentityHistoryLimit  = 50

	AutostartModeNormal     AutostartMode = "normal"
	AutostartModeBackground AutostartMode = "background"

	MapLinkProviderOpenStreetMap MapLinkProvider = "openstreetmap"
)

// LoggingConfig defines runtime logging behavior.
type LoggingConfig struct {
	Level     string `json:"level"`
	LogToFile bool   `json:"log_to_file"`
}

// ConnectionConfig contains transport-specific connection parameters.
type ConnectionConfig struct {
	Transport        TransportType `json:"transport"`
	Host             string        `json:"host"`
	SerialPort       string        `json:"serial_port"`
	SerialBaud       int           `json:"serial_baud"`
	BluetoothAddress string        `json:"bluetooth_address"`
	BluetoothAdapter string        `json:"bluetooth_adapter"`
	// Temporary feature gate: keep unfinished BLE transport hidden in UI by default
	// until Bluetooth support is stabilized (or removed).
	BluetoothTestingEnabled bool `json:"bluetooth_testing_enabled"`
}

// UIConfig stores persistent UI preferences.
type UIConfig struct {
	LastSelectedChat string             `json:"last_selected_chat"`
	Autostart        AutostartConfig    `json:"autostart"`
	MapViewport      MapViewportConfig  `json:"map_viewport"`
	MapDisplay       MapDisplayConfig   `json:"map_display"`
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

// MapDisplayConfig stores map overlay display preferences.
type MapDisplayConfig struct {
	ShowPrecisionCircles            bool            `json:"show_precision_circles"`
	ShowPrecisionCirclesOnlyOnHover bool            `json:"show_precision_circles_only_on_hover"`
	MapLinkProvider                 MapLinkProvider `json:"map_link_provider"`
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

// PersistenceConfig stores persistence behavior and retention settings.
type PersistenceConfig struct {
	HistoryLimits HistoryLimitsConfig `json:"history_limits"`
}

// HistoryLimitsConfig stores per-table node history row caps.
// Nil values mean "use default", zero means unlimited.
type HistoryLimitsConfig struct {
	Position  *int `json:"position"`
	Telemetry *int `json:"telemetry"`
	Identity  *int `json:"identity"`
}

// AppConfig is the root persisted application configuration.
type AppConfig struct {
	Connection  ConnectionConfig  `json:"connection"`
	Logging     LoggingConfig     `json:"logging"`
	Persistence PersistenceConfig `json:"persistence"`
	UI          UIConfig          `json:"ui"`
}

func Default() AppConfig {
	return AppConfig{
		Connection: ConnectionConfig{
			Transport:               TransportIP,
			Host:                    "",
			SerialPort:              "",
			SerialBaud:              DefaultSerialBaud,
			BluetoothAddress:        "",
			BluetoothAdapter:        "",
			BluetoothTestingEnabled: false,
		},
		Logging: LoggingConfig{
			Level:     "info",
			LogToFile: false,
		},
		Persistence: PersistenceConfig{
			HistoryLimits: defaultHistoryLimitsConfig(),
		},
		UI: UIConfig{
			LastSelectedChat: "",
			Autostart: AutostartConfig{
				Enabled: false,
				Mode:    AutostartModeNormal,
			},
			MapViewport: MapViewportConfig{},
			MapDisplay:  MapDisplayConfig{},
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

	// First unmarshal into the regular struct to get defaults for missing fields
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("decode config json: %w", err)
	}

	// Check for deprecated "connector" field and migrate if needed.
	cfg = migrateDeprecatedConnector(raw, cfg)

	cfg.FillMissingDefaults()

	return cfg, nil
}

func (c *AppConfig) FillMissingDefaults() {
	if c.Connection.Transport == "" {
		c.Connection.Transport = TransportIP
	}
	if c.Connection.Transport == TransportBluetooth {
		// Temporary migration behavior:
		// when legacy configs already use Bluetooth transport, treat testing flag as enabled.
		c.Connection.BluetoothTestingEnabled = true
	}
	if c.Connection.SerialBaud <= 0 {
		c.Connection.SerialBaud = DefaultSerialBaud
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	c.UI.Autostart.Mode = normalizeAutostartMode(c.UI.Autostart.Mode)
	c.UI.MapViewport = normalizeMapViewport(c.UI.MapViewport)
	c.UI.MapDisplay = normalizeMapDisplay(c.UI.MapDisplay)
	c.Persistence.HistoryLimits = normalizeHistoryLimitsConfig(c.Persistence.HistoryLimits)
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

func normalizeMapDisplay(display MapDisplayConfig) MapDisplayConfig {
	if !display.ShowPrecisionCircles {
		display.ShowPrecisionCirclesOnlyOnHover = false
	}
	if display.MapLinkProvider == "" {
		display.MapLinkProvider = MapLinkProviderOpenStreetMap
	}
	switch display.MapLinkProvider {
	case MapLinkProviderOpenStreetMap:
	default:
		display.MapLinkProvider = MapLinkProviderOpenStreetMap
	}

	return display
}

func defaultHistoryLimitsConfig() HistoryLimitsConfig {
	return HistoryLimitsConfig{
		Position:  intPtr(DefaultPositionHistoryLimit),
		Telemetry: intPtr(DefaultTelemetryHistoryLimit),
		Identity:  intPtr(DefaultIdentityHistoryLimit),
	}
}

func normalizeHistoryLimitsConfig(limits HistoryLimitsConfig) HistoryLimitsConfig {
	defaults := defaultHistoryLimitsConfig()
	if limits.Position == nil {
		limits.Position = intPtr(*defaults.Position)
	}
	if limits.Telemetry == nil {
		limits.Telemetry = intPtr(*defaults.Telemetry)
	}
	if limits.Identity == nil {
		limits.Identity = intPtr(*defaults.Identity)
	}

	return limits
}

func intPtr(v int) *int {
	value := v

	return &value
}

func (c AppConfig) Validate() error {
	switch c.Connection.Transport {
	case TransportIP:
		if strings.TrimSpace(c.Connection.Host) == "" {
			return errors.New("ip host is required")
		}
	case TransportSerial:
		if strings.TrimSpace(c.Connection.SerialPort) == "" {
			return errors.New("serial port is required")
		}
		if c.Connection.SerialBaud <= 0 {
			return errors.New("serial baud must be positive")
		}
	case TransportBluetooth:
		if strings.TrimSpace(c.Connection.BluetoothAddress) == "" {
			return errors.New("bluetooth address is required")
		}
	default:
		return fmt.Errorf("unknown transport: %s", c.Connection.Transport)
	}
	if c.Persistence.HistoryLimits.Position != nil && *c.Persistence.HistoryLimits.Position < 0 {
		return errors.New("position history limit must be non-negative")
	}
	if c.Persistence.HistoryLimits.Telemetry != nil && *c.Persistence.HistoryLimits.Telemetry < 0 {
		return errors.New("telemetry history limit must be non-negative")
	}
	if c.Persistence.HistoryLimits.Identity != nil && *c.Persistence.HistoryLimits.Identity < 0 {
		return errors.New("identity history limit must be non-negative")
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
