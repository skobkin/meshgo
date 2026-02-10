package resources

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed tray/icon.png
var trayIcon []byte

func TrayIconResource() fyne.Resource {
	return fyne.NewStaticResource("tray-icon.png", trayIcon)
}
