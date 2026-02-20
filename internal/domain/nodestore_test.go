package domain

import (
	"testing"
	"time"
)

func TestNodeStoreUpsert_PreservesCoordinatesOnSparseUpdates(t *testing.T) {
	store := NewNodeStore()
	lat := 37.7749
	lon := -122.4194
	alt := int32(123)
	precisionBits := uint32(15)
	uptimeSeconds := uint32(1200)
	channelUtilization := 18.5
	airUtilTx := 2.2
	positionUpdatedAt := time.Now().Add(-2 * time.Minute).UTC()

	store.Upsert(Node{
		NodeID:                "!11111111",
		Latitude:              &lat,
		Longitude:             &lon,
		Altitude:              &alt,
		PositionPrecisionBits: &precisionBits,
		LongName:              "Alpha",
		ShortName:             "ALPH",
		BoardModel:            "T-Echo",
		FirmwareVersion:       "2.5.0",
		UptimeSeconds:         &uptimeSeconds,
		ChannelUtilization:    &channelUtilization,
		AirUtilTx:             &airUtilTx,
		PositionUpdatedAt:     positionUpdatedAt,
	})
	store.Upsert(Node{
		NodeID:   "!11111111",
		LongName: "Alpha Updated",
	})

	node, ok := store.Get("!11111111")
	if !ok {
		t.Fatalf("expected node in store")
	}
	if node.Latitude == nil || *node.Latitude != lat {
		t.Fatalf("expected latitude preserved, got %v", node.Latitude)
	}
	if node.Longitude == nil || *node.Longitude != lon {
		t.Fatalf("expected longitude preserved, got %v", node.Longitude)
	}
	if node.Altitude == nil || *node.Altitude != alt {
		t.Fatalf("expected altitude preserved, got %v", node.Altitude)
	}
	if node.PositionPrecisionBits == nil || *node.PositionPrecisionBits != precisionBits {
		t.Fatalf("expected precision bits preserved, got %v", node.PositionPrecisionBits)
	}
	if node.LongName != "Alpha Updated" {
		t.Fatalf("expected long name update to apply, got %q", node.LongName)
	}
	if node.ShortName != "ALPH" {
		t.Fatalf("expected short name preserved, got %q", node.ShortName)
	}
	if node.FirmwareVersion != "2.5.0" {
		t.Fatalf("expected firmware preserved, got %q", node.FirmwareVersion)
	}
	if node.UptimeSeconds == nil || *node.UptimeSeconds != uptimeSeconds {
		t.Fatalf("expected uptime preserved, got %v", node.UptimeSeconds)
	}
	if node.ChannelUtilization == nil || *node.ChannelUtilization != channelUtilization {
		t.Fatalf("expected channel utilization preserved, got %v", node.ChannelUtilization)
	}
	if node.AirUtilTx == nil || *node.AirUtilTx != airUtilTx {
		t.Fatalf("expected air util tx preserved, got %v", node.AirUtilTx)
	}
	if !node.PositionUpdatedAt.Equal(positionUpdatedAt) {
		t.Fatalf("expected position updated at preserved, got %v", node.PositionUpdatedAt)
	}
}
