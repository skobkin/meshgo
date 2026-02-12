package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_ResolvesConfigAndCacheDirectories(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "cfg")
	cacheHome := filepath.Join(t.TempDir(), "cache")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatalf("resolve paths: %v", err)
	}

	if paths.RootDir != filepath.Join(configHome, Name) {
		t.Fatalf("unexpected root dir: %q", paths.RootDir)
	}
	if paths.CacheDir != filepath.Join(cacheHome, Name) {
		t.Fatalf("unexpected cache dir: %q", paths.CacheDir)
	}
	if paths.MapTilesDir != filepath.Join(cacheHome, Name, MapTilesDir) {
		t.Fatalf("unexpected map tiles dir: %q", paths.MapTilesDir)
	}
	if _, err := os.Stat(paths.MapTilesDir); err != nil {
		t.Fatalf("expected map tiles directory to exist: %v", err)
	}
}
