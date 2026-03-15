package mapwidgets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultMapTileCacheSizeBytes is the default maximum cache size for map tiles (200MB).
	DefaultMapTileCacheSizeBytes = int64(200 * 1024 * 1024)
	// DefaultMapTileTimeout is the default timeout for map tile HTTP requests.
	DefaultMapTileTimeout = 15 * time.Second
	// MapTileFetchModeHeader is the HTTP header used to indicate tile fetch mode.
	MapTileFetchModeHeader = "X-Meshgo-Tile-Fetch-Mode"
	// MapTileFetchModeSync is the value for synchronous tile fetching.
	MapTileFetchModeSync = "sync"
)

var mapTileCacheLogger = slog.With("component", "ui.map_tile_cache")

// MapTileCacheTransport is an HTTP transport that caches map tiles to disk.
type MapTileCacheTransport struct {
	Base      http.RoundTripper
	CacheDir  string
	MaxBytes  int64
	AsyncMiss bool

	mu                sync.Mutex
	inFlightByPath    map[string]struct{}
	onAsyncTileCached func()
}

// NewMapTileHTTPClient creates an HTTP client with tile caching support.
func NewMapTileHTTPClient(cacheDir string, maxBytes int64) *http.Client {
	base := http.DefaultTransport
	if maxBytes <= 0 {
		maxBytes = DefaultMapTileCacheSizeBytes
	}
	mapTileCacheLogger.Info(
		"initializing map tile HTTP client",
		"cache_enabled", cacheDir != "",
		"cache_dir", cacheDir,
		"max_bytes", maxBytes,
		"timeout", DefaultMapTileTimeout,
	)
	if cacheDir == "" {
		return &http.Client{
			Timeout: DefaultMapTileTimeout,
		}
	}

	return &http.Client{
		Transport: &MapTileCacheTransport{
			Base:           base,
			CacheDir:       cacheDir,
			MaxBytes:       maxBytes,
			AsyncMiss:      true,
			inFlightByPath: make(map[string]struct{}),
		},
		Timeout: DefaultMapTileTimeout,
	}
}

func (t *MapTileCacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.Method != http.MethodGet || req.URL == nil || req.URL.String() == "" {
		return t.baseRoundTripper().RoundTrip(req)
	}
	startedAt := time.Now()
	syncFetch := req.Header.Get(MapTileFetchModeHeader) == MapTileFetchModeSync

	cachePath := t.CachePathForURL(req.URL.String())
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

	if t.AsyncMiss && !syncFetch {
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
		t.WriteCachedTile(cachePath, body)
	}

	return resp, nil
}

func (t *MapTileCacheTransport) startAsyncFetch(cachePath, rawURL string) bool {
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

func (t *MapTileCacheTransport) fetchAndCacheAsync(rawURL, cachePath string) {
	startedAt := time.Now()
	defer func() {
		t.mu.Lock()
		delete(t.inFlightByPath, cachePath)
		t.mu.Unlock()
		if callback := t.asyncCallback(); callback != nil {
			callback()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), DefaultMapTileTimeout)
	defer cancel()

	parsedURL, err := validateMapTileURL(rawURL)
	if err != nil {
		mapTileCacheLogger.Warn("rejecting async map tile request URL", "url", rawURL, "error", err)

		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil) // #nosec G704 -- URL is restricted to http/https with a non-empty host by validateMapTileURL.
	if err != nil {
		mapTileCacheLogger.Warn("building async map tile request failed", "url", rawURL, "error", err)

		return
	}
	req.Header.Set("User-Agent", "meshgo map tile async fetch")
	req.Header.Set(MapTileFetchModeHeader, MapTileFetchModeSync)

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

	t.WriteCachedTile(cachePath, body)
	mapTileCacheLogger.Debug(
		"async map tile fetch cached successfully",
		"url", rawURL,
		"cache_path", cachePath,
		"bytes", len(body),
		"duration", time.Since(startedAt),
	)
}

func (t *MapTileCacheTransport) baseRoundTripper() http.RoundTripper {
	if t != nil && t.Base != nil {
		return t.Base
	}

	return http.DefaultTransport
}

func (t *MapTileCacheTransport) SetOnAsyncTileCached(callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onAsyncTileCached = callback
}

func (t *MapTileCacheTransport) asyncCallback() func() {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.onAsyncTileCached
}

func (t *MapTileCacheTransport) CachePathForURL(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	hash := hex.EncodeToString(sum[:])
	prefix := filepath.Join(hash[:2], hash[2:4])

	return filepath.Join(t.CacheDir, prefix, hash+".tile")
}

func (t *MapTileCacheTransport) readCachedTile(path string) ([]byte, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cleanPath, ok := t.safeCachePath(path)
	if !ok {
		mapTileCacheLogger.Warn("rejecting map tile cache read outside cache directory", "cache_path", path, "cache_dir", t.CacheDir)

		return nil, false
	}

	data, err := os.ReadFile(cleanPath) // #nosec G304 G703 -- cleanPath is constrained to stay within CacheDir by safeCachePath.
	if err != nil {
		if !os.IsNotExist(err) {
			mapTileCacheLogger.Debug("reading cached map tile failed", "cache_path", cleanPath, "error", err)
		}

		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(cleanPath, now, now)

	return data, true
}

func (t *MapTileCacheTransport) WriteCachedTile(path string, data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cleanPath, ok := t.safeCachePath(path)
	if !ok {
		mapTileCacheLogger.Warn("rejecting map tile cache write outside cache directory", "cache_path", path, "cache_dir", t.CacheDir)

		return
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o750); err != nil { // #nosec G703 -- cleanPath is constrained to stay within CacheDir by safeCachePath.
		mapTileCacheLogger.Warn("creating map tile cache directory failed", "cache_path", cleanPath, "error", err)

		return
	}

	tmpPath := cleanPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil { // #nosec G703 -- tmpPath is derived from a safe cache path inside CacheDir.
		mapTileCacheLogger.Warn("writing map tile cache temp file failed", "tmp_path", tmpPath, "error", err)
		_ = os.Remove(tmpPath) // #nosec G703 -- tmpPath is derived from a safe cache path inside CacheDir.

		return
	}
	if err := os.Rename(tmpPath, cleanPath); err != nil { // #nosec G703 -- both paths are constrained to stay within CacheDir by safeCachePath.
		mapTileCacheLogger.Warn("renaming map tile cache temp file failed", "tmp_path", tmpPath, "cache_path", cleanPath, "error", err)
		_ = os.Remove(tmpPath) // #nosec G703 -- tmpPath is derived from a safe cache path inside CacheDir.

		return
	}
	now := time.Now()
	_ = os.Chtimes(cleanPath, now, now)

	t.evictIfNeededLocked()
}

func validateMapTileURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("unsupported URL scheme %q", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("URL host is required")
	}

	return parsedURL, nil
}

func (t *MapTileCacheTransport) safeCachePath(path string) (string, bool) {
	cacheDir := strings.TrimSpace(t.CacheDir)
	if cacheDir == "" {
		return "", false
	}

	base := filepath.Clean(cacheDir)
	target := filepath.Clean(path)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if rel == "." {
		return target, true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	return target, true
}

func (t *MapTileCacheTransport) evictIfNeededLocked() {
	type cacheFile struct {
		path    string
		size    int64
		modTime time.Time
	}

	var (
		files     []cacheFile
		totalSize int64
	)

	_ = filepath.WalkDir(t.CacheDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			mapTileCacheLogger.Debug("walking map tile cache failed", "cache_dir", t.CacheDir, "error", err)

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

	if totalSize <= t.MaxBytes {
		return
	}
	mapTileCacheLogger.Debug("evicting map tile cache entries", "current_bytes", totalSize, "max_bytes", t.MaxBytes, "file_count", len(files))

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	for _, file := range files {
		if totalSize <= t.MaxBytes {
			break
		}
		if err := os.Remove(file.path); err != nil {
			mapTileCacheLogger.Debug("removing cached map tile failed", "cache_path", file.path, "error", err)

			continue
		}
		totalSize -= file.size
	}
	mapTileCacheLogger.Debug("map tile cache eviction completed", "remaining_bytes", totalSize, "max_bytes", t.MaxBytes)
}

// SetMapTileClientAsyncCachedCallback sets a callback for when async tile caching completes.
func SetMapTileClientAsyncCachedCallback(client *http.Client, callback func()) {
	if client == nil {
		return
	}
	transport, ok := client.Transport.(*MapTileCacheTransport)
	if !ok || transport == nil {
		return
	}
	transport.SetOnAsyncTileCached(callback)
}

// MapTileClientCachedProgress returns the number of cached tiles vs total tiles for the given URLs.
func MapTileClientCachedProgress(client *http.Client, urls []string) (cached int, total int, ok bool) {
	progress, ok := MapTileClientLoadProgress(client, urls)
	if !ok {
		return 0, len(urls), false
	}

	return progress.Cached, progress.Total, true
}

// MapTileLoadProgress tracks the loading progress of map tiles.
type MapTileLoadProgress struct {
	Cached   int
	InFlight int
	Total    int
}

// MapTileClientLoadProgress returns the loading progress for the given tile URLs.
func MapTileClientLoadProgress(client *http.Client, urls []string) (MapTileLoadProgress, bool) {
	if client == nil || len(urls) == 0 {
		return MapTileLoadProgress{
			Total: len(urls),
		}, false
	}
	transport, ok := client.Transport.(*MapTileCacheTransport)
	if !ok || transport == nil {
		return MapTileLoadProgress{
			Total: len(urls),
		}, false
	}

	return MapTileLoadProgress{
		Cached:   transport.cachedCountForURLs(urls),
		InFlight: transport.inFlightCountForURLs(urls),
		Total:    len(urls),
	}, true
}

func (t *MapTileCacheTransport) cachedCountForURLs(urls []string) int {
	if t == nil || len(urls) == 0 {
		return 0
	}
	count := 0
	for _, rawURL := range urls {
		path := t.CachePathForURL(rawURL)
		if t.hasCachedTile(path) {
			count++
		}
	}

	return count
}

func (t *MapTileCacheTransport) hasCachedTile(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(filepath.Clean(path))

	return err == nil
}

func (t *MapTileCacheTransport) inFlightCountForURLs(urls []string) int {
	if t == nil || len(urls) == 0 {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for _, rawURL := range urls {
		path := t.CachePathForURL(rawURL)
		if _, ok := t.inFlightByPath[path]; ok {
			count++
		}
	}

	return count
}
