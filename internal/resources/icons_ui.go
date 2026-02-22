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
	UIIconCloudUpload     UIIcon = "cloud_upload"
	UIIconCloudDownload   UIIcon = "cloud_download"
	UIIconLockGreen       UIIcon = "lock_green"
	UIIconLockYellow      UIIcon = "lock_yellow"
	UIIconLockRed         UIIcon = "lock_red"
	UIIconLockRedWarning  UIIcon = "lock_red_warning"
	UIIconSpeakerMute     UIIcon = "speaker_mute"
	UIIconBadgePrimary    UIIcon = "badge_primary"
	UIIconRefresh         UIIcon = "refresh"
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
	UIIconCloudUpload:     fyne.NewStaticResource("resources/ui/dark/cloud_upload.svg", uiDarkCloudUpload),
	UIIconCloudDownload:   fyne.NewStaticResource("resources/ui/dark/cloud_download.svg", uiDarkCloudDownload),
	UIIconLockGreen:       fyne.NewStaticResource("resources/ui/dark/lock_green.svg", uiDarkLockGreen),
	UIIconLockYellow:      fyne.NewStaticResource("resources/ui/dark/lock_yellow.svg", uiDarkLockYellow),
	UIIconLockRed:         fyne.NewStaticResource("resources/ui/dark/lock_red.svg", uiDarkLockRed),
	UIIconLockRedWarning:  fyne.NewStaticResource("resources/ui/dark/lock_red_warning.svg", uiDarkLockRedWarning),
	UIIconSpeakerMute:     fyne.NewStaticResource("resources/ui/dark/speaker_mute.svg", uiDarkSpeakerMute),
	UIIconBadgePrimary:    fyne.NewStaticResource("resources/ui/dark/badge_primary.svg", uiDarkBadgePrimary),
	UIIconRefresh:         fyne.NewStaticResource("resources/ui/dark/refresh.svg", uiDarkRefresh),
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
	UIIconCloudUpload:     fyne.NewStaticResource("resources/ui/light/cloud_upload.svg", uiLightCloudUpload),
	UIIconCloudDownload:   fyne.NewStaticResource("resources/ui/light/cloud_download.svg", uiLightCloudDownload),
	UIIconLockGreen:       fyne.NewStaticResource("resources/ui/light/lock_green.svg", uiLightLockGreen),
	UIIconLockYellow:      fyne.NewStaticResource("resources/ui/light/lock_yellow.svg", uiLightLockYellow),
	UIIconLockRed:         fyne.NewStaticResource("resources/ui/light/lock_red.svg", uiLightLockRed),
	UIIconLockRedWarning:  fyne.NewStaticResource("resources/ui/light/lock_red_warning.svg", uiLightLockRedWarning),
	UIIconSpeakerMute:     fyne.NewStaticResource("resources/ui/light/speaker_mute.svg", uiLightSpeakerMute),
	UIIconBadgePrimary:    fyne.NewStaticResource("resources/ui/light/badge_primary.svg", uiLightBadgePrimary),
	UIIconRefresh:         fyne.NewStaticResource("resources/ui/light/refresh.svg", uiLightRefresh),
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
