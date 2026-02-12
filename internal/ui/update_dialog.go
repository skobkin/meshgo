package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/resources"
)

func showUpdateDialog(
	window fyne.Window,
	variant fyne.ThemeVariant,
	snapshot meshapp.UpdateSnapshot,
	openURL func(string) error,
) {
	if window == nil {
		return
	}

	currentVersion := strings.TrimSpace(snapshot.CurrentVersion)
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	latestVersion := strings.TrimSpace(snapshot.Latest.Version)
	if latestVersion == "" {
		latestVersion = "unknown"
	}

	currentLabel := newUpdateVersionText(currentVersion, variant)
	latestLabel := newUpdateVersionText(latestVersion, variant)

	updateIcon := widget.NewIcon(resources.UIIconResource(resources.UIIconUpdateAvailable, variant))
	header := container.NewGridWithColumns(
		3,
		container.NewCenter(currentLabel),
		container.NewCenter(container.NewGridWrap(fyne.NewSquareSize(36), updateIcon)),
		container.NewCenter(latestLabel),
	)

	changelog := widget.NewMultiLineEntry()
	changelog.SetMinRowsVisible(12)
	changelog.Wrapping = fyne.TextWrapWord
	changelog.SetText(buildUpdateChangelogText(snapshot.Releases))
	changelog.Disable()

	downloadURL := strings.TrimSpace(snapshot.Latest.HTMLURL)
	downloadButton := widget.NewButton("Download", func() {
		if openURL == nil {
			return
		}
		if err := openURL(downloadURL); err != nil {
			dialog.ShowError(err, window)
		}
	})
	downloadButton.Importance = widget.HighImportance
	if downloadURL == "" {
		downloadButton.Disable()
	}

	content := container.NewVBox(
		header,
		changelog,
		downloadButton,
	)

	updateDialog := dialog.NewCustom("Update", "Close", content, window)
	updateDialog.Resize(fyne.NewSize(760, 520))
	updateDialog.Show()
}

func newUpdateVersionText(version string, variant fyne.ThemeVariaTestUIIconResourceUpdateAvailablent) *canvas.Text {
	label := canvas.NewText(version, theme.DefaultTheme().Color(theme.ColorNameForeground, variant))
	label.TextSize = 28
	label.TextStyle = fyne.TextStyle{Bold: true}

	return label
}

func buildUpdateChangelogText(releases []meshapp.ReleaseInfo) string {
	if len(releases) == 0 {
		return "No release notes available."
	}

	sections := make([]string, 0, len(releases))
	for _, release := range releases {
		version := strings.TrimSpace(release.Version)
		if version == "" {
			version = "unknown"
		}
		body := strings.TrimSpace(release.Body)
		if body == "" {
			body = "No changelog provided."
		}
		sections = append(sections, fmt.Sprintf("%s\n\n%s", version, body))
	}

	return strings.Join(sections, "\n\n--------------------\n\n")
}
