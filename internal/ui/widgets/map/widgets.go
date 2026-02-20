package mapwidgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/ui/widgets"
)

const (
	// MapMarkerBaseSize is the base size of a map marker in pixels.
	MapMarkerBaseSize float32 = 20
	// MapMarkerHoverSize is the enlarged size of a map marker when hovered.
	MapMarkerHoverSize float32 = 23
	// MapMarkerBaseTranslucent is the base translucency level of a map marker (0-1).
	MapMarkerBaseTranslucent float64 = 0.08
	// MapMarkerHoverTranslucent is the translucency level when a map marker is hovered.
	MapMarkerHoverTranslucent float64 = 0.0
)

// MapMarkerWidget is an interactive map marker with hover effects and tooltip support.
type MapMarkerWidget struct {
	widget.BaseWidget

	Icon          *canvas.Image
	tooltip       string
	manager       *widgets.HoverTooltipManager
	hovered       bool
	onHoverChange func(hovered bool)
}

var _ desktop.Hoverable = (*MapMarkerWidget)(nil)
var _ fyne.Tappable = (*MapMarkerWidget)(nil)

// NewMapMarkerWidget creates a new map marker widget with the specified icon, tooltip, and tooltip manager.
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

// SetHoverChangeHandler updates callback invoked when marker hover state changes.
func (m *MapMarkerWidget) SetHoverChangeHandler(handler func(hovered bool)) {
	if m == nil {
		return
	}
	m.onHoverChange = handler
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
	if m.onHoverChange != nil {
		m.onHoverChange(hovered)
	}
}

// MapInteractionLayer is a transparent widget layer that handles scroll and drag events for map interaction.
type MapInteractionLayer struct {
	widget.BaseWidget

	onScroll func(*fyne.ScrollEvent)
	onDrag   func(fyne.Position, fyne.Delta)
	bg       *canvas.Rectangle
}

var _ fyne.Scrollable = (*MapInteractionLayer)(nil)
var _ fyne.Draggable = (*MapInteractionLayer)(nil)

// NewMapInteractionLayer creates a new interaction layer with the specified scroll and drag handlers.
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
