package ui

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/resources"
)

const (
	connectorOptionIP        = "IP"
	connectorOptionSerial    = "Serial"
	connectorOptionBluetooth = "Bluetooth LE (unstable)"
	autostartOptionNormal    = "Normal window"
	autostartOptionTray      = "Background tray"
)

var defaultSerialBaudOptions = []string{"9600", "19200", "38400", "57600", "115200", "230400", "460800", "921600"}

func newSettingsTab(dep Dependencies, connStatusLabel *widget.Label) fyne.CanvasObject {
	current := dep.Data.Config
	current.ApplyDefaults()

	connectorSelect := widget.NewSelect([]string{
		connectorOptionIP,
		connectorOptionSerial,
		connectorOptionBluetooth,
	}, nil)
	connectorSelect.SetSelected(connectorOptionFromType(current.Connection.Connector))

	hostEntry := widget.NewEntry()
	hostEntry.SetText(current.Connection.Host)
	hostEntry.SetPlaceHolder("IP address or hostname")

	logToFile := widget.NewCheck("", nil)
	logToFile.SetChecked(current.Logging.LogToFile)

	levelSelect := widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	levelSelect.SetSelected(strings.ToLower(current.Logging.Level))
	if levelSelect.Selected == "" {
		levelSelect.SetSelected("info")
	}

	autostartEnabled := widget.NewCheck("", nil)
	autostartEnabled.SetChecked(current.UI.Autostart.Enabled)

	autostartModeSelect := widget.NewSelect([]string{autostartOptionNormal, autostartOptionTray}, nil)
	autostartModeSelect.SetSelected(autostartOptionFromMode(current.UI.Autostart.Mode))
	if autostartModeSelect.Selected == "" {
		autostartModeSelect.SetSelected(autostartOptionNormal)
	}
	setAutostartModeEnabled := func(enabled bool) {
		if enabled {
			autostartModeSelect.Enable()
			return
		}
		autostartModeSelect.Disable()
	}
	autostartEnabled.OnChanged = func(value bool) {
		setAutostartModeEnabled(value)
	}
	setAutostartModeEnabled(autostartEnabled.Checked)

	status := widget.NewLabel("")

	serialPortSelect := widget.NewSelect(nil, nil)
	serialPortSelect.PlaceHolder = "Select serial port"
	serialPortSelect.SetSelected(current.Connection.SerialPort)

	serialBaudSelect := widget.NewSelect(uniqueValues(append(defaultSerialBaudOptions, strconv.Itoa(current.Connection.SerialBaud))), nil)
	serialBaudSelect.SetSelected(strconv.Itoa(current.Connection.SerialBaud))

	bluetoothAddressEntry := widget.NewEntry()
	bluetoothAddressEntry.SetText(current.Connection.BluetoothAddress)
	bluetoothAddressEntry.SetPlaceHolder("AA:BB:CC:DD:EE:FF")

	bluetoothAdapterEntry := widget.NewEntry()
	bluetoothAdapterEntry.SetText(current.Connection.BluetoothAdapter)
	bluetoothAdapterEntry.SetPlaceHolder("hci0 (optional)")

	bluetoothPairingHint := widget.NewLabel("Pair the node in OS Bluetooth settings before connecting.")
	bluetoothPairingHint.Wrapping = fyne.TextWrapWord

	bluetoothScanner := dep.Platform.BluetoothScanner
	if bluetoothScanner == nil {
		bluetoothScanner = NewTinyGoBluetoothScanner(defaultBluetoothScanDuration)
	}
	openBluetoothSettingsFn := dep.Platform.OpenBluetoothSettings
	if openBluetoothSettingsFn == nil {
		openBluetoothSettingsFn = func() error {
			return fmt.Errorf("open bluetooth settings is not configured")
		}
	}
	currentWindowFn := dep.UIHooks.CurrentWindow
	if currentWindowFn == nil {
		currentWindowFn = currentWindow
	}
	runOnUI := dep.UIHooks.RunOnUI
	if runOnUI == nil {
		runOnUI = fyne.Do
	}
	runAsync := dep.UIHooks.RunAsync
	if runAsync == nil {
		runAsync = func(fn func()) {
			go fn()
		}
	}
	showScanDialogFn := dep.UIHooks.ShowBluetoothScanDialog
	if showScanDialogFn == nil {
		showScanDialogFn = showBluetoothScanDialog
	}
	showErrorDialogFn := dep.UIHooks.ShowErrorDialog
	if showErrorDialogFn == nil {
		showErrorDialogFn = dialog.ShowError
	}
	showInfoDialogFn := dep.UIHooks.ShowInfoDialog
	if showInfoDialogFn == nil {
		showInfoDialogFn = dialog.ShowInformation
	}

	scanBluetoothButton := widget.NewButton("Scan", nil)
	openBluetoothSettingsButton := widget.NewButton("Open Bluetooth Settings", func() {
		if err := openBluetoothSettingsFn(); err != nil {
			status.SetText("Failed to open Bluetooth settings: " + err.Error())
			return
		}
		status.SetText("")
	})
	bluetoothActionRow := container.NewHBox(scanBluetoothButton, openBluetoothSettingsButton)

	scanBluetoothButton.OnTapped = func() {
		window := currentWindowFn()
		if window == nil {
			status.SetText("Bluetooth scan failed: active window is unavailable")
			return
		}

		scanBluetoothButton.Disable()
		openBluetoothSettingsButton.Disable()
		status.SetText("Scanning...")
		progressBar := widget.NewProgressBarInfinite()
		progressBar.Start()
		progress := dialog.NewCustomWithoutButtons(
			"Bluetooth scan",
			container.NewVBox(
				widget.NewLabel("Scanning for nearby devices..."),
				progressBar,
			),
			window,
		)
		progress.Show()

		adapterID := strings.TrimSpace(bluetoothAdapterEntry.Text)
		runAsync(func() {
			devices, err := bluetoothScanner.Scan(context.Background(), adapterID)
			runOnUI(func() {
				progressBar.Stop()
				progress.Hide()
				scanBluetoothButton.Enable()
				openBluetoothSettingsButton.Enable()

				if err != nil {
					status.SetText("Bluetooth scan failed: " + err.Error())
					showErrorDialogFn(err, window)
					return
				}
				if len(devices) == 0 {
					status.SetText("No Bluetooth devices found")
					showInfoDialogFn("Bluetooth scan", "No Bluetooth devices found", window)
					return
				}

				showScanDialogFn(window, devices, func(device BluetoothScanDevice) {
					bluetoothAddressEntry.SetText(device.Address)
					status.SetText("Selected: " + device.Address)
				})
			})
		})
	}

	refreshPorts := func() {
		selectedPort := strings.TrimSpace(serialPortSelect.Selected)
		ports, err := serial.GetPortsList()
		if err != nil {
			status.SetText("Failed to list serial ports: " + err.Error())
			return
		}
		sort.Strings(ports)

		if currentPort := strings.TrimSpace(current.Connection.SerialPort); currentPort != "" {
			ports = append(ports, currentPort)
		}
		if selectedPort != "" {
			ports = append(ports, selectedPort)
		}
		ports = uniqueValues(ports)
		serialPortSelect.SetOptions(ports)

		if selectedPort != "" {
			serialPortSelect.SetSelected(selectedPort)
		} else if current.Connection.SerialPort != "" {
			serialPortSelect.SetSelected(current.Connection.SerialPort)
		}

		if len(ports) == 0 {
			status.SetText("No serial ports detected")
			return
		}
		status.SetText("")
	}

	refreshPortsButton := widget.NewButton("Refresh", refreshPorts)
	serialPortRow := container.NewBorder(nil, nil, nil, refreshPortsButton, serialPortSelect)

	connectorLabel := widget.NewLabel("Connector")
	ipHostLabel := widget.NewLabel("IP Host")
	serialPortLabel := widget.NewLabel("Serial Port")
	serialBaudLabel := widget.NewLabel("Serial Baud")
	bluetoothAddressLabel := widget.NewLabel("Bluetooth Address")
	bluetoothAdapterLabel := widget.NewLabel("Bluetooth Adapter")
	bluetoothActionsLabel := widget.NewLabel("")
	bluetoothHintLabel := widget.NewLabel("")

	connectionFields := container.New(layout.NewFormLayout(),
		connectorLabel, connectorSelect,
		ipHostLabel, hostEntry,
		serialPortLabel, serialPortRow,
		serialBaudLabel, serialBaudSelect,
		bluetoothAddressLabel, bluetoothAddressEntry,
		bluetoothAdapterLabel, bluetoothAdapterEntry,
		bluetoothActionsLabel, bluetoothActionRow,
		bluetoothHintLabel, bluetoothPairingHint,
	)

	setConnectorFields := func(connector config.ConnectorType) {
		showIP := connector == config.ConnectorIP
		showSerial := connector == config.ConnectorSerial
		showBluetooth := connector == config.ConnectorBluetooth

		setVisible(showIP, ipHostLabel, hostEntry)
		setVisible(showSerial, serialPortLabel, serialPortRow, serialBaudLabel, serialBaudSelect)
		setVisible(showBluetooth, bluetoothAddressLabel, bluetoothAddressEntry, bluetoothAdapterLabel, bluetoothAdapterEntry, bluetoothActionsLabel, bluetoothActionRow, bluetoothHintLabel, bluetoothPairingHint)
	}

	connectorSelect.OnChanged = func(value string) {
		next := connectorTypeFromOption(value)
		setConnectorFields(next)
		if next == config.ConnectorSerial {
			refreshPorts()
		}
		if next == config.ConnectorBluetooth {
			status.SetText("Pair the node in OS Bluetooth settings before connecting.")
			return
		}
		status.SetText("")
	}
	setConnectorFields(current.Connection.Connector)
	if current.Connection.Connector == config.ConnectorSerial {
		refreshPorts()
	}

	saveButton := widget.NewButton("Save", func() {
		connector := connectorTypeFromOption(connectorSelect.Selected)

		baud := current.Connection.SerialBaud
		if connector == config.ConnectorSerial {
			var err error
			baud, err = parseSerialBaud(serialBaudSelect.Selected)
			if err != nil {
				status.SetText("Save failed: " + err.Error())
				return
			}
		}

		cfg := current
		cfg.Connection.Connector = connector
		cfg.Connection.Host = strings.TrimSpace(hostEntry.Text)
		cfg.Connection.SerialPort = strings.TrimSpace(serialPortSelect.Selected)
		cfg.Connection.SerialBaud = baud
		cfg.Connection.BluetoothAddress = strings.TrimSpace(bluetoothAddressEntry.Text)
		cfg.Connection.BluetoothAdapter = strings.TrimSpace(bluetoothAdapterEntry.Text)
		cfg.Logging.Level = levelSelect.Selected
		cfg.Logging.LogToFile = logToFile.Checked
		cfg.UI.Autostart.Enabled = autostartEnabled.Checked
		cfg.UI.Autostart.Mode = autostartModeFromOption(autostartModeSelect.Selected)

		saveConfig := func(clearDatabase bool) {
			if clearDatabase {
				if dep.Actions.OnClearDB == nil {
					status.SetText("Save failed: database clear is not available")
					return
				}
				if err := dep.Actions.OnClearDB(); err != nil {
					status.SetText("Save failed: database clear failed: " + err.Error())
					return
				}
			}
			if err := dep.Actions.OnSave(cfg); err != nil {
				var warning *app.AutostartSyncWarning
				if errors.As(err, &warning) {
					current = cfg
					status.SetText("Saved with warning: " + warning.Error())
					return
				}
				status.SetText("Save failed: " + err.Error())
				return
			}
			current = cfg
			status.SetText("Saved")
		}

		if connector != current.Connection.Connector {
			window := currentWindow()
			if window == nil {
				status.SetText("Save failed: active window is unavailable")
				return
			}
			dialog.ShowConfirm(
				"Switch transport?",
				"Changing transport will clear the local database before reconnecting. Continue?",
				func(ok bool) {
					if !ok {
						status.SetText("Save canceled")
						return
					}
					saveConfig(true)
				},
				window,
			)
			return
		}

		saveConfig(false)
	})
	saveButton.Importance = widget.HighImportance

	clearDBButton := widget.NewButton("Clear database", func() {
		if dep.Actions.OnClearDB == nil {
			status.SetText("Database clear is not available")
			return
		}
		if err := dep.Actions.OnClearDB(); err != nil {
			status.SetText("Database clear failed: " + err.Error())
			return
		}
		status.SetText("Database cleared")
	})
	if dep.Actions.OnClearDB == nil {
		clearDBButton.Disable()
	}

	loggingForm := widget.NewForm(
		widget.NewFormItem("Log Level", levelSelect),
		widget.NewFormItem("Log to file", logToFile),
	)
	startupForm := widget.NewForm(
		widget.NewFormItem("Run on system startup", autostartEnabled),
		widget.NewFormItem("Startup mode", autostartModeSelect),
	)

	connectionBlock := widget.NewCard("Connection", "", container.NewVBox(
		connStatusLabel,
		connectionFields,
	))
	startupBlock := widget.NewCard("Startup", "", startupForm)
	loggingBlock := widget.NewCard("Logging", "", loggingForm)
	maintenanceBlock := widget.NewCard("Maintenance", "", container.NewVBox(
		clearDBButton,
	))

	logo := newLinkImage(resources.LogoTextResource(), fyne.NewSize(220, 80), func() {
		if err := openExternalURL(app.SourceURL); err != nil {
			status.SetText("Failed to open source website: " + err.Error())
		}
	})

	sourceLink := widget.NewHyperlink("Source", mustParseURL(app.SourceURL))
	meshtasticLink := widget.NewHyperlink("Meshtastic", mustParseURL(app.MeshtasticURL))
	poweredByRow := container.NewHBox(
		widget.NewLabel("Powered by "),
		meshtasticLink,
	)
	versionBlock := widget.NewCard("", "", container.NewVBox(
		container.NewHBox(logo, layout.NewSpacer()),
		widget.NewLabel("Version: "+app.BuildVersionWithDate()),
		sourceLink,
		poweredByRow,
	))

	content := container.NewVBox(
		widget.NewLabel("App settings"),
		connectionBlock,
		startupBlock,
		loggingBlock,
		maintenanceBlock,
		saveButton,
		versionBlock,
		status,
	)

	return container.NewVScroll(content)
}

func showBluetoothScanDialog(window fyne.Window, devices []BluetoothScanDevice, onSelect func(BluetoothScanDevice)) {
	selected := 0

	list := widget.NewList(
		func() int {
			return len(devices)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabel(" ")
			title.Truncation = fyne.TextTruncateEllipsis
			details := widget.NewLabel(" ")
			details.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, details)
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			row, ok := object.(*fyne.Container)
			if !ok || len(row.Objects) < 2 {
				return
			}
			title, titleOK := row.Objects[0].(*widget.Label)
			details, detailsOK := row.Objects[1].(*widget.Label)
			if !titleOK || !detailsOK {
				return
			}
			device, ok := bluetoothScanDeviceAt(devices, id)
			if !ok {
				title.SetText("")
				details.SetText("")
				return
			}
			title.SetText(bluetoothScanDeviceTitle(device))
			details.SetText(bluetoothScanDeviceDetails(device))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selected = id
	}
	if len(devices) > 0 {
		list.Select(0)
	}

	dialogContent := container.NewBorder(nil, nil, nil, nil, list)
	scanDialog := dialog.NewCustomConfirm(
		"Bluetooth devices",
		"Select",
		"Cancel",
		dialogContent,
		func(ok bool) {
			if !ok {
				return
			}
			device, ok := bluetoothScanDeviceAt(devices, selected)
			if !ok {
				return
			}
			onSelect(device)
		},
		window,
	)
	scanDialog.Resize(fyne.NewSize(560, 420))
	scanDialog.Show()
}

func openExternalURL(rawURL string) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}

	currentApp := fyne.CurrentApp()
	if currentApp == nil {
		return fmt.Errorf("application is not initialized")
	}
	if err := currentApp.OpenURL(parsed); err != nil {
		return fmt.Errorf("open url: %w", err)
	}
	return nil
}

func mustParseURL(rawURL string) *url.URL {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		panic(fmt.Sprintf("invalid url %q: %v", rawURL, err))
	}
	return parsed
}

func setVisible(visible bool, objects ...fyne.CanvasObject) {
	for _, object := range objects {
		if visible {
			object.Show()
			continue
		}
		object.Hide()
	}
}

func uniqueValues(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	return unique
}

func currentWindow() fyne.Window {
	currentApp := fyne.CurrentApp()
	if currentApp == nil || currentApp.Driver() == nil {
		return nil
	}
	windows := currentApp.Driver().AllWindows()
	if len(windows) == 0 {
		return nil
	}
	return windows[0]
}

func connectorOptionFromType(connector config.ConnectorType) string {
	switch connector {
	case config.ConnectorIP:
		return connectorOptionIP
	case config.ConnectorSerial:
		return connectorOptionSerial
	case config.ConnectorBluetooth:
		return connectorOptionBluetooth
	default:
		return connectorOptionIP
	}
}

func connectorTypeFromOption(value string) config.ConnectorType {
	switch strings.TrimSpace(value) {
	case connectorOptionIP:
		return config.ConnectorIP
	case connectorOptionSerial:
		return config.ConnectorSerial
	case connectorOptionBluetooth:
		return config.ConnectorBluetooth
	default:
		return config.ConnectorIP
	}
}

func autostartOptionFromMode(mode config.AutostartMode) string {
	switch mode {
	case config.AutostartModeBackground:
		return autostartOptionTray
	default:
		return autostartOptionNormal
	}
}

func autostartModeFromOption(value string) config.AutostartMode {
	switch strings.TrimSpace(value) {
	case autostartOptionTray:
		return config.AutostartModeBackground
	default:
		return config.AutostartModeNormal
	}
}

func parseSerialBaud(value string) (int, error) {
	baud, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("invalid serial baud %q", value)
	}
	if baud <= 0 {
		return 0, fmt.Errorf("serial baud must be positive")
	}
	return baud, nil
}
