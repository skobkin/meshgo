package ui

import (
	"bytes"
	"errors"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
)

func TestReadMarkdownImageBytes_RemoteRejectsOversizedImageFromHead(t *testing.T) {
	var headHits atomic.Int64
	var getHits atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			headHits.Add(1)
			w.Header().Set("Content-Length", strconv.Itoa(maxMarkdownImageBytes+1))
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			getHits.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("unexpected"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	source, err := storage.ParseURI(server.URL + "/image.png")
	if err != nil {
		t.Fatalf("parse URI: %v", err)
	}

	_, err = readMarkdownImageBytes(source)
	if !errors.Is(err, errMarkdownImageTooLarge) {
		t.Fatalf("expected oversized image error, got %v", err)
	}
	if headHits.Load() != 1 {
		t.Fatalf("expected exactly one HEAD request, got %d", headHits.Load())
	}
	if getHits.Load() != 0 {
		t.Fatalf("expected GET to be skipped, got %d requests", getHits.Load())
	}
}

func TestReadMarkdownImageBytes_RemoteRejectsOversizedImageFromGetHeadersWhenHeadUnavailable(t *testing.T) {
	var headHits atomic.Int64
	var getHits atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			headHits.Add(1)
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodGet:
			getHits.Add(1)
			w.Header().Set("Content-Length", strconv.Itoa(maxMarkdownImageBytes+1))
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	source, err := storage.ParseURI(server.URL + "/image.png")
	if err != nil {
		t.Fatalf("parse URI: %v", err)
	}

	_, err = readMarkdownImageBytes(source)
	if !errors.Is(err, errMarkdownImageTooLarge) {
		t.Fatalf("expected oversized image error, got %v", err)
	}
	if headHits.Load() != 1 {
		t.Fatalf("expected one HEAD request, got %d", headHits.Load())
	}
	if getHits.Load() != 1 {
		t.Fatalf("expected one GET request, got %d", getHits.Load())
	}
}

func TestReadMarkdownImageBytes_RemoteLoadsImageWhenWithinLimit(t *testing.T) {
	var headHits atomic.Int64
	var getHits atomic.Int64
	expected := []byte("tiny-image")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			headHits.Add(1)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			getHits.Add(1)
			w.Header().Set("Content-Length", strconv.Itoa(len(expected)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expected)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	source, err := storage.ParseURI(server.URL + "/image.png")
	if err != nil {
		t.Fatalf("parse URI: %v", err)
	}

	got, err := readMarkdownImageBytes(source)
	if err != nil {
		t.Fatalf("read markdown image bytes: %v", err)
	}
	if !bytes.Equal(got, expected) {
		t.Fatalf("unexpected image bytes")
	}
	if headHits.Load() != 1 {
		t.Fatalf("expected one HEAD request, got %d", headHits.Load())
	}
	if getHits.Load() != 1 {
		t.Fatalf("expected one GET request, got %d", getHits.Load())
	}
}

func TestNewMarkdownImagePlaceholder_UsesCompactHeight(t *testing.T) {
	placeholder := newMarkdownImagePlaceholder("Loading image...")
	size := placeholder.MinSize()
	if size.Width != float32(maxMarkdownImageWidth) {
		t.Fatalf("expected placeholder width %f, got %f", float32(maxMarkdownImageWidth), size.Width)
	}
	if size.Height != float32(markdownPlaceholderHeight) {
		t.Fatalf("expected placeholder height %f, got %f", float32(markdownPlaceholderHeight), size.Height)
	}
	if size.Height >= maxMarkdownImageHeight {
		t.Fatalf("expected placeholder to be shorter than loaded image max height, got %f", size.Height)
	}
}

func TestNewMarkdownImagePlaceholder_HasVisibleBorder(t *testing.T) {
	placeholder := newMarkdownImagePlaceholder("Image unavailable")
	if !hasVisiblePlaceholderBorder(placeholder) {
		t.Fatalf("expected placeholder border with visible stroke")
	}
}

func hasVisiblePlaceholderBorder(object fyne.CanvasObject) bool {
	switch current := object.(type) {
	case *canvas.Rectangle:
		if current.StrokeWidth <= 0 || current.StrokeColor == nil {
			return false
		}
		stroke, ok := color.NRGBAModel.Convert(current.StrokeColor).(color.NRGBA)
		if !ok {
			return false
		}

		return stroke.A > 0
	case *fyne.Container:
		for _, child := range current.Objects {
			if hasVisiblePlaceholderBorder(child) {
				return true
			}
		}
	}

	return false
}
