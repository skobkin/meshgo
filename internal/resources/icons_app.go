package resources

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var appIconResources = map[fyne.ThemeVariant]fyne.Resource{
	theme.VariantDark:  fyne.NewStaticResource("resources/ui/dark/icon_64.png", uiDarkIcon64),
	theme.VariantLight: fyne.NewStaticResource("resources/ui/light/icon_64.png", uiLightIcon64),
}

var trayIconResources = map[fyne.ThemeVariant]fyne.Resource{
	theme.VariantDark:  fyne.NewStaticResource("resources/ui/dark/icon_32.png", uiDarkIcon32),
	theme.VariantLight: fyne.NewStaticResource("resources/ui/light/icon_32.png", uiLightIcon32),
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
