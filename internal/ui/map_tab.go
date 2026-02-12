package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

const (
	mapDefaultZoom             = 11
	mapMarkerOutsidePad        = float32(20)
	mapMarkerBaseSize          = float32(20)
	mapMarkerHoverSize         = float32(23)
	mapMarkerBaseTranslucent   = 0.08
	mapMarkerHoverTranslucent  = 0.0
	mapScrollZoomStep          = float32(15)
	mapZoomFocusDeadZoneRatio  = float32(0.18)
	mapZoomFocusBoostRatio     = float32(0.65)
	mapDragPanThreshold        = float32(96)
	mapViewportPersistDebounce = 500 * time.Millisecond
)

var mapLogger = slog.With("component", "ui.map")

func newMapTab(
	store *domain.NodeStore,
	localNodeID func() string,
	paths meshapp.Paths,
	initialViewport config.MapViewportConfig,
	initialVariant fyne.ThemeVariant,
	onViewportChanged func(zoom, x, y int),
) fyne.CanvasObject {
	if store == nil {
		mapLogger.Warn("map tab is unavailable: node store is nil")
		placeholder := widget.NewLabel("Map is unavailable")
		placeholder.Wrapping = fyne.TextWrapWord

		return container.NewCenter(placeholder)
	}

	mapClient := newMapTileHTTPClient(paths.MapTilesDir, defaultMapTileCacheSizeBytes)
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(mapClient),
	)

	tab := newMapTabWidget(baseMap, localNodeID)
	mapLogger.Info(
		"initializing map tab",
		"tile_cache_dir", paths.MapTilesDir,
		"initial_viewport_set", initialViewport.Set,
		"initial_theme", initialVariant,
	)
	tab.applyThemeVariant(initialVariant)
	tab.onViewportChanged = onViewportChanged
	if initialViewport.Set {
		mapLogger.Debug(
			"applying initial map viewport",
			"zoom", initialViewport.Zoom,
			"x", initialViewport.X,
			"y", initialViewport.Y,
		)
		tab.panToViewport(mapViewportState{
			Zoom: initialViewport.Zoom,
			X:    initialViewport.X,
			Y:    initialViewport.Y,
		})
		tab.autoCentered = true
	}
	tab.setNodes(store.SnapshotSorted(), true)

	go func() {
		for range store.Changes() {
			snapshot := store.SnapshotSorted()
			fyne.Do(func() {
				mapLogger.Debug("applying node store changes to map", "node_count", len(snapshot))
				tab.setNodes(snapshot, false)
			})
		}
	}()

	return tab
}

type mapTabWidget struct {
	widget.BaseWidget

	mapWidget      *xwidget.Map
	viewState      mapViewportState
	nodes          []domain.Node
	localNodeID    func() string
	autoCentered   bool
	lastCanvasSize fyne.Size
	tooltipManager *hoverTooltipManager

	scrollAccumulator float32
	dragAccumulatorX  float32
	dragAccumulatorY  float32

	onViewportChanged  func(zoom, x, y int)
	viewportPersistSeq uint64

	interactionLayer *mapInteractionLayer
	markerLayer      *fyne.Container
	tooltipLayer     *fyne.Container
	emptyLabel       *widget.Label
	emptyLayer       *fyne.Container
	controlPanel     *fyne.Container

	markerVariant fyne.ThemeVariant
}

func newMapTabWidget(mapWidget *xwidget.Map, localNodeID func() string) *mapTabWidget {
	markerLayer := container.NewWithoutLayout()
	tooltipLayer := container.NewWithoutLayout()
	emptyLabel := widget.NewLabel("No node positions yet")
	emptyLayer := container.NewCenter(emptyLabel)

	tab := &mapTabWidget{
		mapWidget:      mapWidget,
		localNodeID:    localNodeID,
		tooltipManager: newHoverTooltipManager(tooltipLayer),
		markerLayer:    markerLayer,
		tooltipLayer:   tooltipLayer,
		emptyLabel:     emptyLabel,
		emptyLayer:     emptyLayer,
		markerVariant:  theme.VariantDark,
	}
	tab.interactionLayer = newMapInteractionLayer(tab.handleMapScroll, tab.handleMapDrag)
	tab.controlPanel = tab.newControlPanel()
	tab.ExtendBaseWidget(tab)

	return tab
}

func (t *mapTabWidget) newControlPanel() *fyne.Container {
	zoomIn := widget.NewButtonWithIcon("", theme.ZoomInIcon(), func() {
		t.mapWidget.ZoomIn()
		t.viewState.ZoomIn()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	zoomOut := widget.NewButtonWithIcon("", theme.ZoomOutIcon(), func() {
		t.mapWidget.ZoomOut()
		t.viewState.ZoomOut()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	panNorth := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		t.mapWidget.PanNorth()
		t.viewState.PanNorth()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	panSouth := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		t.mapWidget.PanSouth()
		t.viewState.PanSouth()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	panWest := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		t.mapWidget.PanWest()
		t.viewState.PanWest()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	panEast := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		t.mapWidget.PanEast()
		t.viewState.PanEast()
		t.renderMarkers()
		t.scheduleViewportPersist()
	})
	recenter := widget.NewButton("Center", func() {
		t.centerToPreferred(t.viewState.Zoom)
		t.renderMarkers()
		t.scheduleViewportPersist()
	})

	panGrid := container.NewGridWithColumns(3,
		layout.NewSpacer(),
		panNorth,
		layout.NewSpacer(),
		panWest,
		layout.NewSpacer(),
		panEast,
		layout.NewSpacer(),
		panSouth,
		layout.NewSpacer(),
	)

	return container.NewVBox(
		zoomIn,
		zoomOut,
		panGrid,
		recenter,
	)
}

func (t *mapTabWidget) setNodes(nodes []domain.Node, initial bool) {
	t.nodes = append([]domain.Node(nil), nodes...)
	mapLogger.Debug("updating map nodes", "initial", initial, "node_count", len(t.nodes))
	if initial || !t.autoCentered {
		zoom := t.viewState.Zoom
		if zoom == 0 {
			zoom = mapDefaultZoom
		}
		if t.centerToPreferred(zoom) {
			mapLogger.Info("auto-centered map viewport", "zoom", t.viewState.Zoom, "x", t.viewState.X, "y", t.viewState.Y)
			t.autoCentered = true
		} else {
			mapLogger.Debug("skipped auto-centering: no suitable node coordinates")
		}
	}
	t.renderMarkers()
}

func mapMarkerResource(variant fyne.ThemeVariant) fyne.Resource {
	res := resources.UIIconResource(resources.UIIconMapNodeMarker, variant)
	if res != nil {
		return res
	}

	return resources.UIIconResource(resources.UIIconMapNodeMarker, theme.VariantDark)
}

func (t *mapTabWidget) applyThemeVariant(variant fyne.ThemeVariant) {
	if t == nil || t.markerVariant == variant {
		return
	}
	mapLogger.Debug("applying map marker theme", "from", t.markerVariant, "to", variant)
	t.markerVariant = variant
	t.renderMarkers()
}

func (t *mapTabWidget) centerToPreferred(zoom int) bool {
	localID := ""
	if t.localNodeID != nil {
		localID = t.localNodeID()
	}
	center, ok := chooseMapCenter(t.nodes, localID)
	if !ok {
		mapLogger.Debug("map center was not resolved", "node_count", len(t.nodes), "local_node_id", localID)

		return false
	}

	mapLogger.Debug("map center resolved", "lat", center.Latitude, "lng", center.Longitude, "zoom", zoom)
	target := centerCoordinateToViewport(center, zoom)
	t.panToViewport(target)

	return true
}

func (t *mapTabWidget) panToViewport(target mapViewportState) {
	mapLogger.Debug(
		"panning map to viewport",
		"from_zoom", t.viewState.Zoom,
		"from_x", t.viewState.X,
		"from_y", t.viewState.Y,
		"to_zoom", target.Zoom,
		"to_x", target.X,
		"to_y", target.Y,
	)
	t.setZoom(target.Zoom)
	for t.viewState.X < target.X {
		t.mapWidget.PanEast()
		t.viewState.PanEast()
	}
	for t.viewState.X > target.X {
		t.mapWidget.PanWest()
		t.viewState.PanWest()
	}
	for t.viewState.Y < target.Y {
		t.mapWidget.PanSouth()
		t.viewState.PanSouth()
	}
	for t.viewState.Y > target.Y {
		t.mapWidget.PanNorth()
		t.viewState.PanNorth()
	}
}

func (t *mapTabWidget) setZoom(target int) {
	if target < 0 {
		target = 0
	}
	if target > 19 {
		target = 19
	}
	for t.viewState.Zoom < target {
		t.mapWidget.ZoomIn()
		t.viewState.ZoomIn()
	}
	for t.viewState.Zoom > target {
		t.mapWidget.ZoomOut()
		t.viewState.ZoomOut()
	}
}

func (t *mapTabWidget) renderMarkers() {
	if t.tooltipManager != nil {
		t.tooltipManager.Hide(nil)
	}
	t.markerLayer.Objects = nil

	positionedNodes := 0
	visibleMarkers := 0
	size := t.markerLayer.Size()
	for _, node := range t.nodes {
		coord, ok := nodeCoordinate(node)
		if !ok {
			continue
		}
		positionedNodes++

		pos, ok := projectCoordinateToScreen(coord, t.viewState, size)
		if !ok {
			continue
		}
		if !isMarkerVisible(pos, size) {
			continue
		}
		visibleMarkers++

		marker := newMapMarkerWidget(mapMarkerResource(t.markerVariant), mapMarkerTooltip(node), t.tooltipManager)
		markerSize := marker.MinSize()
		marker.Resize(markerSize)
		marker.Move(fyne.NewPos(
			pos.X-markerSize.Width/2,
			pos.Y-markerSize.Height,
		))
		t.markerLayer.Add(marker)
	}

	if positionedNodes == 0 {
		t.emptyLabel.Show()
	} else {
		t.emptyLabel.Hide()
	}
	t.markerLayer.Refresh()
	t.emptyLayer.Refresh()
	mapLogger.Debug(
		"rendered map markers",
		"total_nodes", len(t.nodes),
		"positioned_nodes", positionedNodes,
		"visible_markers", visibleMarkers,
		"zoom", t.viewState.Zoom,
		"x", t.viewState.X,
		"y", t.viewState.Y,
	)
}

func isMarkerVisible(pos fyne.Position, size fyne.Size) bool {
	if size.Width <= 0 || size.Height <= 0 {
		return false
	}
	if pos.X < -mapMarkerOutsidePad || pos.Y < -mapMarkerOutsidePad {
		return false
	}
	if pos.X > size.Width+mapMarkerOutsidePad || pos.Y > size.Height+mapMarkerOutsidePad {
		return false
	}

	return true
}

func mapMarkerTooltip(node domain.Node) string {
	name := nodeDisplayName(node)
	if name == "" {
		name = node.NodeID
	}
	if name == node.NodeID {
		return name
	}

	return fmt.Sprintf("%s (%s)", name, node.NodeID)
}

func (t *mapTabWidget) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{
		t.mapWidget,
		t.interactionLayer,
		t.markerLayer,
		t.emptyLayer,
		t.controlPanel,
		t.tooltipLayer,
	}

	return &mapTabRenderer{
		tab:     t,
		objects: objects,
	}
}

type mapTabRenderer struct {
	tab     *mapTabWidget
	objects []fyne.CanvasObject
}

func (r *mapTabRenderer) Layout(size fyne.Size) {
	for _, obj := range []fyne.CanvasObject{
		r.tab.mapWidget,
		r.tab.interactionLayer,
		r.tab.markerLayer,
		r.tab.emptyLayer,
		r.tab.tooltipLayer,
	} {
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(size)
	}

	panelSize := r.tab.controlPanel.MinSize()
	padding := theme.Padding()
	r.tab.controlPanel.Resize(panelSize)
	r.tab.controlPanel.Move(fyne.NewPos(
		max(0, size.Width-panelSize.Width-padding),
		padding,
	))

	if r.tab.lastCanvasSize != size {
		r.tab.lastCanvasSize = size
		r.tab.renderMarkers()
	}
}

func (r *mapTabRenderer) MinSize() fyne.Size {
	return fyne.NewSize(180, 120)
}

func (r *mapTabRenderer) Refresh() {
	for _, obj := range r.objects {
		obj.Refresh()
	}
	r.Layout(r.tab.Size())
}

func (r *mapTabRenderer) Destroy() {}

func (r *mapTabRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

type mapMarkerWidget struct {
	widget.BaseWidget

	icon    *canvas.Image
	tooltip string
	manager *hoverTooltipManager
	hovered bool
}

var _ desktop.Hoverable = (*mapMarkerWidget)(nil)
var _ fyne.Tappable = (*mapMarkerWidget)(nil)

func newMapMarkerWidget(res fyne.Resource, tooltip string, manager *hoverTooltipManager) *mapMarkerWidget {
	icon := canvas.NewImageFromResource(res)
	icon.FillMode = canvas.ImageFillContain
	icon.Translucency = mapMarkerBaseTranslucent

	m := &mapMarkerWidget{
		icon:    icon,
		tooltip: tooltip,
		manager: manager,
	}
	m.ExtendBaseWidget(m)

	return m
}

func (m *mapMarkerWidget) MinSize() fyne.Size {
	return fyne.NewSize(mapMarkerBaseSize, mapMarkerBaseSize)
}

func (m *mapMarkerWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.icon)
}

func (m *mapMarkerWidget) MouseIn(*desktop.MouseEvent) {
	m.setHovered(true)
	m.showTooltip()
}

func (m *mapMarkerWidget) MouseMoved(*desktop.MouseEvent) {}

func (m *mapMarkerWidget) MouseOut() {
	m.setHovered(false)
	if m.manager != nil {
		m.manager.Hide(m)
	}
}

func (m *mapMarkerWidget) Tapped(*fyne.PointEvent) {
	m.showTooltip()
}

func (m *mapMarkerWidget) TappedSecondary(*fyne.PointEvent) {
	m.showTooltip()
}

func (m *mapMarkerWidget) showTooltip() {
	if m.manager == nil || m.tooltip == "" {
		return
	}

	m.manager.Show(m, widget.NewLabel(m.tooltip))
}

func (m *mapMarkerWidget) setHovered(hovered bool) {
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

	newSize := fyne.NewSize(mapMarkerBaseSize, mapMarkerBaseSize)
	if hovered {
		newSize = fyne.NewSize(mapMarkerHoverSize, mapMarkerHoverSize)
		m.icon.Translucency = mapMarkerHoverTranslucent
	} else {
		m.icon.Translucency = mapMarkerBaseTranslucent
	}
	m.Resize(newSize)
	m.Move(fyne.NewPos(
		tipX-newSize.Width/2,
		tipY-newSize.Height,
	))
	m.icon.Refresh()
}

type mapInteractionLayer struct {
	widget.BaseWidget

	onScroll func(*fyne.ScrollEvent)
	onDrag   func(fyne.Delta)
	bg       *canvas.Rectangle
}

var _ fyne.Scrollable = (*mapInteractionLayer)(nil)
var _ fyne.Draggable = (*mapInteractionLayer)(nil)

func newMapInteractionLayer(onScroll func(*fyne.ScrollEvent), onDrag func(fyne.Delta)) *mapInteractionLayer {
	bg := canvas.NewRectangle(color.Transparent)
	layer := &mapInteractionLayer{
		onScroll: onScroll,
		onDrag:   onDrag,
		bg:       bg,
	}
	layer.ExtendBaseWidget(layer)

	return layer
}

func (l *mapInteractionLayer) Scrolled(event *fyne.ScrollEvent) {
	if l == nil || l.onScroll == nil || event == nil {
		return
	}

	l.onScroll(event)
}

func (l *mapInteractionLayer) Dragged(event *fyne.DragEvent) {
	if l == nil || l.onDrag == nil || event == nil {
		return
	}

	l.onDrag(event.Dragged)
}

func (l *mapInteractionLayer) DragEnd() {}

func (l *mapInteractionLayer) MinSize() fyne.Size {
	return fyne.NewSize(1, 1)
}

func (l *mapInteractionLayer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.bg)
}

func (t *mapTabWidget) handleMapScroll(event *fyne.ScrollEvent) {
	if t == nil || event == nil {
		return
	}

	delta := event.Scrolled
	primary := delta.DY
	if math.Abs(float64(delta.DX)) > math.Abs(float64(primary)) {
		primary = delta.DX
	}
	if primary == 0 {
		return
	}
	t.scrollAccumulator += primary

	changed := false
	for t.scrollAccumulator >= mapScrollZoomStep {
		t.mapWidget.ZoomIn()
		t.viewState.ZoomIn()
		t.applyDirectionalZoomPan(true, event.Position)
		t.scrollAccumulator -= mapScrollZoomStep
		changed = true
	}
	for t.scrollAccumulator <= -mapScrollZoomStep {
		t.mapWidget.ZoomOut()
		t.viewState.ZoomOut()
		t.scrollAccumulator += mapScrollZoomStep
		changed = true
	}
	if changed {
		if t.tooltipManager != nil {
			t.tooltipManager.Hide(nil)
		}
		mapLogger.Debug("map viewport changed by scroll", "zoom", t.viewState.Zoom, "x", t.viewState.X, "y", t.viewState.Y)
		t.renderMarkers()
		t.scheduleViewportPersist()
	}
}

func (t *mapTabWidget) applyDirectionalZoomPan(zoomIn bool, cursor fyne.Position) {
	if t == nil {
		return
	}

	size := t.Size()
	if size.Width <= 0 || size.Height <= 0 {
		size = t.lastCanvasSize
	}
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	halfWidth := size.Width / 2
	halfHeight := size.Height / 2
	if halfWidth <= 0 || halfHeight <= 0 {
		return
	}

	dxNorm := (cursor.X - halfWidth) / halfWidth
	dyNorm := (cursor.Y - halfHeight) / halfHeight
	dxNorm = max(-1, min(1, dxNorm))
	dyNorm = max(-1, min(1, dyNorm))

	xSteps := mapZoomFocusPanSteps(dxNorm)
	ySteps := mapZoomFocusPanSteps(dyNorm)
	if !zoomIn {
		xSteps = -xSteps
		ySteps = -ySteps
	}

	xCount := xSteps
	if xCount < 0 {
		xCount = -xCount
	}
	for i := 0; i < xCount; i++ {
		if xSteps > 0 {
			t.mapWidget.PanEast()
			t.viewState.PanEast()
		} else {
			t.mapWidget.PanWest()
			t.viewState.PanWest()
		}
	}
	yCount := ySteps
	if yCount < 0 {
		yCount = -yCount
	}
	for i := 0; i < yCount; i++ {
		if ySteps > 0 {
			t.mapWidget.PanSouth()
			t.viewState.PanSouth()
		} else {
			t.mapWidget.PanNorth()
			t.viewState.PanNorth()
		}
	}
}

func mapZoomFocusPanSteps(norm float32) int {
	absNorm := float32(math.Abs(float64(norm)))
	if absNorm < mapZoomFocusDeadZoneRatio {
		return 0
	}

	steps := 1
	if absNorm >= mapZoomFocusBoostRatio {
		steps = 2
	}
	if norm < 0 {
		return -steps
	}

	return steps
}

func (t *mapTabWidget) handleMapDrag(delta fyne.Delta) {
	if t == nil {
		return
	}
	if delta.DX == 0 && delta.DY == 0 {
		return
	}

	t.dragAccumulatorX += delta.DX
	t.dragAccumulatorY += delta.DY

	changed := false
	for t.dragAccumulatorX >= mapDragPanThreshold {
		// Dragging map right should move viewport west.
		t.mapWidget.PanWest()
		t.viewState.PanWest()
		t.dragAccumulatorX -= mapDragPanThreshold
		changed = true
	}
	for t.dragAccumulatorX <= -mapDragPanThreshold {
		// Dragging map left should move viewport east.
		t.mapWidget.PanEast()
		t.viewState.PanEast()
		t.dragAccumulatorX += mapDragPanThreshold
		changed = true
	}
	for t.dragAccumulatorY >= mapDragPanThreshold {
		// Dragging map down should move viewport north.
		t.mapWidget.PanNorth()
		t.viewState.PanNorth()
		t.dragAccumulatorY -= mapDragPanThreshold
		changed = true
	}
	for t.dragAccumulatorY <= -mapDragPanThreshold {
		// Dragging map up should move viewport south.
		t.mapWidget.PanSouth()
		t.viewState.PanSouth()
		t.dragAccumulatorY += mapDragPanThreshold
		changed = true
	}
	if changed {
		if t.tooltipManager != nil {
			t.tooltipManager.Hide(nil)
		}
		mapLogger.Debug("map viewport changed by drag", "zoom", t.viewState.Zoom, "x", t.viewState.X, "y", t.viewState.Y)
		t.renderMarkers()
		t.scheduleViewportPersist()
	}
}

func (t *mapTabWidget) scheduleViewportPersist() {
	if t == nil || t.onViewportChanged == nil {
		return
	}

	state := t.viewState
	seq := atomic.AddUint64(&t.viewportPersistSeq, 1)
	mapLogger.Debug("scheduled map viewport persistence", "seq", seq, "zoom", state.Zoom, "x", state.X, "y", state.Y)
	go func(localSeq uint64, localState mapViewportState) {
		time.Sleep(mapViewportPersistDebounce)
		if atomic.LoadUint64(&t.viewportPersistSeq) != localSeq {
			mapLogger.Debug("skipping stale map viewport persistence", "seq", localSeq)

			return
		}
		mapLogger.Info("persisting map viewport", "zoom", localState.Zoom, "x", localState.X, "y", localState.Y)
		t.onViewportChanged(localState.Zoom, localState.X, localState.Y)
	}(seq, state)
}
