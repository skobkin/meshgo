package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
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
	mapTileFetchModeHeader       = "X-Meshgo-Tile-Fetch-Mode"
	mapTileFetchModeSync         = "sync"
)

var mapTileCacheLogger = slog.With("component", "ui.map_tile_cache")

type mapTileCacheTransport struct {
	base      http.RoundTripper
	cacheDir  string
	maxBytes  int64
	asyncMiss bool

	mu                sync.Mutex
	inFlightByPath    map[string]struct{}
	onAsyncTileCached func()
}

func newMapTileHTTPClient(cacheDir string, maxBytes int64) *http.Client {
	base := http.DefaultTransport
	if maxBytes <= 0 {
		maxBytes = defaultMapTileCacheSizeBytes
	}
	mapTileCacheLogger.Info(
		"initializing map tile HTTP client",
		"cache_enabled", cacheDir != "",
		"cache_dir", cacheDir,
		"max_bytes", maxBytes,
		"timeout", defaultMapTileTimeout,
	)
	if cacheDir == "" {
		return &http.Client{
			Timeout: defaultMapTileTimeout,
		}
	}

	return &http.Client{
		Transport: &mapTileCacheTransport{
			base:           base,
			cacheDir:       cacheDir,
			maxBytes:       maxBytes,
			asyncMiss:      true,
			inFlightByPath: make(map[string]struct{}),
		},
		Timeout: defaultMapTileTimeout,
	}
}

func (t *mapTileCacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.Method != http.MethodGet || req.URL == nil || req.URL.String() == "" {
		return t.baseRoundTripper().RoundTrip(req)
	}
	startedAt := time.Now()
	syncFetch := req.Header.Get(mapTileFetchModeHeader) == mapTileFetchModeSync

	cachePath := t.cachePathForURL(req.URL.String())
	if data, ok := t.readCachedTile(cachePath); ok {
		mapTileCacheLogger.Debug(
			"served map tile from cache",
			"cache_path", cachePath,
			"bytes", len(data),
			"duration", time.Since(startedAt),
		)

		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Header:        http.Header{"Content-Type": []string{"image/png"}},
			Body:          io.NopCloser(bytes.NewReader(data)),
			ContentLength: int64(len(data)),
			Request:       req,
		}, nil
	}

	if t.asyncMiss && !syncFetch {
		started := t.startAsyncFetch(cachePath, req.URL.String())
		mapTileCacheLogger.Debug(
			"deferred uncached map tile to async fetch",
			"url", req.URL.String(),
			"cache_path", cachePath,
			"async_fetch_started", started,
			"duration", time.Since(startedAt),
		)

		return &http.Response{
			StatusCode:    http.StatusAccepted,
			Status:        "202 Accepted",
			Header:        http.Header{"Content-Type": []string{"application/octet-stream"}},
			Body:          io.NopCloser(bytes.NewReader(nil)),
			ContentLength: 0,
			Request:       req,
		}, nil
	}

	resp, err := t.baseRoundTripper().RoundTrip(req)
	if err != nil {
		mapTileCacheLogger.Warn(
			"map tile request failed",
			"url", req.URL.String(),
			"error", err,
			"duration", time.Since(startedAt),
		)

		return nil, err
	}
	if resp == nil || resp.Body == nil {
		return resp, nil
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		mapTileCacheLogger.Warn(
			"reading map tile response body failed",
			"url", req.URL.String(),
			"error", readErr,
			"duration", time.Since(startedAt),
		)

		return nil, readErr
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		mapTileCacheLogger.Debug(
			"caching downloaded map tile",
			"cache_path", cachePath,
			"bytes", len(body),
			"status_code", resp.StatusCode,
			"duration", time.Since(startedAt),
		)
		t.writeCachedTile(cachePath, body)
	}

	return resp, nil
}

func (t *mapTileCacheTransport) startAsyncFetch(cachePath, rawURL string) bool {
	t.mu.Lock()
	if t.inFlightByPath == nil {
		t.inFlightByPath = make(map[string]struct{})
	}
	if _, inFlight := t.inFlightByPath[cachePath]; inFlight {
		t.mu.Unlock()

		return false
	}
	t.inFlightByPath[cachePath] = struct{}{}
	t.mu.Unlock()

	go t.fetchAndCacheAsync(rawURL, cachePath)

	return true
}

func (t *mapTileCacheTransport) fetchAndCacheAsync(rawURL, cachePath string) {
	startedAt := time.Now()
	defer func() {
		t.mu.Lock()
		delete(t.inFlightByPath, cachePath)
		t.mu.Unlock()
		if callback := t.asyncCallback(); callback != nil {
			callback()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), defaultMapTileTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		mapTileCacheLogger.Warn("building async map tile request failed", "url", rawURL, "error", err)

		return
	}
	req.Header.Set("User-Agent", "meshgo map tile async fetch")
	req.Header.Set(mapTileFetchModeHeader, mapTileFetchModeSync)

	resp, err := t.baseRoundTripper().RoundTrip(req)
	if err != nil {
		mapTileCacheLogger.Warn(
			"async map tile request failed",
			"url", rawURL,
			"error", err,
			"duration", time.Since(startedAt),
		)

		return
	}
	if resp == nil || resp.Body == nil {
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		mapTileCacheLogger.Warn(
			"reading async map tile response body failed",
			"url", rawURL,
			"error", readErr,
			"duration", time.Since(startedAt),
		)

		return
	}
	if resp.StatusCode != http.StatusOK || len(body) == 0 {
		mapTileCacheLogger.Debug(
			"async map tile fetch returned non-cacheable response",
			"url", rawURL,
			"status_code", resp.StatusCode,
			"bytes", len(body),
			"duration", time.Since(startedAt),
		)

		return
	}

	t.writeCachedTile(cachePath, body)
	mapTileCacheLogger.Debug(
		"async map tile fetch cached successfully",
		"url", rawURL,
		"cache_path", cachePath,
		"bytes", len(body),
		"duration", time.Since(startedAt),
	)
}

func (t *mapTileCacheTransport) baseRoundTripper() http.RoundTripper {
	if t != nil && t.base != nil {
		return t.base
	}

	return http.DefaultTransport
}

func (t *mapTileCacheTransport) setOnAsyncTileCached(callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onAsyncTileCached = callback
}

func (t *mapTileCacheTransport) asyncCallback() func() {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.onAsyncTileCached
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
		if !os.IsNotExist(err) {
			mapTileCacheLogger.Debug("reading cached map tile failed", "cache_path", path, "error", err)
		}

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
		mapTileCacheLogger.Warn("creating map tile cache directory failed", "cache_path", path, "error", err)

		return
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		mapTileCacheLogger.Warn("writing map tile cache temp file failed", "tmp_path", tmpPath, "error", err)
		_ = os.Remove(tmpPath)

		return
	}
	if err := os.Rename(tmpPath, path); err != nil {
		mapTileCacheLogger.Warn("renaming map tile cache temp file failed", "tmp_path", tmpPath, "cache_path", path, "error", err)
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
			mapTileCacheLogger.Debug("walking map tile cache failed", "cache_dir", t.cacheDir, "error", err)

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
	mapTileCacheLogger.Debug("evicting map tile cache entries", "current_bytes", totalSize, "max_bytes", t.maxBytes, "file_count", len(files))

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	for _, file := range files {
		if totalSize <= t.maxBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			mapTileCacheLogger.Debug("removing cached map tile failed", "cache_path", file.path, "error", err)

			continue
		}
		totalSize -= file.size
	}
	mapTileCacheLogger.Debug("map tile cache eviction completed", "remaining_bytes", totalSize, "max_bytes", t.maxBytes)
}

func setMapTileClientAsyncCachedCallback(client *http.Client, callback func()) {
	if client == nil {
		return
	}
	transport, ok := client.Transport.(*mapTileCacheTransport)
	if !ok || transport == nil {
		return
	}
	transport.setOnAsyncTileCached(callback)
}

func mapTileClientCachedProgress(client *http.Client, urls []string) (cached int, total int, ok bool) {
	progress, ok := mapTileClientLoadProgress(client, urls)
	if !ok {
		return 0, len(urls), false
	}

	return progress.cached, progress.total, true
}

type mapTileLoadProgress struct {
	cached   int
	inFlight int
	total    int
}

func mapTileClientLoadProgress(client *http.Client, urls []string) (mapTileLoadProgress, bool) {
	if client == nil || len(urls) == 0 {
		return mapTileLoadProgress{
			total: len(urls),
		}, false
	}
	transport, ok := client.Transport.(*mapTileCacheTransport)
	if !ok || transport == nil {
		return mapTileLoadProgress{
			total: len(urls),
		}, false
	}

	return mapTileLoadProgress{
		cached:   transport.cachedCountForURLs(urls),
		inFlight: transport.inFlightCountForURLs(urls),
		total:    len(urls),
	}, true
}

func (t *mapTileCacheTransport) cachedCountForURLs(urls []string) int {
	if t == nil || len(urls) == 0 {
		return 0
	}
	count := 0
	for _, rawURL := range urls {
		path := t.cachePathForURL(rawURL)
		if t.hasCachedTile(path) {
			count++
		}
	}

	return count
}

func (t *mapTileCacheTransport) hasCachedTile(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(filepath.Clean(path))

	return err == nil
}

func (t *mapTileCacheTransport) inFlightCountForURLs(urls []string) int {
	if t == nil || len(urls) == 0 {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for _, rawURL := range urls {
		path := t.cachePathForURL(rawURL)
		if _, ok := t.inFlightByPath[path]; ok {
			count++
		}
	}

	return count
}
