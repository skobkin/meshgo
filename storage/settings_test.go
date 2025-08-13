package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := &Settings{}
	want.Connection.Type = "serial"
	want.Connection.Serial.Port = "COM3"
	want.Connection.Serial.Baud = 9600
	want.Reconnect.InitialMillis = 1000
	want.Reconnect.MaxMillis = 60000
	want.Reconnect.Multiplier = 1.5
	want.Reconnect.Jitter = 0.2
	want.Notifications.Enabled = true
	want.Logging.Enabled = true
	want.UI.StartMinimized = true

	if err := SaveSettings(path, want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	got, err := LoadSettings(path)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestLoadSettingsMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	s, err := LoadSettings(path)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.IsNotExist, got %v", err)
	}
	if s != nil {
		t.Fatalf("expected nil settings, got %v", s)
	}
}
