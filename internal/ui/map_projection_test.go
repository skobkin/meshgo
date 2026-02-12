package ui

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestChooseMapCenter_PrefersLocalNode(t *testing.T) {
	localLat := 37.7749
	localLon := -122.4194
	nodes := []domain.Node{
		{
			NodeID:    "!local",
			Latitude:  &localLat,
			Longitude: &localLon,
		},
		{
			NodeID: "!remote",
			Latitude: func() *float64 {
				v := 52.5200

				return &v
			}(),
			Longitude: func() *float64 {
				v := 13.4050

				return &v
			}(),
		},
	}

	center, ok := chooseMapCenter(nodes, "!local")
	if !ok {
		t.Fatalf("expected center")
	}
	if math.Abs(center.Latitude-localLat) > 0.000001 {
		t.Fatalf("unexpected latitude: %v", center.Latitude)
	}
	if math.Abs(center.Longitude-localLon) > 0.000001 {
		t.Fatalf("unexpected longitude: %v", center.Longitude)
	}
}

func TestRobustClusterCenter_TrimsFarOutlier(t *testing.T) {
	points := []mapCoordinate{
		{Latitude: 37.7740, Longitude: -122.4190},
		{Latitude: 37.7750, Longitude: -122.4180},
		{Latitude: 37.7760, Longitude: -122.4200},
		{Latitude: 37.7770, Longitude: -122.4210},
		{Latitude: 60.0000, Longitude: 10.0000},
	}

	center, ok := robustClusterCenter(points)
	if !ok {
		t.Fatalf("expected center")
	}
	if math.Abs(center.Latitude-37.7755) > 0.01 {
		t.Fatalf("unexpected latitude center: %v", center.Latitude)
	}
	if math.Abs(center.Longitude+122.4195) > 0.01 {
		t.Fatalf("unexpected longitude center: %v", center.Longitude)
	}
}

func TestProjectCoordinateToScreen_CenterCoordinateAppearsInMiddle(t *testing.T) {
	center := mapCoordinate{Latitude: 37.7749, Longitude: -122.4194}
	view := centerCoordinateToViewport(center, 8)
	pos, ok := projectCoordinateToScreen(center, view, fyne.NewSize(1000, 700))
	if !ok {
		t.Fatalf("expected projection")
	}
	if math.Abs(float64(pos.X-500)) > mapTileSize/2 {
		t.Fatalf("expected near horizontal center, got %v", pos.X)
	}
	if math.Abs(float64(pos.Y-350)) > mapTileSize/2 {
		t.Fatalf("expected near vertical center, got %v", pos.Y)
	}
}

func TestMapViewportState_ZoomStepMatchesMapWidgetBehavior(t *testing.T) {
	state := mapViewportState{Zoom: 3, X: 2, Y: -1}
	state.ZoomIn()
	if state.Zoom != 4 {
		t.Fatalf("expected zoom 4, got %d", state.Zoom)
	}
	if state.X != 4 {
		t.Fatalf("expected x=4 after zoom in, got %d", state.X)
	}
	if state.Y != -2 {
		t.Fatalf("expected y=-2 after zoom in, got %d", state.Y)
	}

	state.ZoomOut()
	if state.Zoom != 3 {
		t.Fatalf("expected zoom 3, got %d", state.Zoom)
	}
	if state.X != 2 {
		t.Fatalf("expected x=2 after zoom out, got %d", state.X)
	}
	if state.Y != -1 {
		t.Fatalf("expected y=-1 after zoom out, got %d", state.Y)
	}
}

func TestProjectCoordinateToScreen_TracksMapPanExactlyByOneTile(t *testing.T) {
	center := mapCoordinate{Latitude: 37.7749, Longitude: -122.4194}
	view := centerCoordinateToViewport(center, 8)
	size := fyne.NewSize(1000, 700)

	before, ok := projectCoordinateToScreen(center, view, size)
	if !ok {
		t.Fatalf("expected projection before pan")
	}
	view.PanEast()
	after, ok := projectCoordinateToScreen(center, view, size)
	if !ok {
		t.Fatalf("expected projection after pan")
	}

	shift := before.X - after.X
	if math.Abs(float64(shift-mapTileSize)) > 0.01 {
		t.Fatalf("expected one tile shift (%d), got %v", mapTileSize, shift)
	}
}
