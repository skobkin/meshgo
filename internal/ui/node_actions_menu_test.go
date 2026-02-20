package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewNodeContextMenu_ContainsNodeInfoAction(t *testing.T) {
	node := domain.Node{NodeID: "!0000002a", LongName: "Alpha", ShortName: "ALPH"}

	calledActions := make([]NodeAction, 0, 3)
	menu := newNodeContextMenu(node, func(_ domain.Node, action NodeAction) {
		calledActions = append(calledActions, action)
	})
	if menu == nil {
		t.Fatalf("expected menu")
	}
	if len(menu.Items) != 3 {
		t.Fatalf("expected three menu items, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Direct message" {
		t.Fatalf("unexpected first menu item label: %q", menu.Items[0].Label)
	}
	if menu.Items[1].Label != "Traceroute" {
		t.Fatalf("unexpected second menu item label: %q", menu.Items[1].Label)
	}
	if menu.Items[2].Label != "Node info" {
		t.Fatalf("unexpected third menu item label: %q", menu.Items[2].Label)
	}
	for _, item := range menu.Items {
		item.Action()
	}
	if len(calledActions) != 3 {
		t.Fatalf("expected three callback invocations, got %d", len(calledActions))
	}
	if calledActions[0] != NodeActionDirectMessage {
		t.Fatalf("unexpected first action: %q", calledActions[0])
	}
	if calledActions[1] != NodeActionTraceroute {
		t.Fatalf("unexpected second action: %q", calledActions[1])
	}
	if calledActions[2] != NodeActionInfo {
		t.Fatalf("unexpected third action: %q", calledActions[2])
	}
}
