package ui

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestPositionLogRow(t *testing.T) {
	channel := uint32(2)
	latitude := 12.345678
	longitude := 98.765432
	altitude := int32(123)
	precision := uint32(13)
	observed := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	row := positionLogRow(domain.NodePositionHistoryEntry{
		ObservedAt: observed,
		UpdateType: domain.NodeUpdateTypePositionPacket,
		FromPacket: true,
		Channel:    &channel,
		Latitude:   &latitude,
		Longitude:  &longitude,
		Altitude:   &altitude,
		Precision:  &precision,
	})
	if got := row[0]; got != "12.345678" {
		t.Fatalf("unexpected latitude value: %q", got)
	}
	if got := row[1]; got != "98.765432" {
		t.Fatalf("unexpected longitude value: %q", got)
	}
	if got := row[2]; got != "123 m" {
		t.Fatalf("unexpected altitude value: %q", got)
	}
	if got := row[3]; got != "~ 2.9 km" {
		t.Fatalf("unexpected precision value: %q", got)
	}
	if got := row[4]; got != "2" {
		t.Fatalf("unexpected channel value: %q", got)
	}
	if got := row[5]; got != string(domain.NodeUpdateTypePositionPacket) {
		t.Fatalf("unexpected update type value: %q", got)
	}
	if got := row[6]; got == "unknown" {
		t.Fatalf("expected observed time value")
	}
}

func TestPositionLogHelpersUnknownDefaults(t *testing.T) {
	if got := formatInt32(nil, "%d"); got != "unknown" {
		t.Fatalf("unexpected int fallback: %q", got)
	}
	if got := formatPositionPrecision(nil); got != "unknown" {
		t.Fatalf("unexpected precision fallback: %q", got)
	}
}

func TestPositionLogLatitudeURL_UsesSelectedProvider(t *testing.T) {
	latitude := 37.7749
	longitude := -122.4194
	precision := uint32(13)

	parsed := positionLogLatitudeURL(domain.NodePositionHistoryEntry{
		Latitude:  &latitude,
		Longitude: &longitude,
		Precision: &precision,
	}, config.MapLinkProviderOpenStreetMap)
	if parsed == nil {
		t.Fatalf("expected latitude URL")
	}
	if parsed.Host != "www.openstreetmap.org" {
		t.Fatalf("expected OSM host, got %q", parsed.Host)
	}

	values, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		t.Fatalf("parse OSM query: %v", err)
	}
	if values.Get("mlat") != "37.774900" || values.Get("mlon") != "-122.419400" {
		t.Fatalf("unexpected OSM query: %q", parsed.RawQuery)
	}
	if !strings.Contains(parsed.Fragment, "/37.774900/-122.419400") {
		t.Fatalf("unexpected OSM fragment: %q", parsed.Fragment)
	}
}
