package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
)

type bluetoothScannerFunc func(ctx context.Context, adapterID string) ([]DiscoveredBluetoothDevice, error)

func (f bluetoothScannerFunc) Scan(ctx context.Context, adapterID string) ([]DiscoveredBluetoothDevice, error) {
	return f(ctx, adapterID)
}

func TestSettingsTabBluetoothScanAutofillsAddress(t *testing.T) {
	cfg := config.Default()
	cfg.Connection.Connector = config.ConnectorBluetooth

	var window fyne.Window
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config: cfg,
		},
		Platform: PlatformDependencies{
			BluetoothScanner: bluetoothScannerFunc(func(_ context.Context, _ string) ([]DiscoveredBluetoothDevice, error) {
				return []DiscoveredBluetoothDevice{
					{
						Name:    "MeshNode",
						Address: "AA:BB:CC:DD:EE:FF",
						RSSI:    -64,
					},
				}, nil
			}),
		},
		UIHooks: UIHooks{
			CurrentWindow: func() fyne.Window { return window },
			RunOnUI:       func(fn func()) { fn() },
			RunAsync:      func(fn func()) { fn() },
			ShowBluetoothScanDialog: func(_ fyne.Window, devices []DiscoveredBluetoothDevice, onSelect func(DiscoveredBluetoothDevice)) {
				onSelect(devices[0])
			},
			ShowErrorDialog: func(_ error, _ fyne.Window) {},
			ShowInfoDialog:  func(_, _ string, _ fyne.Window) {},
		},
	}

	tab := newSettingsTab(dep, widget.NewLabel(""))
	window = fynetest.NewTempWindow(t, tab)

	scanButton := mustFindButtonByText(t, tab, "Scan")
	addressEntry := mustFindEntryByPlaceholder(t, tab, "AA:BB:CC:DD:EE:FF")

	fynetest.Tap(scanButton)

	if got := strings.TrimSpace(addressEntry.Text); got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("unexpected bluetooth address: %q", got)
	}

	statusLabel := mustFindLabelByPrefix(t, tab, "Selected: ")
	if got := strings.TrimSpace(statusLabel.Text); got != "Selected: AA:BB:CC:DD:EE:FF" {
		t.Fatalf("unexpected status text: %q", got)
	}
}

func TestSettingsTabBluetoothScanButtonsReEnabledAfterError(t *testing.T) {
	cfg := config.Default()
	cfg.Connection.Connector = config.ConnectorBluetooth

	started := make(chan struct{})
	release := make(chan struct{})

	var window fyne.Window
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config: cfg,
		},
		Platform: PlatformDependencies{
			BluetoothScanner: bluetoothScannerFunc(func(_ context.Context, _ string) ([]DiscoveredBluetoothDevice, error) {
				close(started)
				<-release

				return nil, errors.New("scan failed")
			}),
		},
		UIHooks: UIHooks{
			CurrentWindow:           func() fyne.Window { return window },
			RunOnUI:                 func(fn func()) { fn() },
			RunAsync:                func(fn func()) { go fn() },
			ShowErrorDialog:         func(_ error, _ fyne.Window) {},
			ShowInfoDialog:          func(_, _ string, _ fyne.Window) {},
			ShowBluetoothScanDialog: func(_ fyne.Window, _ []DiscoveredBluetoothDevice, _ func(DiscoveredBluetoothDevice)) {},
		},
	}

	tab := newSettingsTab(dep, widget.NewLabel(""))
	window = fynetest.NewTempWindow(t, tab)

	scanButton := mustFindButtonByText(t, tab, "Scan")
	openSettingsButton := mustFindButtonByText(t, tab, "Open Bluetooth Settings")

	fynetest.Tap(scanButton)

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for scan start")
	}

	if !scanButton.Disabled() {
		t.Fatalf("scan button should be disabled while scanning")
	}
	if !openSettingsButton.Disabled() {
		t.Fatalf("open settings button should be disabled while scanning")
	}

	close(release)

	waitForCondition(t, func() bool {
		return !scanButton.Disabled() && !openSettingsButton.Disabled()
	})

	statusLabel := mustFindLabelByPrefix(t, tab, "Bluetooth scan failed:")
	if got := strings.TrimSpace(statusLabel.Text); got != "Bluetooth scan failed: scan failed" {
		t.Fatalf("unexpected status text: %q", got)
	}
}

func TestSettingsTabOpenBluetoothSettingsErrorIsShown(t *testing.T) {
	cfg := config.Default()
	cfg.Connection.Connector = config.ConnectorBluetooth

	var window fyne.Window
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config: cfg,
		},
		Platform: PlatformDependencies{
			OpenBluetoothSettings: func() error {
				return errors.New("boom")
			},
		},
		UIHooks: UIHooks{
			CurrentWindow: func() fyne.Window { return window },
		},
	}

	tab := newSettingsTab(dep, widget.NewLabel(""))
	window = fynetest.NewTempWindow(t, tab)

	openSettingsButton := mustFindButtonByText(t, tab, "Open Bluetooth Settings")
	fynetest.Tap(openSettingsButton)

	statusLabel := mustFindLabelByPrefix(t, tab, "Failed to open Bluetooth settings:")
	if got := strings.TrimSpace(statusLabel.Text); got != "Failed to open Bluetooth settings: boom" {
		t.Fatalf("unexpected status text: %q", got)
	}
}

func TestSettingsTabAutostartModeDisabledWhenAutostartOff(t *testing.T) {
	tab := newSettingsTab(RuntimeDependencies{Data: DataDependencies{Config: config.Default()}}, widget.NewLabel(""))
	_ = fynetest.NewTempWindow(t, tab)

	modeSelect := mustFindSelectWithOption(t, tab, "Background tray")
	if !modeSelect.Disabled() {
		t.Fatalf("expected autostart mode select to be disabled when autostart is off")
	}
}

func TestSettingsTabAutostartWarningDoesNotBlockSave(t *testing.T) {
	cfg := config.Default()
	cfg.UI.Autostart.Enabled = true
	cfg.UI.Autostart.Mode = config.AutostartModeBackground

	var saved config.AppConfig
	dep := RuntimeDependencies{
		Data: DataDependencies{
			Config: cfg,
		},
		Actions: ActionDependencies{
			OnSave: func(next config.AppConfig) error {
				saved = next

				return &app.AutostartSyncWarning{Err: errors.New("registry denied")}
			},
		},
	}

	tab := newSettingsTab(dep, widget.NewLabel(""))
	_ = fynetest.NewTempWindow(t, tab)

	saveButton := mustFindButtonByText(t, tab, "Save")
	fynetest.Tap(saveButton)

	if !saved.UI.Autostart.Enabled {
		t.Fatalf("expected autostart to be saved as enabled")
	}
	if saved.UI.Autostart.Mode != config.AutostartModeBackground {
		t.Fatalf("expected autostart mode %q, got %q", config.AutostartModeBackground, saved.UI.Autostart.Mode)
	}

	statusLabel := mustFindLabelByPrefix(t, tab, "Saved with warning:")
	if !strings.Contains(statusLabel.Text, "autostart sync failed") {
		t.Fatalf("unexpected warning text: %q", statusLabel.Text)
	}
}

func TestNewSafeHyperlinkInvalidURLUsesFallbackButton(t *testing.T) {
	status := widget.NewLabel("")
	link := newSafeHyperlink("Source", "://not-a-url", status)
	button, ok := link.(*widget.Button)
	if !ok {
		t.Fatalf("expected fallback button for invalid URL, got %T", link)
	}
	_ = fynetest.NewTempWindow(t, button)

	fynetest.Tap(button)

	if !strings.HasPrefix(status.Text, "Source link is unavailable:") {
		t.Fatalf("unexpected fallback status text: %q", status.Text)
	}
}

func TestNewSafeHyperlinkValidURLReturnsHyperlink(t *testing.T) {
	link := newSafeHyperlink("Source", "https://example.com", widget.NewLabel(""))
	if _, ok := link.(*widget.Hyperlink); !ok {
		t.Fatalf("expected hyperlink for valid URL, got %T", link)
	}
}

func mustFindButtonByText(t *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, object := range fynetest.LaidOutObjects(root) {
		button, ok := object.(*widget.Button)
		if !ok {
			continue
		}
		if strings.TrimSpace(button.Text) == text {
			return button
		}
	}
	t.Fatalf("button %q not found", text)

	return nil
}

func mustFindEntryByPlaceholder(t *testing.T, root fyne.CanvasObject, placeholder string) *widget.Entry {
	t.Helper()
	for _, object := range fynetest.LaidOutObjects(root) {
		entry, ok := object.(*widget.Entry)
		if !ok {
			continue
		}
		if strings.TrimSpace(entry.PlaceHolder) == placeholder {
			return entry
		}
	}
	t.Fatalf("entry with placeholder %q not found", placeholder)

	return nil
}

func mustFindLabelByPrefix(t *testing.T, root fyne.CanvasObject, prefix string) *widget.Label {
	t.Helper()
	for _, object := range fynetest.LaidOutObjects(root) {
		label, ok := object.(*widget.Label)
		if !ok {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(label.Text), prefix) {
			return label
		}
	}
	t.Fatalf("label with prefix %q not found", prefix)

	return nil
}

func mustFindSelectWithOption(t *testing.T, root fyne.CanvasObject, option string) *widget.Select {
	t.Helper()
	for _, object := range fynetest.LaidOutObjects(root) {
		selectWidget, ok := object.(*widget.Select)
		if !ok {
			continue
		}
		for _, candidate := range selectWidget.Options {
			if strings.TrimSpace(candidate) == option {
				return selectWidget
			}
		}
	}
	t.Fatalf("select with option %q not found", option)

	return nil
}

func waitForCondition(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition was not met before timeout")
}
