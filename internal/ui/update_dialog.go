package ui

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/resources"
	"github.com/skobkin/meshgo/internal/ui/widgets"
)

var updateDialogLogger = slog.With("component", "ui.update_dialog")
var markdownListLeadingCommitHashPattern = regexp.MustCompile(`^(\s*(?:[*+-]|\d+\.)\s+)[0-9a-f]{40}(\s+.*)$`)

func showUpdateDialog(
	window fyne.Window,
	variant fyne.ThemeVariant,
	snapshot meshapp.UpdateSnapshot,
	openURL func(string) error,
) {
	if window == nil {
		updateDialogLogger.Warn("skipping update dialog: window is nil")

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
	dialogTitle := widget.NewLabelWithStyle("Update", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var updateDialog *widget.PopUp
	dialogCloseButton := widget.NewButton("X", func() {
		updateDialogLogger.Debug("closing update dialog")
		if updateDialog == nil {
			return
		}
		updateDialog.Hide()
	})
	dialogHeader := container.NewHBox(dialogTitle, layout.NewSpacer(), dialogCloseButton)

	updateIcon := widget.NewIcon(resources.UIIconResource(resources.UIIconUpdateAvailable, variant))
	versionsHeader := container.NewGridWithColumns(
		3,
		container.NewCenter(currentLabel),
		container.NewCenter(container.NewGridWrap(fyne.NewSquareSize(36), updateIcon)),
		container.NewCenter(latestLabel),
	)

	changelogRichText := widgets.NewLazyMarkdownRichText(buildUpdateChangelogText(snapshot.Releases))
	changelogRichText.Wrapping = fyne.TextWrapWord
	changelogScroll := container.NewVScroll(changelogRichText)
	changelogScroll.SetMinSize(fyne.NewSize(0, 320))

	downloadURL := strings.TrimSpace(snapshot.Latest.HTMLURL)
	downloadButton := widget.NewButton("Download", func() {
		updateDialogLogger.Info(
			"download button clicked",
			"url", downloadURL,
			"latest_version", strings.TrimSpace(snapshot.Latest.Version),
		)
		if openURL == nil {
			updateDialogLogger.Warn("download action skipped: openURL callback is nil")

			return
		}
		if err := openURL(downloadURL); err != nil {
			updateDialogLogger.Warn("download action failed", "url", downloadURL, "error", err)
			dialog.ShowError(err, window)

			return
		}
		updateDialogLogger.Info("download URL opened", "url", downloadURL)
	})
	downloadButton.Importance = widget.HighImportance
	if downloadURL == "" {
		downloadButton.Disable()
		updateDialogLogger.Debug("download button disabled: release URL is empty")
	}

	content := container.NewBorder(
		container.NewVBox(
			dialogHeader,
			versionsHeader,
		),
		downloadButton,
		nil,
		nil,
		changelogScroll,
	)

	updateDialog = widget.NewModalPopUp(content, window.Canvas())
	updateDialog.Resize(fyne.NewSize(760, 520))
	updateDialogLogger.Info(
		"showing update dialog",
		"current_version", currentVersion,
		"latest_version", latestVersion,
		"release_count", len(snapshot.Releases),
	)
	updateDialog.Show()
}

func newUpdateVersionText(version string, variant fyne.ThemeVariant) *canvas.Text {
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
		} else {
			body = stripLeadingCommitHashesFromMarkdown(body)
		}
		sections = append(sections, fmt.Sprintf("## %s\n\n%s", version, body))
	}

	return strings.Join(sections, "\n\n---\n\n")
}

func stripLeadingCommitHashesFromMarkdown(markdown string) string {
	if markdown == "" {
		return markdown
	}

	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		lines[i] = stripLeadingCommitHashFromLine(line)
	}

	return strings.Join(lines, "\n")
}

func stripLeadingCommitHashFromLine(line string) string {
	matches := markdownListLeadingCommitHashPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return line
	}

	return matches[1] + strings.TrimLeft(matches[2], " ")
}
