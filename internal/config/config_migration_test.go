package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadMigratesDeprecatedConnectorField verifies migration from old "connector" field to "transport".
//
// Deprecated: This test will be removed when migrateDeprecatedConnector is removed.
// Planned removal: approximately 2 months from Feb 2026.
func TestLoadMigratesDeprecatedConnectorField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with old "connector" field but no "transport" field
	oldConfig := `{
  "connection": {
    "connector": "serial",
    "serial_port": "/dev/ttyUSB0",
    "serial_baud": 115200
  }
}`
	if err := os.WriteFile(configPath, []byte(oldConfig), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify the old "connector" value was migrated to "transport"
	if cfg.Connection.Transport != TransportSerial {
		t.Fatalf("expected transport to be migrated to %q, got %q", TransportSerial, cfg.Connection.Transport)
	}
	if cfg.Connection.SerialPort != "/dev/ttyUSB0" {
		t.Fatalf("expected serial_port to be preserved, got %q", cfg.Connection.SerialPort)
	}
}

// TestLoadTransportTakesPrecedenceOverDeprecatedConnector verifies that "transport" field
// takes precedence over deprecated "connector" field when both are present.
//
// Deprecated: This test will be removed when migrateDeprecatedConnector is removed.
// Planned removal: approximately 2 months from Feb 2026.
func TestLoadTransportTakesPrecedenceOverDeprecatedConnector(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with both fields - transport should win
	mixedConfig := `{
  "connection": {
    "transport": "ip",
    "connector": "serial",
    "host": "192.168.1.100"
  }
}`
	if err := os.WriteFile(configPath, []byte(mixedConfig), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Transport field should take precedence
	if cfg.Connection.Transport != TransportIP {
		t.Fatalf("expected transport %q to take precedence over deprecated connector, got %q", TransportIP, cfg.Connection.Transport)
	}
}

// TestSaveDoesNotWriteDeprecatedConnectorField verifies that saving config removes
// the deprecated "connector" field and only writes the new "transport" field.
//
// Deprecated: This test will be removed when migrateDeprecatedConnector is removed.
// Planned removal: approximately 2 months from Feb 2026.
func TestSaveDoesNotWriteDeprecatedConnectorField(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with old "connector" field
	oldConfig := `{
  "connection": {
    "connector": "serial",
    "serial_port": "/dev/ttyUSB0",
    "serial_baud": 115200
  }
}`
	if err := os.WriteFile(configPath, []byte(oldConfig), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load the config (triggers migration)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Save the config back
	if err := Save(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Read the saved config and verify "connector" is gone
	//nolint:gosec // configPath is a temp file in test directory, not user-controlled
	savedData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	savedStr := string(savedData)
	if contains := containsSubstring(savedStr, `"connector"`); contains {
		t.Fatalf("saved config should not contain deprecated 'connector' field, got:\n%s", savedStr)
	}
	if !containsSubstring(savedStr, `"transport"`) {
		t.Fatalf("saved config should contain 'transport' field, got:\n%s", savedStr)
	}
}

// TestLoadWithInvalidConnectorValue verifies graceful handling of invalid connector values.
//
// Deprecated: This test will be removed when migrateDeprecatedConnector is removed.
// Planned removal: approximately 2 months from Feb 2026.
func TestLoadWithInvalidConnectorValue(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create config with invalid connector value
	invalidConfig := `{
  "connection": {
    "connector": "invalid_transport_type",
    "host": "192.168.1.1"
  }
}`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Should migrate the invalid value (validation happens later)
	if cfg.Connection.Transport != TransportType("invalid_transport_type") {
		t.Fatalf("expected transport to be migrated even with invalid value, got %q", cfg.Connection.Transport)
	}
}

// containsSubstring checks if substr exists within s.
// Deprecated: This helper is only used for migration tests and will be removed
// when migrateDeprecatedConnector is removed. Planned removal: ~2 months from Feb 2026.
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

// containsSubstringHelper is a helper for containsSubstring.
// Deprecated: This helper is only used for migration tests and will be removed
// when migrateDeprecatedConnector is removed. Planned removal: ~2 months from Feb 2026.
func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
