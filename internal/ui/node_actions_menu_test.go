package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewNodeContextMenu_ContainsTracerouteAction(t *testing.T) {
	node := domain.Node{NodeID: "!0000002a", LongName: "Alpha", ShortName: "ALPH"}

	calledActions := make([]NodeAction, 0, 2)
	menu := newNodeContextMenu(node, func(_ domain.Node, action NodeAction) {
		calledActions = append(calledActions, action)
	})
	if menu == nil {
		t.Fatalf("expected menu")
	}
	if len(menu.Items) != 2 {
		t.Fatalf("expected two menu items, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Direct message" {
		t.Fatalf("unexpected first menu item label: %q", menu.Items[0].Label)
	}
	if menu.Items[1].Label != "Traceroute" {
		t.Fatalf("unexpected second menu item label: %q", menu.Items[1].Label)
	}
	for _, item := range menu.Items {
		item.Action()
	}
	if len(calledActions) != 2 {
		t.Fatalf("expected two callback invocations, got %d", len(calledActions))
	}
	if calledActions[0] != NodeActionDirectMessage {
		t.Fatalf("unexpected first action: %q", calledActions[0])
	}
	if calledActions[1] != NodeActionTraceroute {
		t.Fatalf("unexpected second action: %q", calledActions[1])
	}
}
