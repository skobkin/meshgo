package resources

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// UIIcon identifies sidebar/status icon resources.
type UIIcon string

const (
	UIIconChats           UIIcon = "chats"
	UIIconNodes           UIIcon = "nodes"
	UIIconMap             UIIcon = "map"
	UIIconNodeSettings    UIIcon = "node_settings"
	UIIconAppSettings     UIIcon = "app_settings"
	UIIconConnected       UIIcon = "connected"
	UIIconDisconnected    UIIcon = "disconnected"
	UIIconMapNodeMarker   UIIcon = "map_node_marker"
	UIIconUpdateAvailable UIIcon = "update_available"
)

var uiDarkIconResources = map[UIIcon]fyne.Resource{
	UIIconChats:           fyne.NewStaticResource("resources/ui/dark/chats.svg", uiDarkChats),
	UIIconNodes:           fyne.NewStaticResource("resources/ui/dark/nodes.svg", uiDarkNodes),
	UIIconMap:             fyne.NewStaticResource("resources/ui/dark/map.svg", uiDarkMap),
	UIIconNodeSettings:    fyne.NewStaticResource("resources/ui/dark/node_settings.svg", uiDarkNodeSettings),
	UIIconAppSettings:     fyne.NewStaticResource("resources/ui/dark/app_settings.svg", uiDarkAppSettings),
	UIIconConnected:       fyne.NewStaticResource("resources/ui/dark/connected.svg", uiDarkConnected),
	UIIconDisconnected:    fyne.NewStaticResource("resources/ui/dark/disconnected.svg", uiDarkDisconnected),
	UIIconMapNodeMarker:   fyne.NewStaticResource("resources/ui/dark/map_node_marker.svg", uiDarkMapNodeMarker),
	UIIconUpdateAvailable: fyne.NewStaticResource("resources/ui/dark/update_available.svg", uiDarkUpdateAvailable),
}

var uiLightIconResources = map[UIIcon]fyne.Resource{
	UIIconChats:           fyne.NewStaticResource("resources/ui/light/chats.svg", uiLightChats),
	UIIconNodes:           fyne.NewStaticResource("resources/ui/light/nodes.svg", uiLightNodes),
	UIIconMap:             fyne.NewStaticResource("resources/ui/light/map.svg", uiLightMap),
	UIIconNodeSettings:    fyne.NewStaticResource("resources/ui/light/node_settings.svg", uiLightNodeSettings),
	UIIconAppSettings:     fyne.NewStaticResource("resources/ui/light/app_settings.svg", uiLightAppSettings),
	UIIconConnected:       fyne.NewStaticResource("resources/ui/light/connected.svg", uiLightConnected),
	UIIconDisconnected:    fyne.NewStaticResource("resources/ui/light/disconnected.svg", uiLightDisconnected),
	UIIconMapNodeMarker:   fyne.NewStaticResource("resources/ui/light/map_node_marker.svg", uiLightMapNodeMarker),
	UIIconUpdateAvailable: fyne.NewStaticResource("resources/ui/light/update_available.svg", uiLightUpdateAvailable),
}

func UIIconResource(icon UIIcon, variant fyne.ThemeVariant) fyne.Resource {
	if variant == theme.VariantLight {
		if res, ok := uiLightIconResources[icon]; ok {
			return res
		}
	}
	if res, ok := uiDarkIconResources[icon]; ok {
		return res
	}

	return nil
}
