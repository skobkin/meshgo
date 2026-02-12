package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	defaultMapTileCacheSizeBytes = int64(200 * 1024 * 1024)
	defaultMapTileTimeout        = 15 * time.Second
)

type mapTileCacheTransport struct {
	base     http.RoundTripper
	cacheDir string
	maxBytes int64

	mu sync.Mutex
}

func newMapTileHTTPClient(cacheDir string, maxBytes int64) *http.Client {
	base := http.DefaultTransport
	if maxBytes <= 0 {
		maxBytes = defaultMapTileCacheSizeBytes
	}
	if cacheDir == "" {
		return &http.Client{
			Timeout: defaultMapTileTimeout,
		}
	}

	return &http.Client{
		Transport: &mapTileCacheTransport{
			base:     base,
			cacheDir: cacheDir,
			maxBytes: maxBytes,
		},
		Timeout: defaultMapTileTimeout,
	}
}

func (t *mapTileCacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.Method != http.MethodGet || req.URL == nil || req.URL.String() == "" {
		return t.baseRoundTripper().RoundTrip(req)
	}

	cachePath := t.cachePathForURL(req.URL.String())
	if data, ok := t.readCachedTile(cachePath); ok {
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"image/png"}},
			Body:          io.NopCloser(bytes.NewReader(data)),
			ContentLength: int64(len(data)),
			Request:       req,
		}, nil
	}

	resp, err := t.baseRoundTripper().RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return resp, nil
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, readErr
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		t.writeCachedTile(cachePath, body)
	}

	return resp, nil
}

func (t *mapTileCacheTransport) baseRoundTripper() http.RoundTripper {
	if t != nil && t.base != nil {
		return t.base
	}

	return http.DefaultTransport
}

func (t *mapTileCacheTransport) cachePathForURL(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	hash := hex.EncodeToString(sum[:])
	prefix := filepath.Join(hash[:2], hash[2:4])

	return filepath.Join(t.cacheDir, prefix, hash+".tile")
}

func (t *mapTileCacheTransport) readCachedTile(path string) ([]byte, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)

	return data, true
}

func (t *mapTileCacheTransport) writeCachedTile(path string, data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		_ = os.Remove(tmpPath)

		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)

		return
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now)

	t.evictIfNeededLocked()
}

func (t *mapTileCacheTransport) evictIfNeededLocked() {
	type cacheFile struct {
		path    string
		size    int64
		modTime time.Time
	}

	var (
		files     []cacheFile
		totalSize int64
	)

	_ = filepath.WalkDir(t.cacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d == nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".tile" {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		totalSize += info.Size()
		files = append(files, cacheFile{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime(),
		})

		return nil
	})

	if totalSize <= t.maxBytes {
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	for _, file := range files {
		if totalSize <= t.maxBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			continue
		}
		totalSize -= file.size
	}
}
