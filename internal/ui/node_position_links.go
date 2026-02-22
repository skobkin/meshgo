package ui

import (
	"fmt"
	"math"
	"net/url"

	"github.com/skobkin/meshgo/internal/config"
)

const (
	openStreetMapURLTemplate       = "https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=%d/%.6f/%.6f"
	nodePositionLinkTargetRadiusPx = 90.0
	nodePositionLinkPreciseZoom    = 19
	nodePositionLinkUnknownZoom    = mapDefaultZoom
	nodePositionLinkMinZoom        = 0
	nodePositionLinkMaxZoom        = 19
)

type mapLinkProviderOption struct {
	Provider config.MapLinkProvider
	Label    string
	URLFor   func(latitude, longitude float64, precisionBits *uint32) (*url.URL, error)
}

var mapLinkProviderOptions = []mapLinkProviderOption{
	{
		Provider: config.MapLinkProviderOpenStreetMap,
		Label:    "OpenStreetMap",
		URLFor:   nodePositionOpenStreetMapURL,
	},
}

func mapLinkProviderLabels() []string {
	labels := make([]string, 0, len(mapLinkProviderOptions))
	for _, option := range mapLinkProviderOptions {
		labels = append(labels, option.Label)
	}

	return labels
}

func mapLinkProviderLabel(provider config.MapLinkProvider) string {
	provider = normalizeMapLinkProvider(provider)
	for _, option := range mapLinkProviderOptions {
		if option.Provider == provider {
			return option.Label
		}
	}

	return mapLinkProviderOptions[0].Label
}

func parseMapLinkProviderLabel(label string) config.MapLinkProvider {
	for _, option := range mapLinkProviderOptions {
		if option.Label == label {
			return option.Provider
		}
	}

	return mapLinkProviderOptions[0].Provider
}

func normalizeMapLinkProvider(provider config.MapLinkProvider) config.MapLinkProvider {
	for _, option := range mapLinkProviderOptions {
		if option.Provider == provider {
			return provider
		}
	}

	return mapLinkProviderOptions[0].Provider
}

func nodePositionMapURL(
	provider config.MapLinkProvider,
	latitude, longitude float64,
	precisionBits *uint32,
) (*url.URL, error) {
	provider = normalizeMapLinkProvider(provider)
	for _, option := range mapLinkProviderOptions {
		if option.Provider == provider {
			return option.URLFor(latitude, longitude, precisionBits)
		}
	}

	return mapLinkProviderOptions[0].URLFor(latitude, longitude, precisionBits)
}

func nodePositionOpenStreetMapURL(latitude, longitude float64, precisionBits *uint32) (*url.URL, error) {
	zoom := nodePositionLinkZoomLevel(latitude, precisionBits)
	rawURL := fmt.Sprintf(openStreetMapURLTemplate, latitude, longitude, zoom, latitude, longitude)

	return parseExternalURL(rawURL)
}

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
