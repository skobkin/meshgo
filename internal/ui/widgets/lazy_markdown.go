package widgets

import (
	"bytes"
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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	// MarkdownImageLoadTimeout is the maximum time allowed for loading a markdown image.
	MarkdownImageLoadTimeout = 12 * time.Second
	// MaxMarkdownImageBytes is the maximum allowed size for markdown images (1MB).
	MaxMarkdownImageBytes = 1 << 20
	// MaxMarkdownImageWidth is the maximum display width for markdown images.
	MaxMarkdownImageWidth = 560
	// MaxMarkdownImageHeight is the maximum display height for markdown images.
	MaxMarkdownImageHeight = 320
	// MarkdownPlaceholderHeight is the height of the image loading placeholder.
	MarkdownPlaceholderHeight = 72
	// MarkdownPlaceholderBorderWidth is the border width of the image loading placeholder.
	MarkdownPlaceholderBorderWidth = 1
)

// MarkdownImageHTTPClient is the HTTP client used for downloading markdown images.
var MarkdownImageHTTPClient = &http.Client{Timeout: MarkdownImageLoadTimeout}

// LazyMarkdownLogger is the structured logger for lazy markdown operations.
var LazyMarkdownLogger = slog.With("component", "ui.lazy_markdown")

// ErrMarkdownImageTooLarge is returned when a markdown image exceeds size limits.
var ErrMarkdownImageTooLarge = bytes.ErrTooLarge

// MarkdownImagePattern matches markdown image syntax to extract alt text and URLs.
var MarkdownImagePattern = regexp.MustCompile(`!\[([^]]*)]\(([^)]+)\)`)

// NewLazyMarkdownRichText creates a RichText widget from markdown with lazy image loading.
func NewLazyMarkdownRichText(markdown string) *widget.RichText {
	text := widget.NewRichTextFromMarkdown(markdown)
	imageCount := CountMarkdownImageSegments(text.Segments)
	LazyMarkdownLogger.Info(
		"prepared lazy markdown",
		"chars", len(markdown),
		"image_segments", imageCount,
	)
	LazyMarkdownLogger.Debug(
		"rewriting markdown image segments",
		"segment_count", len(text.Segments),
		"image_segments", imageCount,
	)
	altTexts := ExtractMarkdownImageAltTexts(markdown)
	text.Segments = RewriteMarkdownImageSegments(text.Segments, altTexts)

	return text
}

// ExtractMarkdownImageAltTexts extracts alt text for each image URL from markdown content.
func ExtractMarkdownImageAltTexts(markdown string) map[string]string {
	altTexts := make(map[string]string)
	for _, match := range MarkdownImagePattern.FindAllStringSubmatch(markdown, -1) {
		if len(match) >= 3 {
			altText := strings.TrimSpace(match[1])
			url := strings.TrimSpace(match[2])
			if url != "" && altText != "" {
				altTexts[url] = altText
			}
		}
	}

	return altTexts
}

// CountMarkdownImageSegments counts the number of image segments in rich text segments.
func CountMarkdownImageSegments(segments []widget.RichTextSegment) int {
	count := 0
	for _, segment := range segments {
		switch current := segment.(type) {
		case *widget.ImageSegment:
			count++
		case *widget.ListSegment:
			count += CountMarkdownImageSegments(current.Items)
		case *widget.ParagraphSegment:
			count += CountMarkdownImageSegments(current.Texts)
		}
	}

	return count
}

// RewriteMarkdownImageSegments replaces ImageSegment with LazyMarkdownImageSegment for lazy loading.
func RewriteMarkdownImageSegments(segments []widget.RichTextSegment, altTexts map[string]string) []widget.RichTextSegment {
	rewritten := make([]widget.RichTextSegment, 0, len(segments))
	for _, segment := range segments {
		switch current := segment.(type) {
		case *widget.ImageSegment:
			title := current.Title
			if title == "" && current.Source != nil {
				title = altTexts[current.Source.String()]
			}
			rewritten = append(rewritten, &LazyMarkdownImageSegment{
				Source:    current.Source,
				Title:     title,
				Alignment: current.Alignment,
			})
		case *widget.ListSegment:
			clone := *current
			clone.Items = RewriteMarkdownImageSegments(current.Items, altTexts)
			rewritten = append(rewritten, &clone)
		case *widget.ParagraphSegment:
			clone := *current
			clone.Texts = RewriteMarkdownImageSegments(current.Texts, altTexts)
			rewritten = append(rewritten, &clone)
		default:
			rewritten = append(rewritten, segment)
		}
	}

	return rewritten
}

// LazyMarkdownImageSegment is a rich text segment that loads images asynchronously.
type LazyMarkdownImageSegment struct {
	Source    fyne.URI
	Title     string
	Alignment fyne.TextAlign
}

func (s *LazyMarkdownImageSegment) Inline() bool {
	return false
}

func (s *LazyMarkdownImageSegment) Textual() string {
	return "Image " + strings.TrimSpace(s.Title)
}

func (s *LazyMarkdownImageSegment) Update(fyne.CanvasObject) {}

func (s *LazyMarkdownImageSegment) Visual() fyne.CanvasObject {
	title := strings.TrimSpace(s.Title)
	LazyMarkdownLogger.Debug(
		"creating lazy markdown image placeholder",
		"source", MarkdownImageSource(s.Source),
		"title", title,
	)
	loading := NewMarkdownImagePlaceholder(title, "Loading image...")
	root, contentIndex := AlignObject(s.Alignment, loading)
	s.loadImageAsync(root, contentIndex)

	return root
}

func (s *LazyMarkdownImageSegment) Select(_, _ fyne.Position) {}

func (s *LazyMarkdownImageSegment) SelectedText() string {
	return ""
}

func (s *LazyMarkdownImageSegment) Unselect() {}

func (s *LazyMarkdownImageSegment) loadImageAsync(root *fyne.Container, contentIndex int) {
	source := s.Source
	title := strings.TrimSpace(s.Title)
	LazyMarkdownLogger.Debug(
		"starting async markdown image load",
		"source", MarkdownImageSource(source),
		"title", title,
	)
	go func() {
		object, err := LoadMarkdownImageObject(source)
		fyne.Do(func() {
			if contentIndex >= len(root.Objects) {
				LazyMarkdownLogger.Debug(
					"skipping markdown image update: stale content index",
					"source", MarkdownImageSource(source),
					"content_index", contentIndex,
					"object_count", len(root.Objects),
				)

				return
			}
			if err != nil {
				LazyMarkdownLogger.Info(
					"markdown image load failed",
					"source", MarkdownImageSource(source),
					"title", title,
					"error", err,
				)
				root.Objects[contentIndex] = NewMarkdownImagePlaceholder(title, "Image unavailable")
				root.Refresh()

				return
			}
			LazyMarkdownLogger.Info(
				"markdown image loaded",
				"source", MarkdownImageSource(source),
				"title", title,
			)
			root.Objects[contentIndex] = object
			root.Refresh()
		})
	}()
}

// AlignObject aligns an object according to the specified text alignment and returns the container and content index.
func AlignObject(alignment fyne.TextAlign, object fyne.CanvasObject) (*fyne.Container, int) {
	switch alignment {
	case fyne.TextAlignLeading:
		return container.NewHBox(object, layout.NewSpacer()), 0
	case fyne.TextAlignTrailing:
		return container.NewHBox(layout.NewSpacer(), object), 1
	default:
		return container.NewHBox(layout.NewSpacer(), object, layout.NewSpacer()), 1
	}
}

// LoadMarkdownImageObject loads an image from the source URI and returns it as a canvas object.
func LoadMarkdownImageObject(source fyne.URI) (fyne.CanvasObject, error) {
	resource, content, err := LoadMarkdownImageResource(source)
	if err != nil {
		return nil, err
	}
	img := canvas.NewImageFromResource(resource)
	img.FillMode = canvas.ImageFillContain
	displaySize := MarkdownImageDisplaySize(content)
	img.SetMinSize(displaySize)

	return container.NewGridWrap(displaySize, img), nil
}

// LoadMarkdownImageResource loads image bytes from the source URI and returns them as a fyne resource.
func LoadMarkdownImageResource(source fyne.URI) (fyne.Resource, []byte, error) {
	if source == nil {
		return nil, nil, bytes.ErrTooLarge
	}

	content, err := ReadMarkdownImageBytes(source)
	if err != nil {
		return nil, nil, err
	}

	name := source.Name()
	if strings.TrimSpace(name) == "" {
		name = "release-image"
	}

	return fyne.NewStaticResource(name, content), content, nil
}

// ReadMarkdownImageBytes reads image bytes from the source URI, supporting both local and remote sources.
func ReadMarkdownImageBytes(source fyne.URI) ([]byte, error) {
	if source == nil {
		return nil, bytes.ErrTooLarge
	}
	scheme := strings.ToLower(strings.TrimSpace(source.Scheme()))
	if scheme == "http" || scheme == "https" {
		rawURL := source.String()
		LazyMarkdownLogger.Debug("loading remote markdown image", "source", MarkdownImageSource(source))
		if tooLarge, err := RemoteMarkdownImageTooLarge(rawURL); err != nil {
			LazyMarkdownLogger.Debug(
				"markdown image size preflight failed; continuing with guarded download",
				"source", MarkdownImageSource(source),
				"error", err,
			)
		} else if tooLarge {
			LazyMarkdownLogger.Warn(
				"skipping markdown image: exceeds size limit from HEAD preflight",
				"source", MarkdownImageSource(source),
				"max_bytes", MaxMarkdownImageBytes,
			)

			return nil, ErrMarkdownImageTooLarge
		}

		request, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		// #nosec G704 -- remote markdown images are an intentional feature; only http/https URIs are allowed.
		response, err := MarkdownImageHTTPClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = response.Body.Close()
		}()
		if response.StatusCode != http.StatusOK {
			return nil, bytes.ErrTooLarge
		}
		if response.ContentLength > MaxMarkdownImageBytes {
			LazyMarkdownLogger.Warn(
				"skipping markdown image: exceeds size limit from GET headers",
				"source", MarkdownImageSource(source),
				"content_length", response.ContentLength,
				"max_bytes", MaxMarkdownImageBytes,
			)

			return nil, ErrMarkdownImageTooLarge
		}

		content, err := ReadLimitedBytes(response.Body, source)
		if err != nil {
			return nil, err
		}
		LazyMarkdownLogger.Debug("downloaded remote markdown image", "source", MarkdownImageSource(source), "bytes", len(content))

		return content, nil
	}

	LazyMarkdownLogger.Debug("loading local markdown image", "source", MarkdownImageSource(source))
	reader, err := storage.Reader(source)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = reader.Close()
	}()

	content, err := ReadLimitedBytes(reader, source)
	if err != nil {
		return nil, err
	}
	LazyMarkdownLogger.Debug("loaded local markdown image", "source", MarkdownImageSource(source), "bytes", len(content))

	return content, nil
}

// ReadLimitedBytes reads from a reader up to MaxMarkdownImageBytes to prevent excessive memory usage.
func ReadLimitedBytes(reader io.Reader, source fyne.URI) ([]byte, error) {
	limited := io.LimitReader(reader, MaxMarkdownImageBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(content) > MaxMarkdownImageBytes {
		LazyMarkdownLogger.Warn(
			"skipping markdown image: exceeds size limit while reading body",
			"source", MarkdownImageSource(source),
			"max_bytes", MaxMarkdownImageBytes,
		)

		return nil, ErrMarkdownImageTooLarge
	}

	return content, nil
}

// RemoteMarkdownImageTooLarge checks if a remote image exceeds size limits via HTTP HEAD request.
func RemoteMarkdownImageTooLarge(rawURL string) (bool, error) {
	request, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return false, err
	}
	// #nosec G704 -- remote markdown images are an intentional feature; only http/https URIs are allowed.
	response, err := MarkdownImageHTTPClient.Do(request)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return false, nil
	}

	return response.ContentLength > MaxMarkdownImageBytes, nil
}

// NewMarkdownImagePlaceholder creates a placeholder widget shown while markdown images are loading.
func NewMarkdownImagePlaceholder(title, status string) fyne.CanvasObject {
	titleLabel := widget.NewLabel(strings.TrimSpace(title))
	titleLabel.Alignment = fyne.TextAlignCenter
	titleLabel.Importance = widget.MediumImportance

	statusLabel := widget.NewLabel(strings.TrimSpace(status))
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Importance = widget.LowImportance

	background := canvas.NewRectangle(markdownPlaceholderBackgroundColor())
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = markdownPlaceholderBorderColor()
	border.StrokeWidth = MarkdownPlaceholderBorderWidth

	return container.NewGridWrap(
		fyne.NewSize(MaxMarkdownImageWidth, MarkdownPlaceholderHeight),
		container.NewStack(
			background,
			border,
			container.NewPadded(container.NewVBox(
				container.NewCenter(titleLabel),
				container.NewCenter(statusLabel),
			)),
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

func toNRGBA(c color.Color) color.NRGBA {
	r, g, b, a := c.RGBA()
	//nolint:gosec // RGBA() returns 16-bit values [0,65535]; shifting right by 8 safely converts to 8-bit [0,255].
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

// MarkdownImageDisplaySize calculates the display size for a markdown image based on its content.
func MarkdownImageDisplaySize(content []byte) fyne.Size {
	defaultSize := fyne.NewSize(MaxMarkdownImageWidth, MaxMarkdownImageHeight)
	if len(content) == 0 {
		return defaultSize
	}

	meta, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || meta.Width <= 0 || meta.Height <= 0 {
		return defaultSize
	}

	width := float32(meta.Width)
	height := float32(meta.Height)
	widthScale := MaxMarkdownImageWidth / width
	heightScale := MaxMarkdownImageHeight / height
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

// MarkdownImageSource returns the string representation of the image source URI.
func MarkdownImageSource(source fyne.URI) string {
	if source == nil {
		return ""
	}

	return strings.TrimSpace(source.String())
}
