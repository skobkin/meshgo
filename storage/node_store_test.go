package storage

import (
	"context"
	"path/filepath"
	"testing"

	"meshgo/domain"
)

func TestNodeStoreUpsertAndList(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nodes.db")
	ns, err := OpenNodeStore(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	n := &domain.Node{ID: "1", ShortName: "N1", RSSI: -90, SNR: 9, Signal: domain.SignalGood}
	if err := ns.UpsertNode(ctx, n); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	nodes, err := ns.ListNodes(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "1" {
		t.Fatalf("unexpected nodes: %+v", nodes)
	}
}

func TestNodeStoreSetters(t *testing.T) {
	ctx := context.Background()
	ns, err := OpenNodeStore(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := ns.UpsertNode(ctx, &domain.Node{ID: "n1"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := ns.SetFavorite(ctx, "n1", true); err != nil {
		t.Fatalf("SetFavorite: %v", err)
	}
	if err := ns.SetIgnored(ctx, "n1", true); err != nil {
		t.Fatalf("SetIgnored: %v", err)
	}
	nodes, err := ns.ListNodes(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 1 || !nodes[0].Favorite || !nodes[0].Ignored {
		t.Fatalf("unexpected node: %+v", nodes[0])
	}
}

func TestNodeStoreRemove(t *testing.T) {
	ctx := context.Background()
	ns, err := OpenNodeStore(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := ns.UpsertNode(ctx, &domain.Node{ID: "n1"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := ns.RemoveNode(ctx, "n1"); err != nil {
		t.Fatalf("RemoveNode: %v", err)
	}
	nodes, err := ns.ListNodes(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected no nodes, got %d", len(nodes))
	}
}
