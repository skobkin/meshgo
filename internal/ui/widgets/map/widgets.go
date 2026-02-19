package mapwidgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/ui/widgets"
)

const (
	MapMarkerBaseSize         float32 = 20
	MapMarkerHoverSize        float32 = 23
	MapMarkerBaseTranslucent  float64 = 0.08
	MapMarkerHoverTranslucent float64 = 0.0
)

type MapMarkerWidget struct {
	widget.BaseWidget

	Icon    *canvas.Image
	tooltip string
	manager *widgets.HoverTooltipManager
	hovered bool
}

var _ desktop.Hoverable = (*MapMarkerWidget)(nil)
var _ fyne.Tappable = (*MapMarkerWidget)(nil)

func NewMapMarkerWidget(res fyne.Resource, tooltip string, manager *widgets.HoverTooltipManager) *MapMarkerWidget {
	icon := canvas.NewImageFromResource(res)
	icon.FillMode = canvas.ImageFillContain
	icon.Translucency = MapMarkerBaseTranslucent

	m := &MapMarkerWidget{
		Icon:    icon,
		tooltip: tooltip,
		manager: manager,
	}
	m.ExtendBaseWidget(m)

	return m
}

func (m *MapMarkerWidget) MinSize() fyne.Size {
	return fyne.NewSize(MapMarkerBaseSize, MapMarkerBaseSize)
}

func (m *MapMarkerWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.Icon)
}

func (m *MapMarkerWidget) MouseIn(*desktop.MouseEvent) {
	m.SetHovered(true)
	m.showTooltip()
}

func (m *MapMarkerWidget) MouseMoved(*desktop.MouseEvent) {}

func (m *MapMarkerWidget) MouseOut() {
	m.SetHovered(false)
	if m.manager != nil {
		m.manager.Hide(m)
	}
}

func (m *MapMarkerWidget) Tapped(*fyne.PointEvent) {
	m.showTooltip()
}

func (m *MapMarkerWidget) TappedSecondary(*fyne.PointEvent) {
	m.showTooltip()
}

func (m *MapMarkerWidget) showTooltip() {
	if m.manager == nil || m.tooltip == "" {
		return
	}

	m.manager.Show(m, widget.NewLabel(m.tooltip))
}

func (m *MapMarkerWidget) SetHovered(hovered bool) {
	if m == nil || m.hovered == hovered {
		return
	}
	m.hovered = hovered

	oldSize := m.Size()
	if oldSize.Width <= 0 || oldSize.Height <= 0 {
		oldSize = m.MinSize()
	}
	oldPos := m.Position()
	tipX := oldPos.X + oldSize.Width/2
	tipY := oldPos.Y + oldSize.Height

	newSize := fyne.NewSize(MapMarkerBaseSize, MapMarkerBaseSize)
	if hovered {
		newSize = fyne.NewSize(MapMarkerHoverSize, MapMarkerHoverSize)
		m.Icon.Translucency = MapMarkerHoverTranslucent
	} else {
		m.Icon.Translucency = MapMarkerBaseTranslucent
	}
	m.Resize(newSize)
	m.Move(fyne.NewPos(
		tipX-newSize.Width/2,
		tipY-newSize.Height,
	))
	m.Icon.Refresh()
}

type MapInteractionLayer struct {
	widget.BaseWidget

	onScroll func(*fyne.ScrollEvent)
	onDrag   func(fyne.Position, fyne.Delta)
	bg       *canvas.Rectangle
}

var _ fyne.Scrollable = (*MapInteractionLayer)(nil)
var _ fyne.Draggable = (*MapInteractionLayer)(nil)

func NewMapInteractionLayer(onScroll func(*fyne.ScrollEvent), onDrag func(fyne.Position, fyne.Delta)) *MapInteractionLayer {
	bg := canvas.NewRectangle(color.Transparent)
	layer := &MapInteractionLayer{
		onScroll: onScroll,
		onDrag:   onDrag,
		bg:       bg,
	}
	layer.ExtendBaseWidget(layer)

	return layer
}

func (l *MapInteractionLayer) Scrolled(event *fyne.ScrollEvent) {
	if l == nil || l.onScroll == nil || event == nil {
		return
	}

	l.onScroll(event)
}

func (l *MapInteractionLayer) Dragged(event *fyne.DragEvent) {
	if l == nil || l.onDrag == nil || event == nil {
		return
	}

	l.onDrag(event.Position, event.Dragged)
}

func (l *MapInteractionLayer) DragEnd() {}

func (l *MapInteractionLayer) MinSize() fyne.Size {
	return fyne.NewSize(1, 1)
}

func (l *MapInteractionLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.bg)
}

type MapProgressPlacement int

const (
	MapProgressPlacementCenter MapProgressPlacement = iota
	MapProgressPlacementTop
)

type MapProgressIndicator struct {
	Layer  *fyne.Container
	Label  *widget.Label
	Bar    *widget.ProgressBar
	Action *widget.Button
}

func NewMapProgressIndicator(placement MapProgressPlacement, labelText, actionText string, width, height float32) *MapProgressIndicator {
	bar := widget.NewProgressBar()
	bar.Min = 0
	bar.Max = 1
	barWrap := container.NewGridWrap(
		fyne.NewSize(width, height),
		bar,
	)

	parts := make([]fyne.CanvasObject, 0, 3)
	var label *widget.Label
	if labelText != "" {
		label = widget.NewLabel(labelText)
		label.Alignment = fyne.TextAlignCenter
		label.Wrapping = fyne.TextWrapWord
		parts = append(parts, label)
	}
	parts = append(parts, barWrap)

	var action *widget.Button
	if actionText != "" {
		action = widget.NewButton(actionText, nil)
		action.Hide()
		parts = append(parts, action)
	}

	content := container.NewVBox(parts...)
	var layer *fyne.Container
	switch placement {
	case MapProgressPlacementTop:
		layer = container.NewBorder(
			container.NewPadded(container.NewCenter(content)),
			nil,
			nil,
			nil,
			nil,
		)
	default:
		layer = container.NewCenter(container.NewPadded(content))
	}
	layer.Hide()

	return &MapProgressIndicator{
		Layer:  layer,
		Label:  label,
		Bar:    bar,
		Action: action,
	}
}
