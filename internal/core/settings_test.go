package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if settings == nil {
		t.Fatal("DefaultSettings returned nil")
	}

	// Test connection defaults
	if settings.Connection.Type != "serial" {
		t.Errorf("Default connection type should be 'serial', got %s", settings.Connection.Type)
	}
	if settings.Connection.Serial.Baud != 115200 {
		t.Errorf("Default serial baud should be 115200, got %d", settings.Connection.Serial.Baud)
	}
	if settings.Connection.IP.Port != 4403 {
		t.Errorf("Default IP port should be 4403, got %d", settings.Connection.IP.Port)
	}

	// Test reconnect defaults
	if settings.Reconnect.InitialMillis != 1000 {
		t.Errorf("Default initial millis should be 1000, got %d", settings.Reconnect.InitialMillis)
	}
	if settings.Reconnect.Multiplier != 1.6 {
		t.Errorf("Default multiplier should be 1.6, got %f", settings.Reconnect.Multiplier)
	}

	// Test notifications defaults
	if !settings.Notifications.Enabled {
		t.Error("Default notifications should be enabled")
	}

	// Test logging defaults
	if settings.Logging.Enabled {
		t.Error("Default logging should be disabled")
	}
	if settings.Logging.Level != "info" {
		t.Errorf("Default log level should be 'info', got %s", settings.Logging.Level)
	}

	// Test UI defaults
	if settings.UI.StartMinimized {
		t.Error("Default start minimized should be false")
	}
	if settings.UI.Theme != "system" {
		t.Errorf("Default theme should be 'system', got %s", settings.UI.Theme)
	}
}

func TestNewConfigManager(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set temporary config dir
	oldConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if oldConfigDir == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", oldConfigDir)
		}
	}()

	cm, err := NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}

	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}

	// Check that config directory was created
	if _, err := os.Stat(cm.ConfigDir()); os.IsNotExist(err) {
		t.Error("Config directory was not created")
	}

	// Check that config file was created with defaults
	if _, err := os.Stat(cm.configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Check that settings are loaded
	settings := cm.Settings()
	if settings == nil {
		t.Error("Settings should not be nil")
	}
}

func TestConfigManager_LoadAndSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "config.json")

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: configFile,
		settings:   DefaultSettings(),
	}

	// Test saving
	err = cm.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created after Save")
	}

	// Modify settings and save again
	cm.settings.Connection.Type = "ip"
	cm.settings.Connection.IP.Host = "test.example.com"
	cm.settings.Notifications.Enabled = false

	err = cm.Save()
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// Create new ConfigManager and test loading
	cm2 := &ConfigManager{
		configDir:  tmpDir,
		configFile: configFile,
		settings:   DefaultSettings(),
	}

	err = cm2.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that loaded settings match what we saved
	if cm2.settings.Connection.Type != "ip" {
		t.Errorf("Connection type should be 'ip', got %s", cm2.settings.Connection.Type)
	}
	if cm2.settings.Connection.IP.Host != "test.example.com" {
		t.Errorf("IP host should be 'test.example.com', got %s", cm2.settings.Connection.IP.Host)
	}
	if cm2.settings.Notifications.Enabled {
		t.Error("Notifications should be disabled after loading")
	}
}

func TestConfigManager_LoadNonExistentFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "nonexistent.json")

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: configFile,
		settings:   DefaultSettings(),
	}

	// Loading non-existent file should not fail (uses defaults)
	err = cm.Load()
	if err != nil {
		t.Errorf("Load should not fail for non-existent file: %v", err)
	}
}

func TestConfigManager_LoadInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configFile := filepath.Join(tmpDir, "invalid.json")

	// Write invalid JSON
	err = os.WriteFile(configFile, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: configFile,
		settings:   DefaultSettings(),
	}

	// Loading invalid JSON should fail
	err = cm.Load()
	if err == nil {
		t.Error("Load should fail for invalid JSON")
	}
}

func TestConfigManager_UpdateConnection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
		settings:   DefaultSettings(),
	}

	err = cm.UpdateConnection("ip", "/dev/ttyUSB1", 9600, "192.168.1.100", 4404)
	if err != nil {
		t.Fatalf("UpdateConnection failed: %v", err)
	}

	settings := cm.Settings()
	if settings.Connection.Type != "ip" {
		t.Errorf("Connection type should be 'ip', got %s", settings.Connection.Type)
	}
	if settings.Connection.Serial.Port != "/dev/ttyUSB1" {
		t.Errorf("Serial port should be '/dev/ttyUSB1', got %s", settings.Connection.Serial.Port)
	}
	if settings.Connection.Serial.Baud != 9600 {
		t.Errorf("Serial baud should be 9600, got %d", settings.Connection.Serial.Baud)
	}
	if settings.Connection.IP.Host != "192.168.1.100" {
		t.Errorf("IP host should be '192.168.1.100', got %s", settings.Connection.IP.Host)
	}
	if settings.Connection.IP.Port != 4404 {
		t.Errorf("IP port should be 4404, got %d", settings.Connection.IP.Port)
	}
}

func TestConfigManager_UpdateNotifications(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
		settings:   DefaultSettings(),
	}

	err = cm.UpdateNotifications(false)
	if err != nil {
		t.Fatalf("UpdateNotifications failed: %v", err)
	}

	if cm.Settings().Notifications.Enabled {
		t.Error("Notifications should be disabled")
	}
}

func TestConfigManager_UpdateLogging(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
		settings:   DefaultSettings(),
	}

	err = cm.UpdateLogging(true, "debug")
	if err != nil {
		t.Fatalf("UpdateLogging failed: %v", err)
	}

	settings := cm.Settings()
	if !settings.Logging.Enabled {
		t.Error("Logging should be enabled")
	}
	if settings.Logging.Level != "debug" {
		t.Errorf("Log level should be 'debug', got %s", settings.Logging.Level)
	}
}

func TestConfigManager_UpdateUI(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
		settings:   DefaultSettings(),
	}

	err = cm.UpdateUI(true, "dark")
	if err != nil {
		t.Fatalf("UpdateUI failed: %v", err)
	}

	settings := cm.Settings()
	if !settings.UI.StartMinimized {
		t.Error("Start minimized should be enabled")
	}
	if settings.UI.Theme != "dark" {
		t.Errorf("Theme should be 'dark', got %s", settings.UI.Theme)
	}
}

func TestConfigManager_UpdateConnectOnStartup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir:  tmpDir,
		configFile: filepath.Join(tmpDir, "config.json"),
		settings:   DefaultSettings(),
	}

	err = cm.UpdateConnectOnStartup(true)
	if err != nil {
		t.Fatalf("UpdateConnectOnStartup failed: %v", err)
	}

	if !cm.Settings().Connection.ConnectOnStartup {
		t.Error("ConnectOnStartup should be enabled")
	}
}

func TestConfigManager_LogDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir: tmpDir,
	}

	logDir := cm.LogDir()
	expectedLogDir := filepath.Join(tmpDir, "logs")

	if logDir != expectedLogDir {
		t.Errorf("LogDir should be %s, got %s", expectedLogDir, logDir)
	}
}

func TestConfigManager_EnsureLogDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "meshgo_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cm := &ConfigManager{
		configDir: tmpDir,
	}

	err = cm.EnsureLogDir()
	if err != nil {
		t.Fatalf("EnsureLogDir failed: %v", err)
	}

	logDir := cm.LogDir()
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}
}

func TestSettingsJSONSerialization(t *testing.T) {
	settings := DefaultSettings()

	// Modify some values
	settings.Connection.Type = "ip"
	settings.Connection.IP.Host = "test.example.com"
	settings.Notifications.Enabled = false

	// Serialize to JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Deserialize from JSON
	var deserializedSettings Settings
	err = json.Unmarshal(data, &deserializedSettings)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Compare values
	if deserializedSettings.Connection.Type != settings.Connection.Type {
		t.Errorf("Connection type mismatch after JSON round-trip")
	}
	if deserializedSettings.Connection.IP.Host != settings.Connection.IP.Host {
		t.Errorf("IP host mismatch after JSON round-trip")
	}
	if deserializedSettings.Notifications.Enabled != settings.Notifications.Enabled {
		t.Errorf("Notifications enabled mismatch after JSON round-trip")
	}
}

func TestConfigManager_ConfigDir(t *testing.T) {
	testDir := "/test/config/dir"
	cm := &ConfigManager{
		configDir: testDir,
	}

	if cm.ConfigDir() != testDir {
		t.Errorf("ConfigDir() should return %s, got %s", testDir, cm.ConfigDir())
	}
}

func TestConfigManager_Settings(t *testing.T) {
	settings := DefaultSettings()
	cm := &ConfigManager{
		settings: settings,
	}

	returnedSettings := cm.Settings()
	if returnedSettings != settings {
		t.Error("Settings() should return the same settings instance")
	}
}
