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

type LaunchMode string

const (
	LaunchModeNormal     LaunchMode = "normal"
	LaunchModeBackground LaunchMode = "background"
)

type AutostartConfig struct {
	Enabled bool
	Mode    LaunchMode
}

type AutostartManager interface {
	Sync(cfg AutostartConfig) error
}

func NewAutostartManager() AutostartManager {
	return newAutostartManager()
}

func normalizeLaunchMode(mode LaunchMode) LaunchMode {
	switch mode {
	case LaunchModeBackground:
		return LaunchModeBackground
	default:
		return LaunchModeNormal
	}
}

func normalizeAutostartConfig(cfg AutostartConfig) AutostartConfig {
	cfg.Mode = normalizeLaunchMode(cfg.Mode)
	return cfg
}

func launchArgsForMode(mode LaunchMode) []string {
	if normalizeLaunchMode(mode) == LaunchModeBackground {
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
