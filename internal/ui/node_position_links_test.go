package ui

import (
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

	parsed, err := nodePositionMapURL(config.MapLinkProviderOpenStreetMap, latitude, longitude, &precision)
	if err != nil {
		t.Fatalf("build map URL: %v", err)
	}
	if parsed.Host != "www.openstreetmap.org" {
		t.Fatalf("unexpected map host: %q", parsed.Host)
	}
	if !strings.Contains(parsed.String(), "#map=") {
		t.Fatalf("expected map fragment in URL, got %q", parsed.String())
	}
}
