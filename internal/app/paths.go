package app

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	RootDir    string
	ConfigFile string
	DBFile     string
	LogFile    string
}

func ResolvePaths() (Paths, error) {
	cfgRoot, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve config dir: %w", err)
	}

	root := filepath.Join(cfgRoot, Name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Paths{}, fmt.Errorf("create app config dir: %w", err)
	}

	return Paths{
		RootDir:    root,
		ConfigFile: filepath.Join(root, ConfigFilename),
		DBFile:     filepath.Join(root, DBFilename),
		LogFile:    filepath.Join(root, LogFilename),
	}, nil
}
