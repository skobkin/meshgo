package ui

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/skobkin/meshgo/internal/config"
)

func TestNodePositionLinkZoomLevel(t *testing.T) {
	latitude := 37.7749
	if got := nodePositionLinkZoomLevel(latitude, nil); got != mapDefaultZoom {
		t.Fatalf("expected default zoom %d for unknown precision, got %d", mapDefaultZoom, got)
	}

	precise := uint32(32)
	if got := nodePositionLinkZoomLevel(latitude, &precise); got != nodePositionLinkPreciseZoom {
		t.Fatalf("expected precise zoom %d, got %d", nodePositionLinkPreciseZoom, got)
	}

	coarse := uint32(10)
	mid := uint32(13)
	fine := uint32(18)
	zoomCoarse := nodePositionLinkZoomLevel(latitude, &coarse)
	zoomMid := nodePositionLinkZoomLevel(latitude, &mid)
	zoomFine := nodePositionLinkZoomLevel(latitude, &fine)
	if zoomCoarse >= zoomMid || zoomMid >= zoomFine {
		t.Fatalf("expected zoom ordering coarse < mid < fine, got %d < %d < %d", zoomCoarse, zoomMid, zoomFine)
	}
}

func TestNodePositionMapURL_UsesProviderImplementation(t *testing.T) {
	latitude := 51.5007
	longitude := -0.1246
	precision := uint32(14)

	tests := []struct {
		name     string
		provider config.MapLinkProvider
		host     string
		assert   func(t *testing.T, parsed *url.URL)
	}{
		{
			name:     "openstreetmap",
			provider: config.MapLinkProviderOpenStreetMap,
			host:     "www.openstreetmap.org",
			assert: func(t *testing.T, parsed *url.URL) {
				t.Helper()
				if !strings.Contains(parsed.String(), "#map=") {
					t.Fatalf("expected map fragment in URL, got %q", parsed.String())
				}
			},
		},
		{
			name:     "kagi",
			provider: config.MapLinkProviderKagi,
			host:     "kagi.com",
		},
		{
			name:     "google",
			provider: config.MapLinkProviderGoogle,
			host:     "www.google.com",
		},
		{
			name:     "yandex",
			provider: config.MapLinkProviderYandex,
			host:     "yandex.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := nodePositionMapURL(tc.provider, latitude, longitude, &precision)
			if err != nil {
				t.Fatalf("build map URL: %v", err)
			}
			if parsed.Host != tc.host {
				t.Fatalf("unexpected map host: %q", parsed.Host)
			}
			if tc.assert != nil {
				tc.assert(t, parsed)
			}
		})
	}
}

func TestMapLinkProviderLabels_Order(t *testing.T) {
	got := mapLinkProviderLabels()
	want := []string{"OpenStreetMap", "Kagi", "Google", "Yandex"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected provider labels order: got %v, want %v", got, want)
	}
}

func TestNodePositionKagiURL_UsesComputedZoomAndCoordinates(t *testing.T) {
	latitude := 51.5007
	longitude := -0.1246
	precision := uint32(14)
	expectedZoom := nodePositionLinkZoomLevel(latitude, &precision)

	parsed, err := nodePositionMapURL(config.MapLinkProviderKagi, latitude, longitude, &precision)
	if err != nil {
		t.Fatalf("build map URL: %v", err)
	}
	if parsed.Path != "/maps" {
		t.Fatalf("unexpected kagi URL path: %q", parsed.Path)
	}

	wantFragmentPrefix := fmt.Sprintf("%d/", expectedZoom)
	if !strings.HasPrefix(parsed.Fragment, wantFragmentPrefix) {
		t.Fatalf("unexpected kagi map fragment %q, expected prefix %q", parsed.Fragment, wantFragmentPrefix)
	}

	query := parsed.Query().Get("q")
	if query != fmt.Sprintf("%.6f,%.6f", latitude, longitude) {
		t.Fatalf("unexpected kagi query coords: %q", query)
	}
}
