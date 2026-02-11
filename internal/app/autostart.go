package app

import (
	"fmt"
	"log/slog"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/platform"
)

// AutostartSyncWarning signals that config save succeeded but autostart sync failed.
type AutostartSyncWarning struct {
	Err error
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
	if r.AutostartManager == nil {
		slog.Debug("skip autostart sync: manager is not initialized", "trigger", trigger)

		return nil
	}

	mode := config.AutostartModeNormal
	if cfg.UI.Autostart.Enabled {
		mode = cfg.UI.Autostart.Mode
	}
	slog.Info("syncing autostart registration", "trigger", trigger, "enabled", cfg.UI.Autostart.Enabled, "mode", mode)

	if err := r.AutostartManager.Sync(platform.AutostartConfig{
		Enabled: cfg.UI.Autostart.Enabled,
		Mode:    platform.AutostartMode(cfg.UI.Autostart.Mode),
	}); err != nil {
		return err
	}

	slog.Info("autostart registration synced", "trigger", trigger, "enabled", cfg.UI.Autostart.Enabled, "mode", mode)

	return nil
}
