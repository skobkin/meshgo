package app

import (
	"fmt"
	"log/slog"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/platform"
	"golang.org/x/mod/semver"
)

// AutostartSyncWarning signals that config save succeeded but autostart sync failed.
type AutostartSyncWarning struct {
	Err error
}

// AutostartDevBuildSkipWarning signals that autostart sync was intentionally skipped in dev builds.
type AutostartDevBuildSkipWarning struct {
	Enabled bool
}

func (w *AutostartDevBuildSkipWarning) Error() string {
	if w == nil {
		return "autostart sync skipped in dev build"
	}
	if w.Enabled {
		return "autostart sync skipped in dev build: dev builds do not support autorun sync"
	}

	return "autostart sync skipped in dev build"
}

func (w *AutostartSyncWarning) Error() string {
	if w == nil || w.Err == nil {
		return "autostart sync failed"
	}

	return fmt.Sprintf("autostart sync failed: %v", w.Err)
}

func (w *AutostartSyncWarning) Unwrap() error {
	if w == nil {
		return nil
	}

	return w.Err
}

func (r *Runtime) syncAutostart(cfg config.AppConfig, trigger string) error {
	if r.Core.AutostartManager == nil {
		slog.Debug("skip autostart sync: manager is not initialized", "trigger", trigger)

		return nil
	}

	mode := config.AutostartModeNormal
	if cfg.UI.Autostart.Enabled {
		mode = cfg.UI.Autostart.Mode
	}

	version := BuildVersion()
	if !semver.IsValid(normalizeSemver(version)) {
		allowDisableSync := trigger == "settings_save" && !cfg.UI.Autostart.Enabled
		if !allowDisableSync {
			slog.Info(
				"skip autostart sync in dev build",
				"trigger", trigger,
				"version", version,
				"enabled", cfg.UI.Autostart.Enabled,
				"mode", mode,
			)
			if trigger == "settings_save" && cfg.UI.Autostart.Enabled {
				return &AutostartDevBuildSkipWarning{Enabled: true}
			}

			return nil
		}

		slog.Info(
			"sync autostart disable in dev build",
			"trigger", trigger,
			"version", version,
			"enabled", cfg.UI.Autostart.Enabled,
			"mode", mode,
		)
	}

	slog.Info("syncing autostart registration", "trigger", trigger, "enabled", cfg.UI.Autostart.Enabled, "mode", mode)

	if err := r.Core.AutostartManager.Sync(platform.AutostartConfig{
		Enabled: cfg.UI.Autostart.Enabled,
		Mode:    platform.AutostartMode(cfg.UI.Autostart.Mode),
	}); err != nil {
		return err
	}

	slog.Info("autostart registration synced", "trigger", trigger, "enabled", cfg.UI.Autostart.Enabled, "mode", mode)

	return nil
}
