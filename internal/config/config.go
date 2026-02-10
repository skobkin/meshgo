package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type ConnectorType string

const (
	ConnectorIP        ConnectorType = "ip"
	ConnectorBluetooth ConnectorType = "bluetooth"
	ConnectorSerial    ConnectorType = "serial"
)

type LoggingConfig struct {
	Level     string `json:"level"`
	LogToFile bool   `json:"log_to_file"`
}

type ConnectionConfig struct {
	Connector ConnectorType `json:"connector"`
	Host      string        `json:"host"`
}

type UIConfig struct {
	LastSelectedChat string `json:"last_selected_chat"`
}

type AppConfig struct {
	Connection ConnectionConfig `json:"connection"`
	Logging    LoggingConfig    `json:"logging"`
	UI         UIConfig         `json:"ui"`
}

func Default() AppConfig {
	return AppConfig{
		Connection: ConnectionConfig{
			Connector: ConnectorIP,
			Host:      "",
		},
		Logging: LoggingConfig{
			Level:     "info",
			LogToFile: false,
		},
		UI: UIConfig{
			LastSelectedChat: "",
		},
	}
}

func Load(path string) (AppConfig, error) {
	cfg := Default()
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("decode config json: %w", err)
	}

	cfg.ApplyDefaults()
	return cfg, nil
}

func (c *AppConfig) ApplyDefaults() {
	if c.Connection.Connector == "" {
		c.Connection.Connector = ConnectorIP
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
}

func (c AppConfig) Validate() error {
	switch c.Connection.Connector {
	case ConnectorIP, ConnectorBluetooth, ConnectorSerial:
	default:
		return fmt.Errorf("unknown connector: %s", c.Connection.Connector)
	}
	return nil
}

func Save(path string, cfg AppConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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
