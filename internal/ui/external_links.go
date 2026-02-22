package ui

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var externalURLLogger = slog.With("component", "ui.external_url")

func openExternalURL(rawURL string) error {
	externalURLLogger.Debug("opening external URL", "url", strings.TrimSpace(rawURL))
	parsed, err := parseExternalURL(rawURL)
	if err != nil {
		externalURLLogger.Warn("invalid external URL", "url", strings.TrimSpace(rawURL), "error", err)

		return err
	}

	currentApp := fyne.CurrentApp()
	if currentApp == nil {
		externalURLLogger.Warn("opening external URL failed: application is not initialized", "url", parsed.String())

		return fmt.Errorf("application is not initialized")
	}
	if err := currentApp.OpenURL(parsed); err != nil {
		externalURLLogger.Warn("opening external URL failed", "url", parsed.String(), "error", err)

		return fmt.Errorf("open url: %w", err)
	}
	externalURLLogger.Info("opened external URL", "url", parsed.String())

	return nil
}

func parseExternalURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if strings.TrimSpace(parsed.Scheme) == "" || strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("invalid url %q: expected absolute URL", rawURL)
	}

	return parsed, nil
}

func newSafeHyperlink(label string, rawURL string, status *widget.Label) fyne.CanvasObject {
	parsed, err := parseExternalURL(rawURL)
	if err == nil {
		return widget.NewHyperlink(label, parsed)
	}

	fallback := widget.NewButton(label, func() {
		if status == nil {
			return
		}
		status.SetText(fmt.Sprintf("%s link is unavailable: %v", label, err))
	})
	fallback.Importance = widget.LowImportance

	return fallback
}
