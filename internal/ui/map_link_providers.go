package ui

import (
	"fmt"
	"net/url"

	"github.com/skobkin/meshgo/internal/config"
)

const (
	openStreetMapURLTemplate = "https://www.openstreetmap.org/?mlat=%.6f&mlon=%.6f#map=%d/%.6f/%.6f"
	kagiURLTemplate          = "https://kagi.com/maps?q=%.6f,%.6f#%d/%.6f/%.6f"
	googleMapsURLTemplate    = "https://www.google.com/maps?ll=%.6f,%.6f&z=%d&q=%.6f,%.6f"
	yandexMapsURLTemplate    = "https://yandex.com/maps/?ll=%.6f%%2C%.6f&z=%d&pt=%.6f,%.6f"
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
	{
		Provider: config.MapLinkProviderKagi,
		Label:    "Kagi",
		URLFor:   nodePositionKagiURL,
	},
	{
		Provider: config.MapLinkProviderGoogle,
		Label:    "Google",
		URLFor:   nodePositionGoogleMapsURL,
	},
	{
		Provider: config.MapLinkProviderYandex,
		Label:    "Yandex",
		URLFor:   nodePositionYandexMapsURL,
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

func nodePositionKagiURL(latitude, longitude float64, precisionBits *uint32) (*url.URL, error) {
	zoom := nodePositionLinkZoomLevel(latitude, precisionBits)
	rawURL := fmt.Sprintf(kagiURLTemplate, latitude, longitude, zoom, latitude, longitude)

	return parseExternalURL(rawURL)
}

func nodePositionGoogleMapsURL(latitude, longitude float64, precisionBits *uint32) (*url.URL, error) {
	zoom := nodePositionLinkZoomLevel(latitude, precisionBits)
	rawURL := fmt.Sprintf(googleMapsURLTemplate, latitude, longitude, zoom, latitude, longitude)

	return parseExternalURL(rawURL)
}

func nodePositionYandexMapsURL(latitude, longitude float64, precisionBits *uint32) (*url.URL, error) {
	zoom := nodePositionLinkZoomLevel(latitude, precisionBits)
	rawURL := fmt.Sprintf(yandexMapsURLTemplate, longitude, latitude, zoom, longitude, latitude)

	return parseExternalURL(rawURL)
}
