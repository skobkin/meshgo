package ui

import "math"

const (
	nodePositionLinkTargetRadiusPx = 90.0
	nodePositionLinkPreciseZoom    = 19
	nodePositionLinkUnknownZoom    = mapDefaultZoom
	nodePositionLinkMinZoom        = 0
	nodePositionLinkMaxZoom        = 19
)

func nodePositionLinkZoomLevel(latitude float64, precisionBits *uint32) int {
	if precisionBits == nil || *precisionBits == 0 {
		return nodePositionLinkUnknownZoom
	}
	if *precisionBits >= 32 {
		return nodePositionLinkPreciseZoom
	}

	radiusMeters, ok := precisionBitsToRadiusMeters(*precisionBits)
	if !ok || radiusMeters <= 0 {
		return nodePositionLinkUnknownZoom
	}

	lat := max(-mapMaxLatitudeMerc, min(mapMaxLatitudeMerc, latitude))
	metersPerPixel := radiusMeters / nodePositionLinkTargetRadiusPx
	zoom := math.Log2(
		(math.Cos(lat*math.Pi/180) * mapEarthCircumferenceMeters) / (float64(mapTileSize) * metersPerPixel),
	)
	if math.IsNaN(zoom) || math.IsInf(zoom, 0) {
		return nodePositionLinkUnknownZoom
	}

	return max(nodePositionLinkMinZoom, min(nodePositionLinkMaxZoom, int(math.Floor(zoom))))
}
