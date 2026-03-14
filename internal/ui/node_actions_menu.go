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
	NodeActionDirectMessage NodeAction = "direct_message"
	NodeActionShare         NodeAction = "share"
	NodeActionFavorite      NodeAction = "favorite"
	NodeActionTraceroute    NodeAction = "traceroute"
	NodeActionInfo          NodeAction = "info"
)

// NodeActionHandler handles selected node action menu item.
type NodeActionHandler func(node domain.Node, action NodeAction)

func newNodeContextMenu(node domain.Node, isLocal bool, onAction NodeActionHandler) *fyne.Menu {
	menuTitle := strings.TrimSpace(nodeDisplayName(node))
	if menuTitle == "" {
		menuTitle = "Node"
	}

	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Direct message", func() {
			if onAction != nil {
				onAction(node, NodeActionDirectMessage)
			}
		}),
		fyne.NewMenuItem("Share", func() {
			if onAction != nil {
				onAction(node, NodeActionShare)
			}
		}),
	}
	if !isLocal {
		items = append(items, fyne.NewMenuItem(nodeFavoriteMenuLabel(node), func() {
			if onAction != nil {
				onAction(node, NodeActionFavorite)
			}
		}))
	}
	items = append(items,
		fyne.NewMenuItem("Traceroute", func() {
			if onAction != nil {
				onAction(node, NodeActionTraceroute)
			}
		}),
		fyne.NewMenuItem("Node info", func() {
			if onAction != nil {
				onAction(node, NodeActionInfo)
			}
		}),
	)

	return fyne.NewMenu(menuTitle, items...)
}

func showNodeContextMenu(
	canvas fyne.Canvas,
	position fyne.Position,
	node domain.Node,
	isLocal bool,
	onAction NodeActionHandler,
) {
	if canvas == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(newNodeContextMenu(node, isLocal, onAction), canvas, position)
}

func nodeFavoriteMenuLabel(node domain.Node) string {
	if node.IsFavorite != nil && *node.IsFavorite {
		return "Unfavorite"
	}

	return "Favorite"
}
