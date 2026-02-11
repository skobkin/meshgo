package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type commandSpec struct {
	name string
	args []string
}

type commandStarter func(name string, args ...string) error

func openBluetoothSettings() error {
	return openBluetoothSettingsForOS(runtime.GOOS, startCommandDetached)
}

func openBluetoothSettingsForOS(goos string, start commandStarter) error {
	commands, err := bluetoothSettingsCommands(goos)
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		return fmt.Errorf("bluetooth settings are not supported on %s", goos)
	}

	var errs []error
	for _, spec := range commands {
		if err := start(spec.name, spec.args...); err == nil {
			return nil
		} else {
			errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
		}
	}

	return errors.Join(errs...)
}

func bluetoothSettingsCommands(goos string) ([]commandSpec, error) {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "windows":
		return []commandSpec{
			{name: "cmd", args: []string{"/c", "start", "", "ms-settings:bluetooth"}},
		}, nil
	case "linux":
		return []commandSpec{
			{name: "systemsettings6", args: []string{"kcm_bluetooth"}},
			{name: "systemsettings5", args: []string{"kcm_bluetooth"}},
			{name: "systemsettings", args: []string{"kcm_bluetooth"}},
			{name: "gnome-control-center", args: []string{"bluetooth"}},
			{name: "kcmshell6", args: []string{"kcm_bluetooth"}},
			{name: "kcmshell5", args: []string{"kcm_bluetooth"}},
			{name: "kcmshell4", args: []string{"bluetooth"}},
			{name: "blueman-manager"},
			{name: "xdg-open", args: []string{"bluetooth://"}},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", goos)
	}
}

func startCommandDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Start()
}
