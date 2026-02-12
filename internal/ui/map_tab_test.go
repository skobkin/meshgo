package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	xwidget "fyne.io/x/fyne/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewMapTabNilStoreShowsPlaceholder(t *testing.T) {
	tab := newMapTab(nil, nil, meshapp.Paths{}, config.MapViewportConfig{}, nil)
	_ = fynetest.NewTempWindow(t, tab)

	if !hasLabelText(tab, "Map is unavailable") {
		t.Fatalf("expected map unavailable placeholder")
	}
}

func TestNewMapTab_UsesSavedViewportWhenProvided(t *testing.T) {
	store := domain.NewNodeStore()
	tabObj := newMapTab(
		store,
		nil,
		meshapp.Paths{},
		config.MapViewportConfig{Set: true, Zoom: 6, X: 5, Y: -3},
		nil,
	)
	tab, ok := tabObj.(*mapTabWidget)
	if !ok {
		t.Fatalf("expected map tab widget")
	}
	if tab.viewState.Zoom != 6 || tab.viewState.X != 5 || tab.viewState.Y != -3 {
		t.Fatalf("expected saved viewport to be applied, got %+v", tab.viewState)
	}
	if !tab.autoCentered {
		t.Fatalf("expected auto centering to be disabled when saved viewport exists")
	}
}

func TestMapTabWidget_InitialCenterUsesLocalNodePosition(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, func() string { return "!local" })
	localLat := 37.7749
	localLon := -122.4194
	nodes := []domain.Node{
		{NodeID: "!local", Latitude: &localLat, Longitude: &localLon},
	}

	tab.setNodes(nodes, true)

	expected := centerCoordinateToViewport(mapCoordinate{Latitude: localLat, Longitude: localLon}, mapDefaultZoom)
	if tab.viewState != expected {
		t.Fatalf("unexpected map center state: got %+v want %+v", tab.viewState, expected)
	}
}

func TestMapTabWidget_RendersMarkersForPositionedNodes(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	lat := 37.7749
	lon := -122.4194
	tab.setNodes([]domain.Node{
		{NodeID: "!local", Latitude: &lat, Longitude: &lon},
	}, true)

	window := fynetest.NewTempWindow(t, tab)
	window.Resize(fyne.NewSize(800, 600))
	tab.Refresh()

	if len(tab.markerLayer.Objects) == 0 {
		t.Fatalf("expected at least one marker")
	}
	if tab.emptyLabel.Visible() {
		t.Fatalf("empty label should be hidden when markers are available")
	}
}

func TestMapTabWidget_ShowsEmptyLabelWhenNoCoordinates(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	tab.setNodes([]domain.Node{{NodeID: "!no-coords"}}, true)

	_ = fynetest.NewTempWindow(t, tab)
	tab.Refresh()

	if !tab.emptyLabel.Visible() {
		t.Fatalf("expected empty label when no coordinates are available")
	}
	if len(tab.markerLayer.Objects) != 0 {
		t.Fatalf("expected no markers without coordinates")
	}
}

func TestMapTabWidget_HandleMapScrollZooms(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	tab.Resize(fyne.NewSize(800, 600))
	tab.viewState = mapViewportState{Zoom: 5, X: 2, Y: -3}

	tab.handleMapScroll(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(400, 300)},
		Scrolled:   fyne.NewDelta(0, mapScrollZoomStep),
	})
	if tab.viewState.Zoom != 6 {
		t.Fatalf("expected zoom to increase, got %d", tab.viewState.Zoom)
	}
	if tab.viewState.X != 4 || tab.viewState.Y != -6 {
		t.Fatalf("expected viewport to follow zoom transform, got x=%d y=%d", tab.viewState.X, tab.viewState.Y)
	}

	tab.handleMapScroll(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(400, 300)},
		Scrolled:   fyne.NewDelta(0, -mapScrollZoomStep),
	})
	if tab.viewState.Zoom != 5 {
		t.Fatalf("expected zoom to decrease, got %d", tab.viewState.Zoom)
	}
	if tab.viewState.X != 2 || tab.viewState.Y != -3 {
		t.Fatalf("expected viewport to return after inverse zoom, got x=%d y=%d", tab.viewState.X, tab.viewState.Y)
	}
}

func TestMapTabWidget_HandleMapScrollZoomInPansTowardCursor(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	tab.Resize(fyne.NewSize(800, 600))
	tab.viewState = mapViewportState{Zoom: 5, X: 0, Y: 0}

	tab.handleMapScroll(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(780, 580)},
		Scrolled:   fyne.NewDelta(0, mapScrollZoomStep),
	})

	if tab.viewState.Zoom != 6 {
		t.Fatalf("expected zoom to increase, got %d", tab.viewState.Zoom)
	}
	if tab.viewState.X <= 0 {
		t.Fatalf("expected zoom-in near right edge to pan east, got x=%d", tab.viewState.X)
	}
	if tab.viewState.Y <= 0 {
		t.Fatalf("expected zoom-in near bottom edge to pan south, got y=%d", tab.viewState.Y)
	}
}

func TestMapTabWidget_HandleMapScrollZoomOutIsNonDirectional(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	tab.Resize(fyne.NewSize(800, 600))
	tab.viewState = mapViewportState{Zoom: 6, X: 6, Y: 6}

	tab.handleMapScroll(&fyne.ScrollEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(780, 580)},
		Scrolled:   fyne.NewDelta(0, -mapScrollZoomStep),
	})

	if tab.viewState.Zoom != 5 {
		t.Fatalf("expected zoom to decrease, got %d", tab.viewState.Zoom)
	}
	if tab.viewState.X != 3 {
		t.Fatalf("expected non-directional zoom-out x=3, got x=%d", tab.viewState.X)
	}
	if tab.viewState.Y != 3 {
		t.Fatalf("expected non-directional zoom-out y=3, got y=%d", tab.viewState.Y)
	}
}

func TestMapTabWidget_HandleMapDragPansInExpectedDirections(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	tab := newMapTabWidget(baseMap, nil)
	tab.viewState = mapViewportState{Zoom: 5, X: 0, Y: 0}

	tab.handleMapDrag(fyne.NewDelta(mapDragPanThreshold, 0))
	if tab.viewState.X != -1 {
		t.Fatalf("drag-right should pan west (x-1), got x=%d", tab.viewState.X)
	}

	tab.handleMapDrag(fyne.NewDelta(-mapDragPanThreshold, 0))
	if tab.viewState.X != 0 {
		t.Fatalf("drag-left should pan east (x+1), got x=%d", tab.viewState.X)
	}

	tab.handleMapDrag(fyne.NewDelta(0, mapDragPanThreshold))
	if tab.viewState.Y != -1 {
		t.Fatalf("drag-down should pan north (y-1), got y=%d", tab.viewState.Y)
	}

	tab.handleMapDrag(fyne.NewDelta(0, -mapDragPanThreshold))
	if tab.viewState.Y != 0 {
		t.Fatalf("drag-up should pan south (y+1), got y=%d", tab.viewState.Y)
	}
}

func TestMapTabWidget_ScheduleViewportPersistDebouncesAndSavesLatest(t *testing.T) {
	baseMap := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.WithHTTPClient(stubTileClient(t)),
	)
	updates := make(chan mapViewportState, 2)
	tab := newMapTabWidget(baseMap, nil)
	tab.onViewportChanged = func(zoom, x, y int) {
		updates <- mapViewportState{Zoom: zoom, X: x, Y: y}
	}

	tab.viewState = mapViewportState{Zoom: 5, X: 0, Y: 0}
	tab.handleMapDrag(fyne.NewDelta(mapDragPanThreshold, 0))
	tab.handleMapDrag(fyne.NewDelta(mapDragPanThreshold, 0))

	select {
	case got := <-updates:
		if got.X != -2 || got.Zoom != 5 || got.Y != 0 {
			t.Fatalf("unexpected persisted viewport: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for viewport save")
	}

	select {
	case <-updates:
		t.Fatalf("expected debounced single update")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestMapMarkerWidget_HoverKeepsTipAnchorAndChangesVisualState(t *testing.T) {
	marker := newMapMarkerWidget(mapMarkerPinResource, "node", nil)
	baseSize := marker.MinSize()
	marker.Resize(baseSize)
	marker.Move(fyne.NewPos(100, 200))

	baseTipX := marker.Position().X + marker.Size().Width/2
	baseTipY := marker.Position().Y + marker.Size().Height

	marker.setHovered(true)
	if marker.Size().Width <= baseSize.Width {
		t.Fatalf("expected marker to grow on hover")
	}
	if marker.icon.Translucency != mapMarkerHoverTranslucent {
		t.Fatalf("expected hover translucency, got %v", marker.icon.Translucency)
	}
	hoverTipX := marker.Position().X + marker.Size().Width/2
	hoverTipY := marker.Position().Y + marker.Size().Height
	if hoverTipX != baseTipX || hoverTipY != baseTipY {
		t.Fatalf("expected marker tip to remain anchored on hover")
	}

	marker.setHovered(false)
	if marker.Size().Width != baseSize.Width || marker.Size().Height != baseSize.Height {
		t.Fatalf("expected marker size to reset on mouse out")
	}
	if marker.icon.Translucency != mapMarkerBaseTranslucent {
		t.Fatalf("expected base translucency, got %v", marker.icon.Translucency)
	}
	resetTipX := marker.Position().X + marker.Size().Width/2
	resetTipY := marker.Position().Y + marker.Size().Height
	if resetTipX != baseTipX || resetTipY != baseTipY {
		t.Fatalf("expected marker tip to remain anchored after reset")
	}
}

func stubTileClient(t *testing.T) *http.Client {
	t.Helper()

	tile := mustStubPNG(t)

	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Header:        http.Header{"Content-Type": []string{"image/png"}},
				Body:          io.NopCloser(bytes.NewReader(tile)),
				ContentLength: int64(len(tile)),
				Request:       req,
			}, nil
		}),
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func mustStubPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	return buf.Bytes()
}
