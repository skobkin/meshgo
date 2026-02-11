package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	autostartEntryName = "meshgo"
	startHiddenArg     = "--start-hidden"
)

// AutostartMode selects how the app is started by the operating system.
type AutostartMode string

const (
	AutostartModeNormal     AutostartMode = "normal"
	AutostartModeBackground AutostartMode = "background"
)

// AutostartConfig represents desired autostart registration state.
type AutostartConfig struct {
	Enabled bool
	Mode    AutostartMode
}

// AutostartManager updates platform autostart registration for the current user.
type AutostartManager interface {
	Sync(cfg AutostartConfig) error
}

func NewAutostartManager() AutostartManager {
	return newAutostartManager()
}

func normalizeAutostartMode(mode AutostartMode) AutostartMode {
	switch mode {
	case AutostartModeBackground:
		return AutostartModeBackground
	default:
		return AutostartModeNormal
	}
}

func normalizeAutostartConfig(cfg AutostartConfig) AutostartConfig {
	cfg.Mode = normalizeAutostartMode(cfg.Mode)

	return cfg
}

func launchArgsForMode(mode AutostartMode) []string {
	if normalizeAutostartMode(mode) == AutostartModeBackground {
		return []string{startHiddenArg}
	}

	return nil
}

func buildLaunchCommand(cfg AutostartConfig) (string, []string, error) {
	executable, err := resolveExecutablePath()
	if err != nil {
		return "", nil, err
	}

	return executable, launchArgsForMode(cfg.Mode), nil
}

func resolveExecutablePath() (string, error) {
	rawPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("resolve executable path: path is empty")
	}
	if !filepath.IsAbs(trimmed) {
		trimmed, err = filepath.Abs(trimmed)
		if err != nil {
			return "", fmt.Errorf("resolve executable absolute path: %w", err)
		}
	}

	if resolved, err := filepath.EvalSymlinks(trimmed); err == nil {
		trimmed = resolved
	}

	return filepath.Clean(trimmed), nil
}
