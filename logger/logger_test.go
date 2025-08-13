package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meshgo.log")
	l, c, err := New(path, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	l.Info("hello", "foo", "bar")
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(b), "hello") {
		t.Fatalf("log file missing entry: %s", string(b))
	}
}

func TestNewDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "meshgo.log")
	_, c, err := New(path, false)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no log file, got %v", err)
	}
}
