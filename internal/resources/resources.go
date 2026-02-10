package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed tray/icon.png
var trayIcon []byte

//go:embed logo/logo_text.png
var logoText []byte

func TrayIconResource() fyne.Resource {
	return fyne.NewStaticResource("tray-icon.png", trayIcon)
}

func LogoTextResource() fyne.Resource {
	return fyne.NewStaticResource("logo-text.png", logoText)
}
