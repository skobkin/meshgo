package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewNodeContextMenu_ContainsTracerouteAction(t *testing.T) {
	node := domain.Node{NodeID: "!0000002a", LongName: "Alpha", ShortName: "ALPH"}

	var calledAction NodeAction
	menu := newNodeContextMenu(node, func(_ domain.Node, action NodeAction) {
		calledAction = action
	})
	if menu == nil {
		t.Fatalf("expected menu")
	}
	if len(menu.Items) != 1 {
		t.Fatalf("expected one menu item, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Traceroute" {
		t.Fatalf("unexpected menu item label: %q", menu.Items[0].Label)
	}

	menu.Items[0].Action()
	if calledAction != NodeActionTraceroute {
		t.Fatalf("unexpected action: %q", calledAction)
	}
}
