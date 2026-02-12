package ui

import (
	"math"
	"sort"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/domain"
)

const (
	mapTileSize         = 256
	mapMaxLatitudeMerc  = 85.05112878
	mapOutlierThreshold = 3.5
)

type mapCoordinate struct {
	Latitude  float64
	Longitude float64
}

type mapViewportState struct {
	Zoom int
	X    int
	Y    int
}

func (s *mapViewportState) PanEast() {
	s.X++
}

func (s *mapViewportState) PanWest() {
	s.X--
}

func (s *mapViewportState) PanNorth() {
	s.Y--
}

func (s *mapViewportState) PanSouth() {
	s.Y++
}

func (s *mapViewportState) ZoomIn() {
	if s.Zoom >= 19 {
		return
	}
	s.Zoom++
	s.X *= 2
	s.Y *= 2
}

func (s *mapViewportState) ZoomOut() {
	if s.Zoom <= 0 {
		return
	}
	s.X /= 2
	s.Y /= 2
	s.Zoom--
}

func (s *mapViewportState) SetZoom(target int) {
	if target < 0 {
		target = 0
	}
	if target > 19 {
		target = 19
	}

	for s.Zoom < target {
		s.ZoomIn()
	}
	for s.Zoom > target {
		s.ZoomOut()
	}
}

func nodeCoordinate(node domain.Node) (mapCoordinate, bool) {
	if node.Latitude == nil || node.Longitude == nil {
		return mapCoordinate{}, false
	}
	if !isValidCoordinate(*node.Latitude, *node.Longitude) {
		return mapCoordinate{}, false
	}

	return mapCoordinate{Latitude: *node.Latitude, Longitude: *node.Longitude}, true
}

func isValidCoordinate(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func chooseMapCenter(nodes []domain.Node, localNodeID string) (mapCoordinate, bool) {
	localNodeID = strings.TrimSpace(localNodeID)
	var all []mapCoordinate
	for _, node := range nodes {
		coord, ok := nodeCoordinate(node)
		if !ok {
			continue
		}
		all = append(all, coord)
		if localNodeID != "" && node.NodeID == localNodeID {
			return coord, true
		}
	}

	return robustClusterCenter(all)
}

func robustClusterCenter(points []mapCoordinate) (mapCoordinate, bool) {
	if len(points) == 0 {
		return mapCoordinate{}, false
	}

	center := mapCoordinate{
		Latitude:  medianFloat64(project(points, func(p mapCoordinate) float64 { return p.Latitude })),
		Longitude: medianFloat64(project(points, func(p mapCoordinate) float64 { return p.Longitude })),
	}
	if len(points) < 4 {
		return center, true
	}

	distances := make([]float64, 0, len(points))
	for _, point := range points {
		distances = append(distances, haversineKilometers(center, point))
	}
	distMedian := medianFloat64(distances)
	absDev := make([]float64, 0, len(points))
	for _, distance := range distances {
		absDev = append(absDev, math.Abs(distance-distMedian))
	}
	mad := medianFloat64(absDev)
	if mad <= 0 {
		return center, true
	}

	maxDistance := distMedian + mapOutlierThreshold*mad
	filtered := make([]mapCoordinate, 0, len(points))
	for _, point := range points {
		if haversineKilometers(center, point) <= maxDistance {
			filtered = append(filtered, point)
		}
	}
	if len(filtered) == 0 {
		return center, true
	}

	return mapCoordinate{
		Latitude:  medianFloat64(project(filtered, func(p mapCoordinate) float64 { return p.Latitude })),
		Longitude: medianFloat64(project(filtered, func(p mapCoordinate) float64 { return p.Longitude })),
	}, true
}

func centerCoordinateToViewport(center mapCoordinate, zoom int) mapViewportState {
	zoom = max(0, min(19, zoom))
	tileX, tileY := latLonToTile(center, zoom)
	offset := mapTileOffset(zoom)
	centerBias := mapCenterTileBias(zoom)
	mx := int(math.Round(tileX - centerBias))
	my := int(math.Round(tileY - centerBias))

	return mapViewportState{
		Zoom: zoom,
		X:    mx - offset,
		Y:    my - offset,
	}
}

func projectCoordinateToScreen(coord mapCoordinate, view mapViewportState, canvasSize fyne.Size) (fyne.Position, bool) {
	if canvasSize.Width <= 0 || canvasSize.Height <= 0 {
		return fyne.Position{}, false
	}

	midTileX := (int(canvasSize.Width) - mapTileSize*2) / 2
	midTileY := (int(canvasSize.Height) - mapTileSize*2) / 2
	if view.Zoom == 0 {
		midTileX += mapTileSize / 2
		midTileY += mapTileSize / 2
	}
	offset := mapTileOffset(view.Zoom)
	mx := float64(view.X + offset)
	my := float64(view.Y + offset)
	tileX, tileY := latLonToTile(coord, view.Zoom)

	screenX := float64(midTileX) + (tileX-mx)*mapTileSize
	screenY := float64(midTileY) + (tileY-my)*mapTileSize

	return fyne.NewPos(float32(screenX), float32(screenY)), true
}

func mapTileOffset(zoom int) int {
	if zoom < 0 {
		zoom = 0
	}

	count := 1 << zoom

	return int(float32(count)/2 - 0.5)
}

func mapCenterTileBias(zoom int) float64 {
	if zoom == 0 {
		return 0.5
	}

	return 1.0
}

func latLonToTile(coord mapCoordinate, zoom int) (float64, float64) {
	n := math.Pow(2, float64(zoom))
	lon := coord.Longitude
	lat := max(-mapMaxLatitudeMerc, min(mapMaxLatitudeMerc, coord.Latitude))
	latRad := lat * math.Pi / 180

	x := (lon + 180.0) / 360.0 * n
	y := (1 - math.Log(math.Tan(latRad)+1/math.Cos(latRad))/math.Pi) / 2 * n

	return x, y
}

func haversineKilometers(a, b mapCoordinate) float64 {
	const earthRadiusKm = 6371.0

	lat1 := a.Latitude * math.Pi / 180
	lat2 := b.Latitude * math.Pi / 180
	dLat := lat2 - lat1
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180

	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon

	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}

	return (sorted[mid-1] + sorted[mid]) / 2
}

func project[T any](values []T, fn func(T) float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, fn(value))
	}

	return out
}
