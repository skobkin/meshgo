package app

import (
	"errors"
	"testing"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/platform"
)

func TestAutostartSyncWarningError(t *testing.T) {
	warning := &AutostartSyncWarning{Err: errors.New("boom")}
	if got := warning.Error(); got != "autostart sync failed: boom" {
		t.Fatalf("unexpected warning error text: %q", got)
	}
	if !errors.Is(warning, warning.Err) {
		t.Fatalf("expected warning to unwrap original error")
	}
}

func TestAutostartDevBuildSkipWarningError(t *testing.T) {
	warning := &AutostartDevBuildSkipWarning{Enabled: true}
	if got := warning.Error(); got != "autostart sync skipped in dev build: dev builds do not support autorun sync" {
		t.Fatalf("unexpected warning error text: %q", got)
	}
}

func TestRuntimeSyncAutostartSkipsInDevBuildOnStartup(t *testing.T) {
	originalVersion := Version
	Version = "main-123-gabcdef"
	t.Cleanup(func() {
		Version = originalVersion
	})

	manager := &stubAutostartManager{}
	rt := &Runtime{
		Core: RuntimeCore{
			AutostartManager: manager,
		},
	}
	cfg := config.Default()
	cfg.UI.Autostart.Enabled = true
	cfg.UI.Autostart.Mode = config.AutostartModeBackground

	if err := rt.syncAutostart(cfg, "startup"); err != nil {
		t.Fatalf("sync autostart: %v", err)
	}
	if got := len(manager.calls); got != 0 {
		t.Fatalf("expected no manager sync calls in dev build, got %d", got)
	}
}

func TestRuntimeSyncAutostartReturnsDevWarningOnSettingsSave(t *testing.T) {
	originalVersion := Version
	Version = "nightly-2026-02-15"
	t.Cleanup(func() {
		Version = originalVersion
	})

	manager := &stubAutostartManager{}
	rt := &Runtime{
		Core: RuntimeCore{
			AutostartManager: manager,
		},
	}
	cfg := config.Default()
	cfg.UI.Autostart.Enabled = true
	cfg.UI.Autostart.Mode = config.AutostartModeBackground

	err := rt.syncAutostart(cfg, "settings_save")
	var warning *AutostartDevBuildSkipWarning
	if !errors.As(err, &warning) {
		t.Fatalf("expected dev-build skip warning, got %v", err)
	}
	if !warning.Enabled {
		t.Fatalf("expected warning to carry enabled autostart state")
	}
	if got := len(manager.calls); got != 0 {
		t.Fatalf("expected no manager sync calls in dev build, got %d", got)
	}
}

func TestRuntimeSyncAutostartAppliesDisableOnSettingsSaveInDevBuild(t *testing.T) {
	originalVersion := Version
	Version = "feature-branch-build"
	t.Cleanup(func() {
		Version = originalVersion
	})

	manager := &stubAutostartManager{}
	rt := &Runtime{
		Core: RuntimeCore{
			AutostartManager: manager,
		},
	}
	cfg := config.Default()
	cfg.UI.Autostart.Enabled = false
	cfg.UI.Autostart.Mode = config.AutostartModeBackground

	if err := rt.syncAutostart(cfg, "settings_save"); err != nil {
		t.Fatalf("sync autostart: %v", err)
	}
	if got := len(manager.calls); got != 1 {
		t.Fatalf("expected one manager sync call in dev build when disabling, got %d", got)
	}
	if manager.calls[0].Enabled {
		t.Fatalf("expected synced autostart to be disabled")
	}
}

func TestRuntimeSyncAutostartCallsManagerOnReleaseBuild(t *testing.T) {
	originalVersion := Version
	Version = "0.12.3"
	t.Cleanup(func() {
		Version = originalVersion
	})

	manager := &stubAutostartManager{}
	rt := &Runtime{
		Core: RuntimeCore{
			AutostartManager: manager,
		},
	}
	cfg := config.Default()
	cfg.UI.Autostart.Enabled = true
	cfg.UI.Autostart.Mode = config.AutostartModeBackground

	if err := rt.syncAutostart(cfg, "settings_save"); err != nil {
		t.Fatalf("sync autostart: %v", err)
	}
	if got := len(manager.calls); got != 1 {
		t.Fatalf("expected one manager sync call, got %d", got)
	}
	if manager.calls[0].Mode != platform.AutostartModeBackground {
		t.Fatalf("unexpected synced autostart mode: %q", manager.calls[0].Mode)
	}
}

type stubAutostartManager struct {
	calls []platform.AutostartConfig
}

func (m *stubAutostartManager) Sync(cfg platform.AutostartConfig) error {
	m.calls = append(m.calls, cfg)

	return nil
}
