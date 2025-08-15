package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Settings struct {
	Connection    ConnectionSettings    `json:"connection"`
	Reconnect     ReconnectSettings     `json:"reconnect"`
	Notifications NotificationSettings  `json:"notifications"`
	Logging       LoggingSettings       `json:"logging"`
	UI            UISettings            `json:"ui"`
}

type ConnectionSettings struct {
	Type            string         `json:"type"` // "serial" or "ip"
	ConnectOnStartup bool          `json:"connect_on_startup"`
	Serial          SerialSettings `json:"serial"`
	IP              IPSettings     `json:"ip"`
}

type SerialSettings struct {
	Port string `json:"port"`
	Baud int    `json:"baud"`
}

type IPSettings struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ReconnectSettings struct {
	InitialMillis int     `json:"initial_millis"`
	MaxMillis     int     `json:"max_millis"`
	Multiplier    float64 `json:"multiplier"`
	Jitter        float64 `json:"jitter"`
}

type NotificationSettings struct {
	Enabled bool `json:"enabled"`
}

type LoggingSettings struct {
	Enabled bool   `json:"enabled"`
	Level   string `json:"level"`
}

type UISettings struct {
	StartMinimized bool   `json:"start_minimized"`
	Theme          string `json:"theme"`
}

func DefaultSettings() *Settings {
	return &Settings{
		Connection: ConnectionSettings{
			Type: "serial",
			Serial: SerialSettings{
				Port: "",
				Baud: 115200,
			},
			IP: IPSettings{
				Host: "192.168.1.1",
				Port: 4403,
			},
		},
		Reconnect: ReconnectSettings{
			InitialMillis: 1000,   // 1 second
			MaxMillis:     60000,  // 60 seconds
			Multiplier:    1.6,
			Jitter:        0.2,    // ±20%
		},
		Notifications: NotificationSettings{
			Enabled: true,
		},
		Logging: LoggingSettings{
			Enabled: false,
			Level:   "info",
		},
		UI: UISettings{
			StartMinimized: false,
			Theme:          "system",
		},
	}
}

type ConfigManager struct {
	configDir  string
	configFile string
	settings   *Settings
}

func NewConfigManager() (*ConfigManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}

	meshgoDir := filepath.Join(configDir, "meshgo")
	if err := os.MkdirAll(meshgoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(meshgoDir, "config.json")

	cm := &ConfigManager{
		configDir:  meshgoDir,
		configFile: configFile,
		settings:   DefaultSettings(),
	}

	// Load existing config if it exists
	if err := cm.Load(); err != nil {
		// If load fails, use defaults and save them
		if err := cm.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	return cm, nil
}

func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	settings := DefaultSettings()
	if err := json.Unmarshal(data, settings); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	cm.settings = settings
	return nil
}

func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (cm *ConfigManager) Settings() *Settings {
	return cm.settings
}

func (cm *ConfigManager) UpdateConnection(connType string, serialPort string, serialBaud int, ipHost string, ipPort int) error {
	cm.settings.Connection.Type = connType
	cm.settings.Connection.Serial.Port = serialPort
	cm.settings.Connection.Serial.Baud = serialBaud
	cm.settings.Connection.IP.Host = ipHost
	cm.settings.Connection.IP.Port = ipPort
	return cm.Save()
}

func (cm *ConfigManager) UpdateConnectOnStartup(enabled bool) error {
	cm.settings.Connection.ConnectOnStartup = enabled
	return cm.Save()
}

func (cm *ConfigManager) UpdateNotifications(enabled bool) error {
	cm.settings.Notifications.Enabled = enabled
	return cm.Save()
}

func (cm *ConfigManager) UpdateLogging(enabled bool, level string) error {
	cm.settings.Logging.Enabled = enabled
	cm.settings.Logging.Level = level
	return cm.Save()
}

func (cm *ConfigManager) UpdateUI(startMinimized bool, theme string) error {
	cm.settings.UI.StartMinimized = startMinimized
	cm.settings.UI.Theme = theme
	return cm.Save()
}

func (cm *ConfigManager) ConfigDir() string {
	return cm.configDir
}

func (cm *ConfigManager) LogDir() string {
	return filepath.Join(cm.configDir, "logs")
}

func (cm *ConfigManager) EnsureLogDir() error {
	logDir := cm.LogDir()
	return os.MkdirAll(logDir, 0755)
}