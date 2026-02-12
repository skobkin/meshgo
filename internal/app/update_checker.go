package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"
)

const (
	defaultUpdateCheckInterval  = 12 * time.Hour
	defaultUpdateRequestTimeout = 15 * time.Second
	defaultReleaseQueryURL      = "https://git.skobk.in/api/v1/repos/skobkin/meshgo/releases?draft=false&pre-release=false&limit=5"
)

// ReleaseInfo contains release metadata used by update UI.
type ReleaseInfo struct {
	Version     string
	Body        string
	HTMLURL     string
	PublishedAt time.Time
}

// UpdateSnapshot stores a single successful update check result.
type UpdateSnapshot struct {
	CurrentVersion  string
	Latest          ReleaseInfo
	Releases        []ReleaseInfo
	UpdateAvailable bool
	CheckedAt       time.Time
}

// UpdateCheckerConfig customizes update checker behavior.
type UpdateCheckerConfig struct {
	CurrentVersion string
	Endpoint       string
	HTTPClient     *http.Client
	Interval       time.Duration
	Logger         *slog.Logger
}

// UpdateChecker periodically fetches releases and publishes update snapshots.
type UpdateChecker struct {
	currentVersion string
	endpoint       string
	client         *http.Client
	interval       time.Duration
	logger         *slog.Logger

	snapshots chan UpdateSnapshot

	mu          sync.RWMutex
	latest      UpdateSnapshot
	latestKnown bool

	startOnce sync.Once
}

type forgejoRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

func NewUpdateChecker(cfg UpdateCheckerConfig) *UpdateChecker {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = defaultReleaseQueryURL
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultUpdateRequestTimeout}
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = defaultUpdateCheckInterval
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &UpdateChecker{
		currentVersion: strings.TrimSpace(cfg.CurrentVersion),
		endpoint:       endpoint,
		client:         client,
		interval:       interval,
		logger:         logger,
		snapshots:      make(chan UpdateSnapshot, 1),
	}
}

func (c *UpdateChecker) Start(ctx context.Context) {
	if c == nil {
		return
	}

	c.startOnce.Do(func() {
		go c.run(ctx)
	})
}

func (c *UpdateChecker) Snapshots() <-chan UpdateSnapshot {
	if c == nil {
		return nil
	}

	return c.snapshots
}

func (c *UpdateChecker) CurrentSnapshot() (UpdateSnapshot, bool) {
	if c == nil {
		return UpdateSnapshot{}, false
	}

	c.mu.RLock()
	snapshot := c.latest
	known := c.latestKnown
	c.mu.RUnlock()

	return snapshot, known
}

func (c *UpdateChecker) run(ctx context.Context) {
	c.logger.Info("update checker started", "endpoint", c.endpoint, "interval", c.interval.String(), "current_version", c.currentVersion)

	c.logger.Debug("running initial update check")
	if err := c.checkAndPublish(ctx); err != nil {
		c.logger.Warn("check for updates", "error", err)
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("update checker stopped")

			return
		case <-ticker.C:
			c.logger.Debug("running scheduled update check")
			if err := c.checkAndPublish(ctx); err != nil {
				c.logger.Warn("check for updates", "error", err)
			}
		}
	}
}

func (c *UpdateChecker) checkAndPublish(ctx context.Context) error {
	c.logger.Debug("checking for updates")

	snapshot, err := c.fetchSnapshot(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.latest = snapshot
	c.latestKnown = true
	c.mu.Unlock()

	c.publish(snapshot)
	c.logger.Info(
		"update check completed",
		"checked_at", snapshot.CheckedAt.Format(time.RFC3339),
		"current_version", snapshot.CurrentVersion,
		"latest_version", snapshot.Latest.Version,
		"update_available", snapshot.UpdateAvailable,
		"release_count", len(snapshot.Releases),
	)

	return nil
}

func (c *UpdateChecker) publish(snapshot UpdateSnapshot) {
	select {
	case c.snapshots <- snapshot:
		c.logger.Debug("published update snapshot")

		return
	default:
	}
	c.logger.Debug("update snapshot channel full, replacing stale value")

	select {
	case <-c.snapshots:
	default:
	}

	select {
	case c.snapshots <- snapshot:
		c.logger.Debug("published update snapshot after replacing stale value")
	default:
		c.logger.Debug("skipped update snapshot publish after replace attempt")
	}
}

func (c *UpdateChecker) fetchSnapshot(ctx context.Context) (UpdateSnapshot, error) {
	releases, err := c.fetchReleases(ctx)
	if err != nil {
		return UpdateSnapshot{}, err
	}
	if len(releases) == 0 {
		return UpdateSnapshot{}, fmt.Errorf("release API response is empty")
	}

	latest := releases[0]
	updateAvailable := isReleaseNewer(c.currentVersion, latest.Version)
	c.logger.Debug(
		"resolved latest release",
		"current_version", c.currentVersion,
		"latest_version", latest.Version,
		"release_count", len(releases),
		"update_available", updateAvailable,
	)

	return UpdateSnapshot{
		CurrentVersion:  c.currentVersion,
		Latest:          latest,
		Releases:        releases,
		UpdateAvailable: updateAvailable,
		CheckedAt:       time.Now().UTC(),
	}, nil
}

func (c *UpdateChecker) fetchReleases(ctx context.Context) ([]ReleaseInfo, error) {
	c.logger.Debug("requesting releases", "endpoint", c.endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create releases request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request releases: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	c.logger.Debug("received releases response", "status_code", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		trimmedBody := strings.TrimSpace(string(body))
		if trimmedBody == "" {
			return nil, fmt.Errorf("request releases: unexpected status %d", resp.StatusCode)
		}

		return nil, fmt.Errorf("request releases: unexpected status %d: %s", resp.StatusCode, trimmedBody)
	}

	var payload []forgejoRelease
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode releases response: %w", err)
	}

	releases := make([]ReleaseInfo, 0, len(payload))
	skippedWithoutVersion := 0
	for _, item := range payload {
		version := strings.TrimSpace(item.TagName)
		if version == "" {
			skippedWithoutVersion++

			continue
		}
		releases = append(releases, ReleaseInfo{
			Version:     version,
			Body:        strings.TrimSpace(item.Body),
			HTMLURL:     strings.TrimSpace(item.HTMLURL),
			PublishedAt: item.PublishedAt,
		})
	}
	c.logger.Debug(
		"parsed releases response",
		"items_total", len(payload),
		"items_without_version", skippedWithoutVersion,
		"items_usable", len(releases),
	)

	return releases, nil
}

func isReleaseNewer(currentVersion string, latestVersion string) bool {
	current := normalizeSemver(currentVersion)
	latest := normalizeSemver(latestVersion)

	latestValid := semver.IsValid(latest)
	if !latestValid {
		return false
	}

	currentValid := semver.IsValid(current)
	if !currentValid {
		return true
	}

	return semver.Compare(current, latest) < 0
}

func normalizeSemver(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "v") {
		return "v" + trimmed
	}

	return trimmed
}
