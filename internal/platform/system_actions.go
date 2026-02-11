package platform

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// SystemActions provides OS-specific helpers triggered from the UI.
type SystemActions interface {
	OpenBluetoothSettings() error
}

func NewSystemActions() SystemActions {
	return newSystemActions()
}

type commandSpec struct {
	name string
	args []string
}

type commandStarter func(name string, args ...string) error

func openBluetoothSettingsForOS(goos string, start commandStarter) error {
	normalizedOS := strings.ToLower(strings.TrimSpace(goos))
	commands, err := bluetoothSettingsCommandsForOS(normalizedOS)
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		return fmt.Errorf("bluetooth settings are not supported on %s", normalizedOS)
	}

	slog.Info("opening bluetooth settings", "goos", normalizedOS, "attempts", len(commands))

	var errs []error
	for i, spec := range commands {
		attempt := i + 1
		if err := start(spec.name, spec.args...); err == nil {
			slog.Info("opened bluetooth settings", "goos", normalizedOS, "command", spec.name, "attempt", attempt)

			return nil
		} else {
			slog.Debug(
				"bluetooth settings command failed",
				"goos", normalizedOS,
				"command", spec.name,
				"args", spec.args,
				"attempt", attempt,
				"error", err,
			)
			errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
		}
	}

	joinedErr := errors.Join(errs...)
	slog.Warn("failed to open bluetooth settings", "goos", normalizedOS, "error", joinedErr)

	return joinedErr
}

func bluetoothSettingsCommandsForOS(goos string) ([]commandSpec, error) {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return windowsBluetoothSettingsCommands, nil
	case "linux":
		return linuxBluetoothSettingsCommands, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", goos)
	}
}

func startCommandDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	return cmd.Start()
}
