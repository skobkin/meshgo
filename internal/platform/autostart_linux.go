//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type linuxAutostartManager struct{}

func newAutostartManager() AutostartManager {
	return linuxAutostartManager{}
}

func (linuxAutostartManager) Sync(cfg AutostartConfig) error {
	cfg = normalizeAutostartConfig(cfg)

	desktopPath, err := linuxDesktopEntryPath()
	if err != nil {
		return err
	}

	if !cfg.Enabled {
		if err := os.Remove(desktopPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove autostart desktop entry: %w", err)
		}

		return nil
	}

	executable, args, err := buildLaunchCommand(cfg)
	if err != nil {
		return err
	}

	entry := renderLinuxDesktopEntry(desktopExecLine(executable, args))
	if err := writeFileAtomically(desktopPath, []byte(entry), 0o644); err != nil {
		return fmt.Errorf("write autostart desktop entry: %w", err)
	}

	return nil
}

func linuxDesktopEntryPath() (string, error) {
	cfgHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if cfgHome == "" {
		var err error
		cfgHome, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve user config dir: %w", err)
		}
	}

	return filepath.Join(filepath.Clean(cfgHome), "autostart", autostartEntryName+".desktop"), nil
}

func renderLinuxDesktopEntry(execLine string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Version=1.0
Name=meshgo
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, execLine)
}

func desktopExecLine(executable string, args []string) string {
	fields := make([]string, 0, 1+len(args))
	fields = append(fields, quoteDesktopExecArg(executable))
	for _, arg := range args {
		fields = append(fields, quoteDesktopExecArg(arg))
	}

	return strings.Join(fields, " ")
}

func quoteDesktopExecArg(arg string) string {
	escaped := strings.ReplaceAll(arg, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, `"`, `\\"`)

	return `"` + escaped + `"`
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create dir %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, autostartEntryName+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()

		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()

		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
