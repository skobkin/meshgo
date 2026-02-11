//go:build windows

package platform

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const windowsRunKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

type windowsAutostartManager struct{}

func newAutostartManager() AutostartManager {
	return windowsAutostartManager{}
}

func (windowsAutostartManager) Sync(cfg AutostartConfig) error {
	cfg = normalizeAutostartConfig(cfg)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsRunKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open autostart registry key: %w", err)
	}
	defer key.Close()

	if !cfg.Enabled {
		if err := key.DeleteValue(autostartEntryName); err != nil && !isWindowsValueNotFound(err) {
			return fmt.Errorf("remove autostart registry value: %w", err)
		}
		return nil
	}

	executable, args, err := buildLaunchCommand(cfg)
	if err != nil {
		return err
	}
	if err := key.SetStringValue(autostartEntryName, buildWindowsCommandLine(executable, args)); err != nil {
		return fmt.Errorf("set autostart registry value: %w", err)
	}
	return nil
}

func isWindowsValueNotFound(err error) bool {
	return errors.Is(err, registry.ErrNotExist) || errors.Is(err, syscall.Errno(2))
}
