package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/config"
)

func newSettingsTab(dep Dependencies, connStatusLabel *widget.Label) fyne.CanvasObject {
	current := dep.Config

	connectorSelect := widget.NewSelect([]string{
		"IP",
		"Bluetooth (disabled)",
		"Serial (disabled)",
	}, nil)
	connectorSelect.SetSelected("IP")

	hostEntry := widget.NewEntry()
	hostEntry.SetText(current.Connection.Host)
	hostEntry.SetPlaceHolder("IP address or hostname")

	logToFile := widget.NewCheck("Log to file", nil)
	logToFile.SetChecked(current.Logging.LogToFile)

	levelSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	levelSelect.SetSelected(strings.ToLower(current.Logging.Level))
	if levelSelect.Selected == "" {
		levelSelect.SetSelected("info")
	}

	status := widget.NewLabel("")

	connectorSelect.OnChanged = func(value string) {
		switch value {
		case "IP":
			status.SetText("")
		case "Bluetooth (disabled)", "Serial (disabled)":
			status.SetText("Selected connector is not implemented in this draft")
			connectorSelect.SetSelected("IP")
		}
	}

	saveButton := widget.NewButton("Save", func() {
		cfg := current
		cfg.Connection.Connector = config.ConnectorIP
		cfg.Connection.Host = strings.TrimSpace(hostEntry.Text)
		cfg.Logging.Level = levelSelect.Selected
		cfg.Logging.LogToFile = logToFile.Checked

		if err := dep.OnSave(cfg); err != nil {
			status.SetText("Save failed: " + err.Error())
			return
		}
		current = cfg
		status.SetText("Saved")
	})

	form := widget.NewForm(
		widget.NewFormItem("Connector", connectorSelect),
		widget.NewFormItem("IP Host", hostEntry),
		widget.NewFormItem("Log Level", levelSelect),
	)

	return container.NewVBox(
		widget.NewLabel("App Settings & Connection"),
		connStatusLabel,
		form,
		logToFile,
		saveButton,
		status,
	)
}
