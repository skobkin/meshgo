//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxAutostartSyncWritesUpdatesAndRemovesDesktopEntry(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	mgr := newAutostartManager()
	if err := mgr.Sync(AutostartConfig{Enabled: true, Mode: AutostartModeNormal}); err != nil {
		t.Fatalf("sync normal mode: %v", err)
	}

	entryPath, err := linuxDesktopEntryPath()
	if err != nil {
		t.Fatalf("resolve desktop path: %v", err)
	}
	// #nosec G304 -- test controls XDG_CONFIG_HOME and entry path.
	raw, err := os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read desktop entry: %v", err)
	}
	entry := string(raw)
	if !strings.Contains(entry, "[Desktop Entry]") {
		t.Fatalf("expected desktop entry header, got %q", entry)
	}
	if strings.Contains(entry, startHiddenArg) {
		t.Fatalf("did not expect %q in normal mode entry", startHiddenArg)
	}

	if err := mgr.Sync(AutostartConfig{Enabled: true, Mode: AutostartModeBackground}); err != nil {
		t.Fatalf("sync background mode: %v", err)
	}
	// #nosec G304 -- test controls XDG_CONFIG_HOME and entry path.
	raw, err = os.ReadFile(entryPath)
	if err != nil {
		t.Fatalf("read updated desktop entry: %v", err)
	}
	if !strings.Contains(string(raw), startHiddenArg) {
		t.Fatalf("expected %q in background mode entry, got %q", startHiddenArg, string(raw))
	}

	if err := mgr.Sync(AutostartConfig{Enabled: false, Mode: AutostartModeBackground}); err != nil {
		t.Fatalf("disable autostart: %v", err)
	}
	if _, err := os.Stat(entryPath); !os.IsNotExist(err) {
		t.Fatalf("expected desktop entry to be removed, stat err: %v", err)
	}
}

func TestLinuxDesktopEntryPathUsesXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	path, err := linuxDesktopEntryPath()
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}

	want := filepath.Join(root, "autostart", "meshgo.desktop")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestDesktopExecLine(t *testing.T) {
	line := desktopExecLine("/opt/meshgo/bin/meshgo", []string{"--start-hidden"})
	if !strings.Contains(line, `"/opt/meshgo/bin/meshgo"`) {
		t.Fatalf("expected quoted executable, got %q", line)
	}
	if !strings.Contains(line, `"--start-hidden"`) {
		t.Fatalf("expected quoted arg, got %q", line)
	}
}
