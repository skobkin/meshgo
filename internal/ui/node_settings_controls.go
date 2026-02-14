package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type nodeSettingsPageControls struct {
	saveButton   *widget.Button
	cancelButton *widget.Button
	reloadButton *widget.Button
	statusLabel  *widget.Label
	progressBar  *widget.ProgressBar
	root         fyne.CanvasObject
}

func newNodeSettingsPageControls(initialStatus string) *nodeSettingsPageControls {
	status := widget.NewLabel(strings.TrimSpace(initialStatus))
	status.Wrapping = fyne.TextWrapWord

	progress := widget.NewProgressBar()
	progress.SetValue(0)

	saveButton := widget.NewButton("Save", nil)
	cancelButton := widget.NewButton("Cancel", nil)
	reloadButton := widget.NewButton("Reload", nil)

	buttons := container.NewHBox(reloadButton, layout.NewSpacer(), cancelButton, saveButton)
	root := container.NewVBox(
		widget.NewSeparator(),
		progress,
		status,
		buttons,
	)

	return &nodeSettingsPageControls{
		saveButton:   saveButton,
		cancelButton: cancelButton,
		reloadButton: reloadButton,
		statusLabel:  status,
		progressBar:  progress,
		root:         root,
	}
}

func (c *nodeSettingsPageControls) SetStatus(text string, completed, total int) {
	if c == nil {
		return
	}
	if strings.TrimSpace(text) != "" {
		c.statusLabel.SetText(text)
	}
	c.progressBar.SetValue(nodeSettingsProgress(completed, total))
}

func nodeSettingsProgress(completed, total int) float64 {
	if total <= 0 {
		return 0
	}
	if completed <= 0 {
		return 0
	}
	if completed >= total {
		return 1
	}

	return float64(completed) / float64(total)
}

func wrapNodeSettingsPage(content fyne.CanvasObject, controls *nodeSettingsPageControls) fyne.CanvasObject {
	if controls == nil {
		return content
	}

	return container.NewBorder(nil, controls.root, nil, nil, container.NewVScroll(content))
}
