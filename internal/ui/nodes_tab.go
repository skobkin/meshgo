package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
			return container.NewVBox(widget.NewLabel("node"), widget.NewLabel("details"))
		},
		Update: func(obj fyne.CanvasObject, node domain.Node) {
			box := obj.(*fyne.Container)
			name := node.LongName
			if name == "" {
				name = node.ShortName
			}
			if name == "" {
				name = node.NodeID
			}
			details := fmt.Sprintf("id=%s heard=%s", node.NodeID, node.LastHeardAt.Format(time.RFC3339))
			if node.RSSI != nil {
				details += fmt.Sprintf(" rssi=%d", *node.RSSI)
			}
			if node.SNR != nil {
				details += fmt.Sprintf(" snr=%.2f", *node.SNR)
			}
			box.Objects[0].(*widget.Label).SetText(name)
			box.Objects[1].(*widget.Label).SetText(details)
		},
	}
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
