//go:build !linux && !windows

package platform

import (
	"fmt"
	"runtime"
)

type unsupportedAutostartManager struct{}

func newAutostartManager() AutostartManager {
	return unsupportedAutostartManager{}
}

func (unsupportedAutostartManager) Sync(cfg AutostartConfig) error {
	cfg = normalizeAutostartConfig(cfg)
	if !cfg.Enabled {
		return nil
	}
	return fmt.Errorf("autostart is not supported on %s", runtime.GOOS)
}
