package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMapTileCacheTransport_CachesTilesOnDisk(t *testing.T) {
	cacheDir := t.TempDir()
	var hits int
	tile := mustPNGBytes(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(tile)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: &mapTileCacheTransport{
			base:     http.DefaultTransport,
			cacheDir: cacheDir,
			maxBytes: 1024 * 1024,
		},
	}

	for i := 0; i < 2; i++ {
		resp, err := client.Get(server.URL + "/0/0/0.png")
		if err != nil {
			t.Fatalf("get tile: %v", err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !bytes.Equal(body, tile) {
			t.Fatalf("unexpected tile response body")
		}
	}

	if hits != 1 {
		t.Fatalf("expected one network hit after cache warmup, got %d", hits)
	}
}

func TestMapTileCacheTransport_EvictsOldestFilesWhenSizeCapExceeded(t *testing.T) {
	cacheDir := t.TempDir()
	transport := &mapTileCacheTransport{
		base:     http.DefaultTransport,
		cacheDir: cacheDir,
		maxBytes: 20,
	}

	pathA := transport.cachePathForURL("https://tile.example/1")
	pathB := transport.cachePathForURL("https://tile.example/2")

	transport.writeCachedTile(pathA, []byte("123456789012"))
	time.Sleep(10 * time.Millisecond)
	transport.writeCachedTile(pathB, []byte("abcdefghijk"))

	var tileFiles []string
	_ = filepath.WalkDir(cacheDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && d != nil && !d.IsDir() && filepath.Ext(path) == ".tile" {
			tileFiles = append(tileFiles, path)
		}

		return nil
	})
	if len(tileFiles) == 0 {
		t.Fatalf("expected at least one cached tile")
	}

	var totalSize int64
	for _, path := range tileFiles {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat tile file: %v", err)
		}
		totalSize += info.Size()
	}
	if totalSize > transport.maxBytes {
		t.Fatalf("expected cache size <= %d, got %d", transport.maxBytes, totalSize)
	}
	if _, err := os.Stat(pathA); err == nil {
		t.Fatalf("expected oldest tile to be evicted")
	}
	if _, err := os.Stat(pathB); err != nil {
		t.Fatalf("expected newest tile to remain cached: %v", err)
	}
}

func mustPNGBytes(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return buf.Bytes()
}
