package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeRowRenderer defines create/update callbacks for nodes list row widgets.
type NodeRowRenderer struct {
	Create func() fyne.CanvasObject
	Update func(obj fyne.CanvasObject, node domain.Node)
}

// NodesTabActions contains optional callbacks for node row interactions.
type NodesTabActions struct {
	OnNodeSecondaryTapped func(node domain.Node, position fyne.Position)
}

const nodeFilterDebounce = 500 * time.Millisecond

func DefaultNodeRowRenderer() NodeRowRenderer {
	return NodeRowRenderer{
		Create: func() fyne.CanvasObject {
			nameLabel := widget.NewLabel("name")
			nameLabel.TextStyle = fyne.TextStyle{Bold: true}
			line1Right := widget.NewLabel("seen")
			line2Model := widget.NewLabel("model")
			line2Role := widget.NewLabel("role")
			line2Role.Alignment = fyne.TextAlignCenter
			line2Signal := canvas.NewText("", signalColorGood)
			line2Signal.TextStyle = fyne.TextStyle{Monospace: true}
			line2Signal.Hide()
			line2Right := widget.NewLabel("id")
			line2Right.TextStyle = fyne.TextStyle{Monospace: true}
			line2RightBox := container.NewHBox(line2Signal, line2Right)

			return container.NewVBox(
				container.NewHBox(nameLabel, layout.NewSpacer(), line1Right),
				container.NewBorder(nil, nil, line2Model, line2RightBox, line2Role),
			)
		},
		Update: func(obj fyne.CanvasObject, node domain.Node) {
			labels, ok := extractNodeRowLabels(obj)
			if !ok {
				return
			}
			labels.name.SetText(nodeDisplayName(node))
			labels.seen.SetText(nodeLine1Right(node, time.Now()))
			labels.model.SetText(nodeLine2Model(node))
			labels.role.SetText(nodeLine2Role(node))
			signal := nodeLine2Signal(node)
			labels.signal.Text = signal.Text
			labels.signal.Color = signal.Color
			if signal.Visible {
				labels.signal.Show()
			} else {
				labels.signal.Hide()
			}
			labels.signal.Refresh()
			labels.id.SetText(node.NodeID)
		},
	}
}

type nodeRowLabels struct {
	name   *widget.Label
	seen   *widget.Label
	model  *widget.Label
	role   *widget.Label
	signal *canvas.Text
	id     *widget.Label
}

type nodeRowItem struct {
	widget.BaseWidget

	content     fyne.CanvasObject
	onSecondary func(position fyne.Position)
}

var _ fyne.Tappable = (*nodeRowItem)(nil)
var _ fyne.SecondaryTappable = (*nodeRowItem)(nil)

func newNodeRowItem(content fyne.CanvasObject) *nodeRowItem {
	item := &nodeRowItem{content: content}
	item.ExtendBaseWidget(item)

	return item
}

func (r *nodeRowItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.content)
}

func (r *nodeRowItem) Tapped(*fyne.PointEvent) {}

func (r *nodeRowItem) TappedSecondary(event *fyne.PointEvent) {
	if r == nil || r.onSecondary == nil || event == nil {
		return
	}
	r.onSecondary(event.AbsolutePosition)
}

// extractNodeRowLabels maps list row container objects to typed labels.
func extractNodeRowLabels(obj fyne.CanvasObject) (nodeRowLabels, bool) {
	root, ok := obj.(*fyne.Container)
	if !ok || len(root.Objects) < 2 {
		return nodeRowLabels{}, false
	}
	line1, ok := root.Objects[0].(*fyne.Container)
	if !ok || len(line1.Objects) < 3 {
		return nodeRowLabels{}, false
	}
	line2, ok := root.Objects[1].(*fyne.Container)
	if !ok || len(line2.Objects) < 3 {
		return nodeRowLabels{}, false
	}
	name, ok := line1.Objects[0].(*widget.Label)
	if !ok {
		return nodeRowLabels{}, false
	}
	seen, ok := line1.Objects[2].(*widget.Label)
	if !ok {
		return nodeRowLabels{}, false
	}
	role, ok := line2.Objects[0].(*widget.Label)
	if !ok {
		return nodeRowLabels{}, false
	}
	model, ok := line2.Objects[1].(*widget.Label)
	if !ok {
		return nodeRowLabels{}, false
	}
	rightBox, ok := line2.Objects[2].(*fyne.Container)
	if !ok || len(rightBox.Objects) < 2 {
		return nodeRowLabels{}, false
	}
	signal, ok := rightBox.Objects[0].(*canvas.Text)
	if !ok {
		return nodeRowLabels{}, false
	}
	id, ok := rightBox.Objects[1].(*widget.Label)
	if !ok {
		return nodeRowLabels{}, false
	}

	return nodeRowLabels{
		name:   name,
		seen:   seen,
		model:  model,
		role:   role,
		signal: signal,
		id:     id,
	}, true
}

type nodeSignalView struct {
	Text    string
	Color   color.Color
	Visible bool
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

func nodeLine2Signal(node domain.Node) nodeSignalView {
	quality, ok := signalQualityFromMetrics(node.RSSI, node.SNR)
	if !ok {
		return nodeSignalView{Visible: false}
	}
	level := signalLevelLabel(quality)
	bars := signalBarsForQuality(quality)
	if level == "" || bars == "" {
		return nodeSignalView{Visible: false}
	}

	return nodeSignalView{
		Text:    fmt.Sprintf("%s %s", bars, level),
		Color:   signalColorForQuality(quality),
		Visible: true,
	}
}

func signalLevelLabel(quality domain.SignalQuality) string {
	switch quality {
	case domain.SignalGood:
		return "Good"
	case domain.SignalFair:
		return "Fair"
	case domain.SignalBad:
		return "Bad"
	default:
		return ""
	}
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
	return newNodesTabWithActions(store, renderer, NodesTabActions{})
}

func newNodesTabWithActions(store *domain.NodeStore, renderer NodeRowRenderer, actions NodesTabActions) fyne.CanvasObject {
	if store == nil {
		title := widget.NewLabel("Nodes (0)")
		filterEntry := widget.NewEntry()
		filterEntry.SetPlaceHolder("Filter nodes")
		filterEntry.Disable()
		filterSize := fyne.NewSize(260, filterEntry.MinSize().Height)
		filterWidget := container.NewGridWrap(filterSize, filterEntry)
		header := container.NewHBox(title, layout.NewSpacer(), filterWidget)
		placeholder := widget.NewLabel("Nodes are unavailable")
		placeholder.Wrapping = fyne.TextWrapWord

		return container.NewBorder(header, nil, nil, nil, container.NewCenter(placeholder))
	}

	allNodes := store.SnapshotSorted()
	appliedFilter := ""
	nodes := filterNodesByName(allNodes, appliedFilter)
	title := widget.NewLabel(nodeCountLabelText(len(allNodes), len(nodes), appliedFilter))

	list := widget.NewList(
		func() int { return len(nodes) },
		func() fyne.CanvasObject {
			return newNodeRowItem(renderer.Create())
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(nodes) {
				return
			}
			row, ok := obj.(*nodeRowItem)
			if !ok {
				return
			}
			renderer.Update(row.content, nodes[id])
			if actions.OnNodeSecondaryTapped == nil {
				row.onSecondary = nil

				return
			}
			node := nodes[id]
			row.onSecondary = func(position fyne.Position) {
				actions.OnNodeSecondaryTapped(node, position)
			}
		},
	)

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter nodes")
	filterSize := fyne.NewSize(260, filterEntry.MinSize().Height)
	filterWidget := container.NewGridWrap(filterSize, filterEntry)
	var filterDebounceSeq uint64

	applyFilter := func(value string) {
		appliedFilter = value
		nodes = filterNodesByName(allNodes, appliedFilter)
		title.SetText(nodeCountLabelText(len(allNodes), len(nodes), appliedFilter))
		list.Refresh()
	}

	filterEntry.OnChanged = func(text string) {
		seq := atomic.AddUint64(&filterDebounceSeq, 1)
		go func(localSeq uint64, localText string) {
			time.Sleep(nodeFilterDebounce)
			if atomic.LoadUint64(&filterDebounceSeq) != localSeq {
				return
			}
			fyne.Do(func() {
				applyFilter(localText)
			})
		}(seq, text)
	}

	filterEntry.OnSubmitted = func(text string) {
		atomic.AddUint64(&filterDebounceSeq, 1)
		applyFilter(text)
	}

	go func() {
		for range store.Changes() {
			fyne.Do(func() {
				allNodes = store.SnapshotSorted()
				nodes = filterNodesByName(allNodes, appliedFilter)
				title.SetText(nodeCountLabelText(len(allNodes), len(nodes), appliedFilter))
				list.Refresh()
			})
		}
	}()

	header := container.NewHBox(title, layout.NewSpacer(), filterWidget)

	return container.NewBorder(header, nil, nil, nil, list)
}

func nodeCountLabelText(total int, visible int, rawFilter string) string {
	if strings.TrimSpace(rawFilter) == "" {
		return fmt.Sprintf("Nodes (%d)", total)
	}

	return fmt.Sprintf("Nodes (%d/%d)", visible, total)
}

func filterNodesByName(nodes []domain.Node, rawFilter string) []domain.Node {
	needle := strings.ToLower(strings.TrimSpace(rawFilter))
	if needle == "" {
		out := make([]domain.Node, len(nodes))
		copy(out, nodes)

		return out
	}

	out := make([]domain.Node, 0, len(nodes))
	for _, node := range nodes {
		shortName := strings.ToLower(strings.TrimSpace(node.ShortName))
		longName := strings.ToLower(strings.TrimSpace(node.LongName))
		if strings.Contains(shortName, needle) || strings.Contains(longName, needle) {
			out = append(out, node)
		}
	}

	return out
}
