package ui

import (
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestTelemetryLogRow(t *testing.T) {
	battery := uint32(77)
	voltage := 4.01
	uptime := uint32(3600)
	air := 3.5
	soilTemperature := 19.4
	soilMoisture := uint32(56)
	gas := 0.42
	lux := 256.0
	uvLux := 180.0
	radiation := 0.14
	aqi := 42.0
	temperature := 20.0
	humidity := 60.0
	observed := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	row := telemetryLogRow(domain.NodeTelemetryHistoryEntry{
		ObservedAt:         observed,
		UpdateType:         domain.NodeUpdateTypeTelemetryPacket,
		FromPacket:         true,
		BatteryLevel:       &battery,
		Voltage:            &voltage,
		UptimeSeconds:      &uptime,
		ChannelUtilization: &air,
		Temperature:        &temperature,
		Humidity:           &humidity,
		SoilTemperature:    &soilTemperature,
		SoilMoisture:       &soilMoisture,
		GasResistance:      &gas,
		Lux:                &lux,
		UVLux:              &uvLux,
		Radiation:          &radiation,
		AirQualityIndex:    &aqi,
	})
	if got := row[0]; got != "77%" {
		t.Fatalf("unexpected battery value: %q", got)
	}
	if got := row[2]; got == "unknown" {
		t.Fatalf("expected uptime to be formatted")
	}
	if got := row[8]; got == "unknown" {
		t.Fatalf("expected soil temperature to be formatted")
	}
	if got := row[9]; got == "unknown" {
		t.Fatalf("expected soil moisture to be formatted")
	}
	if got := row[10]; got == "unknown" {
		t.Fatalf("expected gas resistance to be formatted")
	}
	if got := row[11]; got == "unknown" {
		t.Fatalf("expected AQI to be formatted")
	}
	if got := row[12]; got == "unknown" {
		t.Fatalf("expected dew point to be formatted")
	}
	if got := row[13]; got == "unknown" {
		t.Fatalf("expected lux to be formatted")
	}
	if got := row[14]; got == "unknown" {
		t.Fatalf("expected uv lux to be formatted")
	}
	if got := row[15]; got == "unknown" {
		t.Fatalf("expected radiation to be formatted")
	}
	if got := row[19]; got != string(domain.NodeUpdateTypeTelemetryPacket) {
		t.Fatalf("unexpected update type value: %q", got)
	}
	if got := row[20]; got == "unknown" {
		t.Fatalf("expected observed time value")
	}
}

func TestTelemetryLogHelpersUnknownDefaults(t *testing.T) {
	if got := telemetryLogUpdateType(""); got != "unknown" {
		t.Fatalf("unexpected update type fallback: %q", got)
	}
	if got := formatFloat64(nil, "%.1f"); got != "unknown" {
		t.Fatalf("unexpected float fallback: %q", got)
	}
	if got := formatUint32(nil, "%d"); got != "unknown" {
		t.Fatalf("unexpected uint fallback: %q", got)
	}
	if got := telemetryLogTime(time.Time{}); got != "unknown" {
		t.Fatalf("unexpected time fallback: %q", got)
	}
}
