package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

type NodeRowRenderer struct {
	Create func() fyne.CanvasObject
	Update func(obj fyne.CanvasObject, node domain.Node)
}

func DefaultNodeRowRenderer() NodeRowRenderer {
	return NodeRowRenderer{
		Create: func() fyne.CanvasObject {
			nameLabel := widget.NewLabel("name")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			line1Right := widget.NewLabel("seen")
			line2Model := widget.NewLabel("model")
			line2Role := widget.NewLabel("role")
			line2Role.Alignment = fyne.TextAlignCenter
			line2Right := widget.NewLabel("id")
			line2Right.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewVBox(
				container.NewHBox(nameLabel, layout.NewSpacer(), line1Right),
				container.NewBorder(nil, nil, line2Model, line2Right, line2Role),
			)
		},
		Update: func(obj fyne.CanvasObject, node domain.Node) {
			root := obj.(*fyne.Container)
			line1 := root.Objects[0].(*fyne.Container)
			line2 := root.Objects[1].(*fyne.Container)
			line1Left := line1.Objects[0].(*widget.Label)
			line1Right := line1.Objects[2].(*widget.Label)
			line2Role := line2.Objects[0].(*widget.Label)
			line2Model := line2.Objects[1].(*widget.Label)
			line2Right := line2.Objects[2].(*widget.Label)

			line1Left.SetText(nodeDisplayName(node))
			line1Right.SetText(nodeLine1Right(node, time.Now()))
			line2Model.SetText(nodeLine2Model(node))
			line2Role.SetText(nodeLine2Role(node))
			line2Right.SetText(node.NodeID)
		},
	}
}

func nodeLine1Right(node domain.Node, now time.Time) string {
	parts := []string{}
	if charge := nodeCharge(node); charge != "" {
		parts = append(parts, charge)
	}
	parts = append(parts, formatSeenAgo(node.LastHeardAt, now))
	return strings.Join(parts, " | ")
}

func nodeLine2Model(node domain.Node) string {
	if v := strings.TrimSpace(node.BoardModel); v != "" {
		return v
	}
	return "Unknown device"
}

func nodeLine2Role(node domain.Node) string {
	if v := strings.TrimSpace(node.Role); v != "" {
		return v
	}
	return ""
}

func nodeDisplayName(node domain.Node) string {
	shortName := strings.TrimSpace(node.ShortName)
	longName := strings.TrimSpace(node.LongName)
	var base string
	switch {
	case shortName != "" && longName != "":
		base = fmt.Sprintf("[%s] %s", shortName, longName)
	case longName != "":
		base = longName
	case shortName != "":
		base = fmt.Sprintf("[%s]", shortName)
	default:
		base = node.NodeID
	}
	if node.IsUnmessageable != nil && *node.IsUnmessageable {
		return base + " {INFRA}"
	}
	return base
}

func nodeCharge(node domain.Node) string {
	if node.BatteryLevel == nil {
		return ""
	}
	v := *node.BatteryLevel
	if v > 100 {
		return "Charge: ext"
	}
	return fmt.Sprintf("Charge: %d%%", v)
}

func formatSeenAgo(lastSeen time.Time, now time.Time) string {
	if lastSeen.IsZero() {
		return "seen: ?"
	}
	d := now.Sub(lastSeen)
	if d < 0 {
		d = 0
	}

	if d < time.Hour {
		minutes := maxRounded(d.Minutes())
		return fmt.Sprintf("%d min", minutes)
	}
	if d < 24*time.Hour {
		hours := maxRounded(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := maxRounded(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func maxRounded(v float64) int {
	n := int(math.Round(v))
	if n < 1 {
		return 1
	}
	return n
}

func newNodesTab(store *domain.NodeStore, renderer NodeRowRenderer) fyne.CanvasObject {
	nodes := store.SnapshotSorted()

	list := widget.NewList(
		func() int { return len(nodes) },
		renderer.Create,
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(nodes) {
				return
			}
			renderer.Update(obj, nodes[id])
		},
	)

	go func() {
		for range store.Changes() {
			fyne.Do(func() {
				nodes = store.SnapshotSorted()
				list.Refresh()
			})
		}
	}()

	return container.NewBorder(widget.NewLabel("Nodes"), nil, nil, nil, list)
}
