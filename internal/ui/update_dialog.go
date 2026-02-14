package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // register GIF decoder for image metadata probing
	_ "image/jpeg" // register JPEG decoder for image metadata probing
	_ "image/png"  // register PNG decoder for image metadata probing
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/resources"
)

const (
	markdownImageLoadTimeout       = 12 * time.Second
	maxMarkdownImageBytes          = 1 << 20
	maxMarkdownImageWidth          = 560
	maxMarkdownImageHeight         = 320
	markdownPlaceholderHeight      = 72
	markdownPlaceholderBorderWidth = 1
)

var markdownImageHTTPClient = &http.Client{Timeout: markdownImageLoadTimeout}
var lazyMarkdownLogger = slog.With("component", "ui.lazy_markdown")
var updateDialogLogger = slog.With("component", "ui.update_dialog")
var errMarkdownImageTooLarge = errors.New("image too large")
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

	changelogRichText := newLazyMarkdownRichText(buildUpdateChangelogText(snapshot.Releases))
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

func newLazyMarkdownRichText(markdown string) *widget.RichText {
	text := widget.NewRichTextFromMarkdown(markdown)
	imageCount := countMarkdownImageSegments(text.Segments)
	lazyMarkdownLogger.Info(
		"prepared lazy markdown",
		"chars", len(markdown),
		"image_segments", imageCount,
	)
	lazyMarkdownLogger.Debug(
		"rewriting markdown image segments",
		"segment_count", len(text.Segments),
		"image_segments", imageCount,
	)
	text.Segments = rewriteMarkdownImageSegments(text.Segments)

	return text
}

func countMarkdownImageSegments(segments []widget.RichTextSegment) int {
	count := 0
	for _, segment := range segments {
		switch current := segment.(type) {
		case *widget.ImageSegment:
			count++
		case *widget.ListSegment:
			count += countMarkdownImageSegments(current.Items)
		case *widget.ParagraphSegment:
			count += countMarkdownImageSegments(current.Texts)
		}
	}

	return count
}

func rewriteMarkdownImageSegments(segments []widget.RichTextSegment) []widget.RichTextSegment {
	rewritten := make([]widget.RichTextSegment, 0, len(segments))
	for _, segment := range segments {
		switch current := segment.(type) {
		case *widget.ImageSegment:
			rewritten = append(rewritten, &lazyMarkdownImageSegment{
				Source:    current.Source,
				Title:     current.Title,
				Alignment: current.Alignment,
			})
		case *widget.ListSegment:
			clone := *current
			clone.Items = rewriteMarkdownImageSegments(current.Items)
			rewritten = append(rewritten, &clone)
		case *widget.ParagraphSegment:
			clone := *current
			clone.Texts = rewriteMarkdownImageSegments(current.Texts)
			rewritten = append(rewritten, &clone)
		default:
			rewritten = append(rewritten, segment)
		}
	}

	return rewritten
}

type lazyMarkdownImageSegment struct {
	Source    fyne.URI
	Title     string
	Alignment fyne.TextAlign
}

func (s *lazyMarkdownImageSegment) Inline() bool {
	return false
}

func (s *lazyMarkdownImageSegment) Textual() string {
	return "Image " + strings.TrimSpace(s.Title)
}

func (s *lazyMarkdownImageSegment) Update(fyne.CanvasObject) {}

func (s *lazyMarkdownImageSegment) Visual() fyne.CanvasObject {
	lazyMarkdownLogger.Debug(
		"creating lazy markdown image placeholder",
		"source", markdownImageSource(s.Source),
		"title", strings.TrimSpace(s.Title),
	)
	loading := newMarkdownImagePlaceholder("Loading image...")
	root, contentIndex := alignObject(s.Alignment, loading)
	s.loadImageAsync(root, contentIndex)

	return root
}

func (s *lazyMarkdownImageSegment) Select(_, _ fyne.Position) {}

func (s *lazyMarkdownImageSegment) SelectedText() string {
	return ""
}

func (s *lazyMarkdownImageSegment) Unselect() {}

func (s *lazyMarkdownImageSegment) loadImageAsync(root *fyne.Container, contentIndex int) {
	source := s.Source
	title := strings.TrimSpace(s.Title)
	lazyMarkdownLogger.Debug(
		"starting async markdown image load",
		"source", markdownImageSource(source),
		"title", title,
	)
	go func() {
		object, err := loadMarkdownImageObject(source)
		fyne.Do(func() {
			if contentIndex >= len(root.Objects) {
				lazyMarkdownLogger.Debug(
					"skipping markdown image update: stale content index",
					"source", markdownImageSource(source),
					"content_index", contentIndex,
					"object_count", len(root.Objects),
				)

				return
			}
			if err != nil {
				lazyMarkdownLogger.Info(
					"markdown image load failed",
					"source", markdownImageSource(source),
					"title", title,
					"error", err,
				)
				text := "Image unavailable"
				if title != "" {
					text += ": " + title
				}
				root.Objects[contentIndex] = newMarkdownImagePlaceholder(text)
				root.Refresh()

				return
			}
			lazyMarkdownLogger.Info(
				"markdown image loaded",
				"source", markdownImageSource(source),
				"title", title,
			)
			root.Objects[contentIndex] = object
			root.Refresh()
		})
	}()
}

func alignObject(alignment fyne.TextAlign, object fyne.CanvasObject) (*fyne.Container, int) {
	switch alignment {
	case fyne.TextAlignLeading:
		return container.NewHBox(object, layout.NewSpacer()), 0
	case fyne.TextAlignTrailing:
		return container.NewHBox(layout.NewSpacer(), object), 1
	default:
		return container.NewHBox(layout.NewSpacer(), object, layout.NewSpacer()), 1
	}
}

func loadMarkdownImageObject(source fyne.URI) (fyne.CanvasObject, error) {
	resource, content, err := loadMarkdownImageResource(source)
	if err != nil {
		return nil, err
	}
	image := canvas.NewImageFromResource(resource)
	image.FillMode = canvas.ImageFillContain
	displaySize := markdownImageDisplaySize(content)
	image.SetMinSize(displaySize)

	return container.NewGridWrap(displaySize, image), nil
}

func loadMarkdownImageResource(source fyne.URI) (fyne.Resource, []byte, error) {
	if source == nil {
		return nil, nil, fmt.Errorf("empty image source")
	}

	content, err := readMarkdownImageBytes(source)
	if err != nil {
		return nil, nil, err
	}

	name := source.Name()
	if strings.TrimSpace(name) == "" {
		name = "release-image"
	}

	return fyne.NewStaticResource(name, content), content, nil
}

func readMarkdownImageBytes(source fyne.URI) ([]byte, error) {
	if source == nil {
		return nil, fmt.Errorf("empty image source")
	}
	scheme := strings.ToLower(strings.TrimSpace(source.Scheme()))
	if scheme == "http" || scheme == "https" {
		rawURL := source.String()
		lazyMarkdownLogger.Debug("loading remote markdown image", "source", markdownImageSource(source))
		if tooLarge, err := remoteMarkdownImageTooLarge(rawURL); err != nil {
			lazyMarkdownLogger.Debug(
				"markdown image size preflight failed; continuing with guarded download",
				"source", markdownImageSource(source),
				"error", err,
			)
		} else if tooLarge {
			lazyMarkdownLogger.Warn(
				"skipping markdown image: exceeds size limit from HEAD preflight",
				"source", markdownImageSource(source),
				"max_bytes", maxMarkdownImageBytes,
			)

			return nil, errMarkdownImageTooLarge
		}

		request, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create image request: %w", err)
		}
		response, err := markdownImageHTTPClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("request image: %w", err)
		}
		defer func() {
			_ = response.Body.Close()
		}()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request image: unexpected status %d", response.StatusCode)
		}
		if response.ContentLength > maxMarkdownImageBytes {
			lazyMarkdownLogger.Warn(
				"skipping markdown image: exceeds size limit from GET headers",
				"source", markdownImageSource(source),
				"content_length", response.ContentLength,
				"max_bytes", maxMarkdownImageBytes,
			)

			return nil, errMarkdownImageTooLarge
		}

		content, err := readLimitedBytes(response.Body, source)
		if err != nil {
			return nil, err
		}
		lazyMarkdownLogger.Debug("downloaded remote markdown image", "source", markdownImageSource(source), "bytes", len(content))

		return content, nil
	}

	lazyMarkdownLogger.Debug("loading local markdown image", "source", markdownImageSource(source))
	reader, err := storage.Reader(source)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	content, err := readLimitedBytes(reader, source)
	if err != nil {
		return nil, err
	}
	lazyMarkdownLogger.Debug("loaded local markdown image", "source", markdownImageSource(source), "bytes", len(content))

	return content, nil
}

func readLimitedBytes(reader io.Reader, source fyne.URI) ([]byte, error) {
	limited := io.LimitReader(reader, maxMarkdownImageBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(content) > maxMarkdownImageBytes {
		lazyMarkdownLogger.Warn(
			"skipping markdown image: exceeds size limit while reading body",
			"source", markdownImageSource(source),
			"max_bytes", maxMarkdownImageBytes,
		)

		return nil, errMarkdownImageTooLarge
	}

	return content, nil
}

func remoteMarkdownImageTooLarge(rawURL string) (bool, error) {
	request, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return false, fmt.Errorf("create image head request: %w", err)
	}
	response, err := markdownImageHTTPClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("head request image: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, nil
	}

	return response.ContentLength > maxMarkdownImageBytes, nil
}

func newMarkdownImagePlaceholder(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Alignment = fyne.TextAlignCenter

	background := canvas.NewRectangle(markdownPlaceholderBackgroundColor())
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = markdownPlaceholderBorderColor()
	border.StrokeWidth = markdownPlaceholderBorderWidth

	return container.NewGridWrap(
		fyne.NewSize(maxMarkdownImageWidth, markdownPlaceholderHeight),
		container.NewStack(
			background,
			border,
			container.NewPadded(container.NewCenter(label)),
		),
	)
}

func markdownPlaceholderBorderColor() color.Color {
	app := fyne.CurrentApp()
	if app == nil {
		return color.NRGBA{R: 120, G: 120, B: 120, A: 90}
	}

	palette := app.Settings().Theme()
	variant := app.Settings().ThemeVariant()
	border := toNRGBA(palette.Color(theme.ColorNameForeground, variant))
	border.A = 80

	return border
}

func markdownPlaceholderBackgroundColor() color.Color {
	app := fyne.CurrentApp()
	if app == nil {
		return color.NRGBA{R: 0, G: 0, B: 0, A: 10}
	}

	palette := app.Settings().Theme()
	variant := app.Settings().ThemeVariant()
	bg := toNRGBA(palette.Color(theme.ColorNameInputBackground, variant))
	bg.A = 55

	return bg
}

func markdownImageDisplaySize(content []byte) fyne.Size {
	defaultSize := fyne.NewSize(maxMarkdownImageWidth, maxMarkdownImageHeight)
	if len(content) == 0 {
		return defaultSize
	}

	meta, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || meta.Width <= 0 || meta.Height <= 0 {
		return defaultSize
	}

	width := float32(meta.Width)
	height := float32(meta.Height)
	widthScale := maxMarkdownImageWidth / width
	heightScale := maxMarkdownImageHeight / height
	scaleFactor := widthScale
	if heightScale < scaleFactor {
		scaleFactor = heightScale
	}
	if scaleFactor < 1 {
		width *= scaleFactor
		height *= scaleFactor
	}

	return fyne.NewSize(width, height)
}

func markdownImageSource(source fyne.URI) string {
	if source == nil {
		return ""
	}

	return strings.TrimSpace(source.String())
}
