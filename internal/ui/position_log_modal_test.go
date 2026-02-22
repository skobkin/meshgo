package ui

import (
	"strings"
	"testing"
	"time"

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
	if got := row[0]; got == "unknown" {
		t.Fatalf("expected observed time value")
	}
	if got := row[1]; got != string(domain.NodeUpdateTypePositionPacket) {
		t.Fatalf("unexpected update type value: %q", got)
	}
	if got := row[2]; got != "yes" {
		t.Fatalf("unexpected from packet value: %q", got)
	}
	if got := row[3]; got != "2" {
		t.Fatalf("unexpected channel value: %q", got)
	}
	if got := row[4]; got != "12.345678" {
		t.Fatalf("unexpected latitude value: %q", got)
	}
	if got := row[5]; got != "98.765432" {
		t.Fatalf("unexpected longitude value: %q", got)
	}
	if got := row[6]; got != "123 m" {
		t.Fatalf("unexpected altitude value: %q", got)
	}
	if got := row[7]; !strings.Contains(got, "(13)") {
		t.Fatalf("unexpected precision value: %q", got)
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
