package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewNodeContextMenu_ContainsNodeInfoAction(t *testing.T) {
	node := domain.Node{NodeID: "!0000002a", LongName: "Alpha", ShortName: "ALPH"}

	calledActions := make([]NodeAction, 0, 5)
	menu := newNodeContextMenu(node, false, func(_ domain.Node, action NodeAction) {
		calledActions = append(calledActions, action)
	})
	if menu == nil {
		t.Fatalf("expected menu")
	}
	if len(menu.Items) != 5 {
		t.Fatalf("expected five menu items, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Direct message" {
		t.Fatalf("unexpected first menu item label: %q", menu.Items[0].Label)
	}
	if menu.Items[1].Label != "Share" {
		t.Fatalf("unexpected second menu item label: %q", menu.Items[1].Label)
	}
	if menu.Items[2].Label != "Favorite" {
		t.Fatalf("unexpected third menu item label: %q", menu.Items[2].Label)
	}
	if menu.Items[3].Label != "Traceroute" {
		t.Fatalf("unexpected fourth menu item label: %q", menu.Items[3].Label)
	}
	if menu.Items[4].Label != "Node info" {
		t.Fatalf("unexpected fifth menu item label: %q", menu.Items[4].Label)
	}
	for _, item := range menu.Items {
		item.Action()
	}
	if len(calledActions) != 5 {
		t.Fatalf("expected five callback invocations, got %d", len(calledActions))
	}
	if calledActions[0] != NodeActionDirectMessage {
		t.Fatalf("unexpected first action: %q", calledActions[0])
	}
	if calledActions[1] != NodeActionShare {
		t.Fatalf("unexpected second action: %q", calledActions[1])
	}
	if calledActions[2] != NodeActionFavorite {
		t.Fatalf("unexpected third action: %q", calledActions[2])
	}
	if calledActions[3] != NodeActionTraceroute {
		t.Fatalf("unexpected fourth action: %q", calledActions[3])
	}
	if calledActions[4] != NodeActionInfo {
		t.Fatalf("unexpected fifth action: %q", calledActions[4])
	}
}

func TestNewNodeContextMenu_LocalNodeDoesNotContainFavoriteAction(t *testing.T) {
	node := domain.Node{NodeID: "!0000002a", LongName: "Alpha", ShortName: "ALPH"}

	calledActions := make([]NodeAction, 0, 4)
	menu := newNodeContextMenu(node, true, func(_ domain.Node, action NodeAction) {
		calledActions = append(calledActions, action)
	})
	if menu == nil {
		t.Fatalf("expected menu")
	}
	if len(menu.Items) != 4 {
		t.Fatalf("expected four menu items, got %d", len(menu.Items))
	}
	if menu.Items[0].Label != "Direct message" {
		t.Fatalf("unexpected first menu item label: %q", menu.Items[0].Label)
	}
	if menu.Items[1].Label != "Share" {
		t.Fatalf("unexpected second menu item label: %q", menu.Items[1].Label)
	}
	if menu.Items[2].Label != "Traceroute" {
		t.Fatalf("unexpected third menu item label: %q", menu.Items[2].Label)
	}
	if menu.Items[3].Label != "Node info" {
		t.Fatalf("unexpected fourth menu item label: %q", menu.Items[3].Label)
	}
	for _, item := range menu.Items {
		item.Action()
	}
	if len(calledActions) != 4 {
		t.Fatalf("expected four callback invocations, got %d", len(calledActions))
	}
	if calledActions[0] != NodeActionDirectMessage {
		t.Fatalf("unexpected first action: %q", calledActions[0])
	}
	if calledActions[1] != NodeActionShare {
		t.Fatalf("unexpected second action: %q", calledActions[1])
	}
	if calledActions[2] != NodeActionTraceroute {
		t.Fatalf("unexpected third action: %q", calledActions[2])
	}
	if calledActions[3] != NodeActionInfo {
		t.Fatalf("unexpected fourth action: %q", calledActions[3])
	}
}

func TestNodeFavoriteMenuLabel(t *testing.T) {
	if got := nodeFavoriteMenuLabel(domain.Node{}); got != "Favorite" {
		t.Fatalf("unexpected default favorite label: %q", got)
	}
	isFavorite := true
	if got := nodeFavoriteMenuLabel(domain.Node{IsFavorite: &isFavorite}); got != "Unfavorite" {
		t.Fatalf("unexpected favorite label for marked node: %q", got)
	}
}
