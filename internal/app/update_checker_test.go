package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

func TestNormalizeSemver(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "plain", in: "0.7.0", want: "v0.7.0"},
		{name: "already prefixed", in: "v0.7.0", want: "v0.7.0"},
		{name: "trim spaces", in: " 0.7.0 ", want: "v0.7.0"},
	}

	for _, tt := range tests {
		if got := normalizeSemver(tt.in); got != tt.want {
			t.Fatalf("%s: normalizeSemver(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestIsReleaseNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "newer release", current: "0.9.2", latest: "0.10.1", want: true},
		{name: "equal release", current: "0.10.1", latest: "0.10.1", want: false},
		{name: "current newer", current: "0.10.2", latest: "0.10.1", want: false},
		{name: "dev current treated older", current: "dev", latest: "0.10.1", want: true},
		{name: "invalid latest ignored", current: "0.10.1", latest: "not-semver", want: false},
	}

	for _, tt := range tests {
		if got := isReleaseNewer(tt.current, tt.latest); got != tt.want {
			t.Fatalf("%s: isReleaseNewer(%q, %q) = %v, want %v", tt.name, tt.current, tt.latest, got, tt.want)
		}
	}
}

func TestUpdateCheckerFetchSnapshot(t *testing.T) {
	var acceptHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptHeader = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"tag_name":"0.7.0","body":"body-1","html_url":"https://example.com/r/0.7.0","published_at":"2026-02-12T01:00:00Z"},
			{"tag_name":"0.6.1","body":"body-2","html_url":"https://example.com/r/0.6.1","published_at":"2026-02-10T01:00:00Z"}
		]`)
	}))
	defer server.Close()

	checker := NewUpdateChecker(UpdateCheckerDependencies{
		CurrentVersion: "0.6.0",
		Endpoint:       server.URL,
		HTTPClient:     server.Client(),
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	snapshot, err := checker.fetchSnapshot(context.Background())
	if err != nil {
		t.Fatalf("fetchSnapshot() error = %v", err)
	}

	if acceptHeader != "application/json" {
		t.Fatalf("expected Accept header application/json, got %q", acceptHeader)
	}
	if snapshot.CurrentVersion != "0.6.0" {
		t.Fatalf("unexpected current version: %q", snapshot.CurrentVersion)
	}
	if snapshot.Latest.Version != "0.7.0" {
		t.Fatalf("unexpected latest version: %q", snapshot.Latest.Version)
	}
	if snapshot.Latest.HTMLURL != "https://example.com/r/0.7.0" {
		t.Fatalf("unexpected latest html url: %q", snapshot.Latest.HTMLURL)
	}
	if len(snapshot.Releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(snapshot.Releases))
	}
	if !snapshot.UpdateAvailable {
		t.Fatalf("expected update available")
	}
}

func TestUpdateCheckerStartRecoversAfterFailedCheck(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "boom")

			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"tag_name":"0.7.0","body":"body-1","html_url":"https://example.com/r/0.7.0","published_at":"2026-02-12T01:00:00Z"}
		]`)
	}))
	defer server.Close()

	checker := NewUpdateChecker(UpdateCheckerDependencies{
		CurrentVersion: "0.6.0",
		Endpoint:       server.URL,
		HTTPClient:     server.Client(),
		Interval:       25 * time.Millisecond,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checker.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if snapshot, ok := checker.CurrentSnapshot(); ok {
			if snapshot.Latest.Version != "0.7.0" {
				t.Fatalf("unexpected latest version after recovery: %q", snapshot.Latest.Version)
			}

			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected checker to recover and publish snapshot")
}

func TestUpdateCheckerPublishesSnapshotsToBus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	t.Cleanup(func() {
		messageBus.Close()
	})

	sub := messageBus.Subscribe(connectors.TopicUpdateSnapshot)
	t.Cleanup(func() {
		messageBus.Unsubscribe(sub, connectors.TopicUpdateSnapshot)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[
			{"tag_name":"0.7.0","body":"body-1","html_url":"https://example.com/r/0.7.0","published_at":"2026-02-12T01:00:00Z"}
		]`)
	}))
	defer server.Close()

	checker := NewUpdateChecker(UpdateCheckerDependencies{
		CurrentVersion: "0.6.0",
		Endpoint:       server.URL,
		HTTPClient:     server.Client(),
		MessageBus:     messageBus,
		Logger:         logger,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checker.Start(ctx)

	select {
	case raw := <-sub:
		snapshot, ok := raw.(UpdateSnapshot)
		if !ok {
			t.Fatalf("expected UpdateSnapshot payload, got %T", raw)
		}
		if snapshot.Latest.Version != "0.7.0" {
			t.Fatalf("unexpected latest version: %q", snapshot.Latest.Version)
		}
		if snapshot.CurrentVersion != "0.6.0" {
			t.Fatalf("unexpected current version: %q", snapshot.CurrentVersion)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected update snapshot to be published to bus")
	}
}
