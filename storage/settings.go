package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings represents user-configurable application settings persisted to disk.
type Settings struct {
	Connection struct {
		Type   string `json:"type"`
		Serial struct {
			Port string `json:"port"`
			Baud int    `json:"baud"`
		} `json:"serial"`
		IP struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"ip"`
	} `json:"connection"`
	Reconnect struct {
		InitialMillis int     `json:"initialMillis"`
		MaxMillis     int     `json:"maxMillis"`
		Multiplier    float64 `json:"multiplier"`
		Jitter        float64 `json:"jitter"`
	} `json:"reconnect"`
	Notifications struct {
		Enabled bool `json:"enabled"`
	} `json:"notifications"`
	Logging struct {
		Enabled bool `json:"enabled"`
	} `json:"logging"`
	UI struct {
		StartMinimized bool `json:"startMinimized"`
	} `json:"ui"`
}

// LoadSettings reads the settings file from disk. If the file does not exist it
// returns nil and os.ErrNotExist.
func LoadSettings(path string) (*Settings, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSettings writes the settings structure to the specified file, creating
// parent directories as needed.
func SaveSettings(path string, s *Settings) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
