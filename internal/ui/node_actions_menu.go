package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeAction identifies a node-level action available from context menu.
type NodeAction string

const (
	NodeActionTraceroute NodeAction = "traceroute"
)

// NodeActionHandler handles selected node action menu item.
type NodeActionHandler func(node domain.Node, action NodeAction)

func newNodeContextMenu(node domain.Node, onAction NodeActionHandler) *fyne.Menu {
	menuTitle := strings.TrimSpace(nodeDisplayName(node))
	if menuTitle == "" {
		menuTitle = "Node"
	}

	return fyne.NewMenu(
		menuTitle,
		fyne.NewMenuItem("Traceroute", func() {
			if onAction != nil {
				onAction(node, NodeActionTraceroute)
			}
		}),
	)
}

func showNodeContextMenu(canvas fyne.Canvas, position fyne.Position, node domain.Node, onAction NodeActionHandler) {
	if canvas == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(newNodeContextMenu(node, onAction), canvas, position)
}
