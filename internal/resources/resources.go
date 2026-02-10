package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed logo/logo_text.png
var logoText []byte

type UIIcon string

const (
	UIIconChats        UIIcon = "chats"
	UIIconNodes        UIIcon = "nodes"
	UIIconMap          UIIcon = "map"
	UIIconNodeSettings UIIcon = "node_settings"
	UIIconAppSettings  UIIcon = "app_settings"
	UIIconConnected    UIIcon = "connected"
	UIIconDisconnected UIIcon = "disconnected"
)

//go:embed ui/dark/chats.svg
var uiDarkChats []byte

//go:embed ui/dark/nodes.svg
var uiDarkNodes []byte

//go:embed ui/dark/map.svg
var uiDarkMap []byte

//go:embed ui/dark/node_settings.svg
var uiDarkNodeSettings []byte

//go:embed ui/dark/app_settings.svg
var uiDarkAppSettings []byte

//go:embed ui/dark/connected.svg
var uiDarkConnected []byte

//go:embed ui/dark/disconnected.svg
var uiDarkDisconnected []byte

//go:embed ui/light/chats.svg
var uiLightChats []byte

//go:embed ui/light/nodes.svg
var uiLightNodes []byte

//go:embed ui/light/map.svg
var uiLightMap []byte

//go:embed ui/light/node_settings.svg
var uiLightNodeSettings []byte

//go:embed ui/light/app_settings.svg
var uiLightAppSettings []byte

//go:embed ui/light/connected.svg
var uiLightConnected []byte

//go:embed ui/light/disconnected.svg
var uiLightDisconnected []byte

//go:embed ui/dark/icon_32.png
var uiDarkIcon32 []byte

//go:embed ui/dark/icon_64.png
var uiDarkIcon64 []byte

//go:embed ui/light/icon_32.png
var uiLightIcon32 []byte

//go:embed ui/light/icon_64.png
var uiLightIcon64 []byte

var uiDarkIconResources = map[UIIcon]fyne.Resource{
	UIIconChats:        fyne.NewStaticResource("resources/ui/dark/chats.svg", uiDarkChats),
	UIIconNodes:        fyne.NewStaticResource("resources/ui/dark/nodes.svg", uiDarkNodes),
	UIIconMap:          fyne.NewStaticResource("resources/ui/dark/map.svg", uiDarkMap),
	UIIconNodeSettings: fyne.NewStaticResource("resources/ui/dark/node_settings.svg", uiDarkNodeSettings),
	UIIconAppSettings:  fyne.NewStaticResource("resources/ui/dark/app_settings.svg", uiDarkAppSettings),
	UIIconConnected:    fyne.NewStaticResource("resources/ui/dark/connected.svg", uiDarkConnected),
	UIIconDisconnected: fyne.NewStaticResource("resources/ui/dark/disconnected.svg", uiDarkDisconnected),
}

var uiLightIconResources = map[UIIcon]fyne.Resource{
	UIIconChats:        fyne.NewStaticResource("resources/ui/light/chats.svg", uiLightChats),
	UIIconNodes:        fyne.NewStaticResource("resources/ui/light/nodes.svg", uiLightNodes),
	UIIconMap:          fyne.NewStaticResource("resources/ui/light/map.svg", uiLightMap),
	UIIconNodeSettings: fyne.NewStaticResource("resources/ui/light/node_settings.svg", uiLightNodeSettings),
	UIIconAppSettings:  fyne.NewStaticResource("resources/ui/light/app_settings.svg", uiLightAppSettings),
	UIIconConnected:    fyne.NewStaticResource("resources/ui/light/connected.svg", uiLightConnected),
	UIIconDisconnected: fyne.NewStaticResource("resources/ui/light/disconnected.svg", uiLightDisconnected),
}

var appIconResources = map[fyne.ThemeVariant]fyne.Resource{
	theme.VariantDark:  fyne.NewStaticResource("resources/ui/dark/icon_64.png", uiDarkIcon64),
	theme.VariantLight: fyne.NewStaticResource("resources/ui/light/icon_64.png", uiLightIcon64),
}

var trayIconResources = map[fyne.ThemeVariant]fyne.Resource{
	theme.VariantDark:  fyne.NewStaticResource("resources/ui/dark/icon_32.png", uiDarkIcon32),
	theme.VariantLight: fyne.NewStaticResource("resources/ui/light/icon_32.png", uiLightIcon32),
}

func LogoTextResource() fyne.Resource {
	return fyne.NewStaticResource("logo-text.png", logoText)
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

func AppIconResource(variant fyne.ThemeVariant) fyne.Resource {
	if res, ok := appIconResources[variant]; ok {
		return res
	}
	return appIconResources[theme.VariantDark]
}

func TrayIconResource(variant fyne.ThemeVariant) fyne.Resource {
	if res, ok := trayIconResources[variant]; ok {
		return res
	}
	return trayIconResources[theme.VariantDark]
}
