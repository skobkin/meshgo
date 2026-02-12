package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeClearCache_RemovesAndRecreatesCacheDirs(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("LOCALAPPDATA", cacheHome)

	cacheDir := filepath.Join(cacheHome, Name)
	mapTilesDir := filepath.Join(cacheDir, MapTilesDir)
	if err := os.MkdirAll(filepath.Join(mapTilesDir, "nested"), 0o750); err != nil {
		t.Fatalf("create cache fixture dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "old.bin"), []byte("stale"), 0o600); err != nil {
		t.Fatalf("create cache fixture file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mapTilesDir, "nested", "tile.png"), []byte("tile"), 0o600); err != nil {
		t.Fatalf("create tile cache fixture file: %v", err)
	}

	rt := &Runtime{
		Core: RuntimeCore{
			Paths: Paths{
				CacheDir:    cacheDir,
				MapTilesDir: mapTilesDir,
			},
		},
	}

	if err := rt.ClearCache(); err != nil {
		t.Fatalf("clear cache: %v", err)
	}

	if info, err := os.Stat(cacheDir); err != nil || !info.IsDir() {
		t.Fatalf("expected cache dir to exist after clear, err=%v", err)
	}
	if info, err := os.Stat(mapTilesDir); err != nil || !info.IsDir() {
		t.Fatalf("expected map tiles dir to exist after clear, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "old.bin")); !os.IsNotExist(err) {
		t.Fatalf("expected stale cache file to be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(mapTilesDir, "nested", "tile.png")); !os.IsNotExist(err) {
		t.Fatalf("expected stale tile file to be removed, err=%v", err)
	}
}

func TestRuntimeClearCache_EmptyPathFails(t *testing.T) {
	rt := &Runtime{Core: RuntimeCore{Paths: Paths{}}}

	err := rt.ClearCache()
	if err == nil {
		t.Fatalf("expected clear cache to fail for empty cache dir")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeClearCache_OutsideAppCacheDirFails(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("LOCALAPPDATA", cacheHome)

	outsideDir := filepath.Join(t.TempDir(), "outside-cache")
	if err := os.MkdirAll(outsideDir, 0o750); err != nil {
		t.Fatalf("create outside cache dir: %v", err)
	}
	sentinel := filepath.Join(outsideDir, "do-not-remove.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatalf("create outside cache sentinel: %v", err)
	}

	rt := &Runtime{
		Core: RuntimeCore{
			Paths: Paths{
				CacheDir: outsideDir,
			},
		},
	}

	err := rt.ClearCache()
	if err == nil {
		t.Fatalf("expected clear cache to fail for outside cache dir")
	}
	if !strings.Contains(err.Error(), "outside app cache dir") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(sentinel); statErr != nil {
		t.Fatalf("expected sentinel to remain untouched, err=%v", statErr)
	}
}

func TestRuntimeClearCache_MapTilesOutsideCacheDirFails(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("LOCALAPPDATA", cacheHome)

	cacheDir := filepath.Join(cacheHome, Name)
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		t.Fatalf("create app cache dir: %v", err)
	}
	sentinel := filepath.Join(cacheDir, "do-not-remove.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatalf("create app cache sentinel: %v", err)
	}
	outsideMapTilesDir := filepath.Join(t.TempDir(), "outside-tiles")

	rt := &Runtime{
		Core: RuntimeCore{
			Paths: Paths{
				CacheDir:    cacheDir,
				MapTilesDir: outsideMapTilesDir,
			},
		},
	}

	err := rt.ClearCache()
	if err == nil {
		t.Fatalf("expected clear cache to fail when map tiles dir is outside cache")
	}
	if !strings.Contains(err.Error(), "outside cache dir") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(sentinel); statErr != nil {
		t.Fatalf("expected app cache sentinel to remain untouched, err=%v", statErr)
	}
}
