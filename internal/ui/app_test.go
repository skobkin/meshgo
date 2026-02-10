package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestResolveNodeDisplayName_Priority(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!11111111", LongName: "Long", ShortName: "Short"})
	store.Upsert(domain.Node{NodeID: "!22222222", ShortName: "ShortOnly"})
	store.Upsert(domain.Node{NodeID: "!33333333"})

	resolve := resolveNodeDisplayName(store)
	if resolve == nil {
		t.Fatalf("expected non-nil resolver")
	}

	if got := resolve("!11111111"); got != "Long" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := resolve("!22222222"); got != "ShortOnly" {
		t.Fatalf("expected short name, got %q", got)
	}
	if got := resolve("!33333333"); got != "" {
		t.Fatalf("expected empty fallback for id-only node, got %q", got)
	}
}

func TestFormatWindowTitle(t *testing.T) {
	got := formatWindowTitle(connectors.ConnStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
	})
	want := "MeshGo - connected via ip"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
