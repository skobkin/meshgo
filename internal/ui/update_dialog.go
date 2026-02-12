package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // register GIF decoder for image metadata probing
	_ "image/jpeg" // register JPEG decoder for image metadata probing
	_ "image/png"  // register PNG decoder for image metadata probing
	"io"
	"net/http"
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
	markdownImageLoadTimeout = 12 * time.Second
	maxMarkdownImageBytes    = 1 << 20
	maxMarkdownImageWidth    = 560
	maxMarkdownImageHeight   = 320
)

var markdownImageHTTPClient = &http.Client{Timeout: markdownImageLoadTimeout}

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

	changelogRichText := newLazyMarkdownRichText(buildUpdateChangelogText(snapshot.Releases))
	changelogRichText.Wrapping = fyne.TextWrapWord
	changelogScroll := container.NewVScroll(changelogRichText)
	changelogScroll.SetMinSize(fyne.NewSize(0, 320))

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
		changelogScroll,
		downloadButton,
	)

	updateDialog := dialog.NewCustom("Update", "Close", content, window)
	updateDialog.Resize(fyne.NewSize(760, 520))
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
		}
		sections = append(sections, fmt.Sprintf("## %s\n\n%s", version, body))
	}

	return strings.Join(sections, "\n\n---\n\n")
}

func newLazyMarkdownRichText(markdown string) *widget.RichText {
	text := widget.NewRichTextFromMarkdown(markdown)
	text.Segments = rewriteMarkdownImageSegments(text.Segments)

	return text
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
	go func() {
		object, err := loadMarkdownImageObject(source)
		fyne.Do(func() {
			if contentIndex >= len(root.Objects) {
				return
			}
			if err != nil {
				text := "Image unavailable"
				if title != "" {
					text += ": " + title
				}
				root.Objects[contentIndex] = newMarkdownImagePlaceholder(text)
				root.Refresh()

				return
			}
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
		request, err := http.NewRequest(http.MethodGet, source.String(), nil)
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

		return readLimitedBytes(response.Body)
	}

	reader, err := storage.Reader(source)
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	return readLimitedBytes(reader)
}

func readLimitedBytes(reader io.Reader) ([]byte, error) {
	limited := io.LimitReader(reader, maxMarkdownImageBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(content) > maxMarkdownImageBytes {
		return nil, fmt.Errorf("image too large")
	}

	return content, nil
}

func newMarkdownImagePlaceholder(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Alignment = fyne.TextAlignCenter

	return container.NewGridWrap(
		fyne.NewSize(maxMarkdownImageWidth, maxMarkdownImageHeight),
		container.NewCenter(label),
	)
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
