package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths stores resolved runtime file locations for user config, logs, and cache.
type Paths struct {
	RootDir     string
	ConfigFile  string
	DBFile      string
	LogFile     string
	CacheDir    string
	MapTilesDir string
}

func ResolvePaths() (Paths, error) {
	cfgRoot, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve config dir: %w", err)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve cache dir: %w", err)
	}

	root := filepath.Join(cfgRoot, Name)
	if err := os.MkdirAll(root, 0o750); err != nil {
		return Paths{}, fmt.Errorf("create app config dir: %w", err)
	}
	cache := filepath.Join(cacheRoot, Name)
	if err := os.MkdirAll(cache, 0o750); err != nil {
		return Paths{}, fmt.Errorf("create app cache dir: %w", err)
	}
	mapTiles := filepath.Join(cache, MapTilesDir)
	if err := os.MkdirAll(mapTiles, 0o750); err != nil {
		return Paths{}, fmt.Errorf("create map tile cache dir: %w", err)
	}

	return Paths{
		RootDir:     root,
		ConfigFile:  filepath.Join(root, ConfigFilename),
		DBFile:      filepath.Join(root, DBFilename),
		LogFile:     filepath.Join(root, LogFilename),
		CacheDir:    cache,
		MapTilesDir: mapTiles,
	}, nil
}
