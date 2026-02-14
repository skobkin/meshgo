package domain

import "testing"

func TestNodeStoreUpsert_PreservesCoordinatesOnSparseUpdates(t *testing.T) {
	store := NewNodeStore()
	lat := 37.7749
	lon := -122.4194
	alt := int32(123)

	store.Upsert(Node{
		NodeID:     "!11111111",
		Latitude:   &lat,
		Longitude:  &lon,
		Altitude:   &alt,
		LongName:   "Alpha",
		ShortName:  "ALPH",
		BoardModel: "T-Echo",
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
	if node.LongName != "Alpha Updated" {
		t.Fatalf("expected long name update to apply, got %q", node.LongName)
	}
	if node.ShortName != "ALPH" {
		t.Fatalf("expected short name preserved, got %q", node.ShortName)
	}
}
