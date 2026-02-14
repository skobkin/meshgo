package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
var settingsLogger = slog.With("component", "ui.settings")
var externalURLLogger = slog.With("component", "ui.external_url")

func newSettingsTab(dep RuntimeDependencies, connStatusLabel *widget.Label) fyne.CanvasObject {
	current := dep.Data.Config
	current.FillMissingDefaults()
	settingsLogger.Debug(
		"building settings tab",
		"connector", current.Connection.Connector,
		"log_level", strings.ToLower(strings.TrimSpace(current.Logging.Level)),
		"log_to_file", current.Logging.LogToFile,
		"autostart_enabled", current.UI.Autostart.Enabled,
		"autostart_mode", current.UI.Autostart.Mode,
		"notify_when_focused", current.UI.Notifications.NotifyWhenFocused,
		"notify_incoming_message", current.UI.Notifications.Events.IncomingMessage,
		"notify_node_discovered", current.UI.Notifications.Events.NodeDiscovered,
		"notify_connection_status", current.UI.Notifications.Events.ConnectionStatus,
		"notify_update_available", current.UI.Notifications.Events.UpdateAvailable,
	)

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

	notifyWhenFocused := widget.NewCheck("Notify when app is focused", nil)
	notifyWhenFocused.SetChecked(current.UI.Notifications.NotifyWhenFocused)
	notifyIncomingMessage := widget.NewCheck("Incoming chat messages", nil)
	notifyIncomingMessage.SetChecked(current.UI.Notifications.Events.IncomingMessage)
	notifyNodeDiscovered := widget.NewCheck("New node discovered", nil)
	notifyNodeDiscovered.SetChecked(current.UI.Notifications.Events.NodeDiscovered)
	notifyConnectionStatus := widget.NewCheck("Connection status changes", nil)
	notifyConnectionStatus.SetChecked(current.UI.Notifications.Events.ConnectionStatus)
	notifyUpdateAvailable := widget.NewCheck("Update available", nil)
	notifyUpdateAvailable.SetChecked(current.UI.Notifications.Events.UpdateAvailable)

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
		settingsLogger.Info(
			"opening Bluetooth settings from UI",
			"adapter", strings.TrimSpace(bluetoothAdapterEntry.Text),
		)
		if err := openBluetoothSettingsFn(); err != nil {
			settingsLogger.Warn("open Bluetooth settings failed", "error", err)
			status.SetText("Failed to open Bluetooth settings: " + err.Error())

			return
		}
		settingsLogger.Debug("opened Bluetooth settings")
		status.SetText("")
	})
	bluetoothActionRow := container.NewHBox(scanBluetoothButton, openBluetoothSettingsButton)

	scanBluetoothButton.OnTapped = func() {
		adapterID := strings.TrimSpace(bluetoothAdapterEntry.Text)
		settingsLogger.Info("starting Bluetooth scan", "adapter", adapterID)
		window := currentWindowFn()
		if window == nil {
			settingsLogger.Warn("Bluetooth scan failed: active window unavailable")
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

		runAsync(func() {
			devices, err := bluetoothScanner.Scan(context.Background(), adapterID)
			runOnUI(func() {
				progressBar.Stop()
				progress.Hide()
				scanBluetoothButton.Enable()
				openBluetoothSettingsButton.Enable()

				if err != nil {
					settingsLogger.Warn("Bluetooth scan failed", "adapter", adapterID, "error", err)
					status.SetText("Bluetooth scan failed: " + err.Error())
					showErrorDialogFn(err, window)

					return
				}
				settingsLogger.Info("Bluetooth scan finished", "adapter", adapterID, "devices_found", len(devices))
				if len(devices) == 0 {
					status.SetText("No Bluetooth devices found")
					showInfoDialogFn("Bluetooth scan", "No Bluetooth devices found", window)

					return
				}

				showScanDialogFn(window, devices, func(device DiscoveredBluetoothDevice) {
					bluetoothAddressEntry.SetText(device.Address)
					status.SetText("Selected: " + device.Address)
				})
			})
		})
	}

	refreshPorts := func() {
		selectedPort := strings.TrimSpace(serialPortSelect.Selected)
		settingsLogger.Debug("refreshing serial ports list", "selected_port", selectedPort)
		ports, err := serial.GetPortsList()
		if err != nil {
			settingsLogger.Warn("refreshing serial ports failed", "error", err)
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
			settingsLogger.Info("serial ports refresh completed: no ports detected")
			status.SetText("No serial ports detected")

			return
		}
		settingsLogger.Info("serial ports refreshed", "ports_detected", len(ports))
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
		settingsLogger.Debug("connector changed", "selected", strings.TrimSpace(value), "connector", next)
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

	applyConfigToForm := func(next config.AppConfig) {
		next.FillMissingDefaults()

		connectorSelect.SetSelected(connectorOptionFromType(next.Connection.Connector))
		hostEntry.SetText(next.Connection.Host)
		serialPortSelect.SetSelected(next.Connection.SerialPort)
		serialBaudSelect.SetOptions(uniqueValues(append(defaultSerialBaudOptions, strconv.Itoa(next.Connection.SerialBaud))))
		serialBaudSelect.SetSelected(strconv.Itoa(next.Connection.SerialBaud))
		bluetoothAddressEntry.SetText(next.Connection.BluetoothAddress)
		bluetoothAdapterEntry.SetText(next.Connection.BluetoothAdapter)

		levelSelect.SetSelected(strings.ToLower(next.Logging.Level))
		if strings.TrimSpace(levelSelect.Selected) == "" {
			levelSelect.SetSelected("info")
		}
		logToFile.SetChecked(next.Logging.LogToFile)

		autostartEnabled.SetChecked(next.UI.Autostart.Enabled)
		autostartModeSelect.SetSelected(autostartOptionFromMode(next.UI.Autostart.Mode))
		if strings.TrimSpace(autostartModeSelect.Selected) == "" {
			autostartModeSelect.SetSelected(autostartOptionNormal)
		}
		setAutostartModeEnabled(autostartEnabled.Checked)

		notifyWhenFocused.SetChecked(next.UI.Notifications.NotifyWhenFocused)
		notifyIncomingMessage.SetChecked(next.UI.Notifications.Events.IncomingMessage)
		notifyNodeDiscovered.SetChecked(next.UI.Notifications.Events.NodeDiscovered)
		notifyConnectionStatus.SetChecked(next.UI.Notifications.Events.ConnectionStatus)
		notifyUpdateAvailable.SetChecked(next.UI.Notifications.Events.UpdateAvailable)

		setConnectorFields(next.Connection.Connector)
		if next.Connection.Connector == config.ConnectorSerial {
			refreshPorts()
		}
		if next.Connection.Connector == config.ConnectorBluetooth {
			status.SetText("Pair the node in OS Bluetooth settings before connecting.")

			return
		}
		status.SetText("")
	}

	saveButton := widget.NewButton("Save", func() {
		connector := connectorTypeFromOption(connectorSelect.Selected)
		settingsLogger.Info(
			"settings save requested",
			"connector", connector,
			"log_level", strings.TrimSpace(levelSelect.Selected),
			"log_to_file", logToFile.Checked,
			"autostart_enabled", autostartEnabled.Checked,
			"autostart_mode", autostartModeFromOption(autostartModeSelect.Selected),
			"notify_when_focused", notifyWhenFocused.Checked,
			"notify_incoming_message", notifyIncomingMessage.Checked,
			"notify_node_discovered", notifyNodeDiscovered.Checked,
			"notify_connection_status", notifyConnectionStatus.Checked,
			"notify_update_available", notifyUpdateAvailable.Checked,
		)

		baud := current.Connection.SerialBaud
		if connector == config.ConnectorSerial {
			var err error
			baud, err = parseSerialBaud(serialBaudSelect.Selected)
			if err != nil {
				settingsLogger.Warn("settings save failed: invalid serial baud", "value", strings.TrimSpace(serialBaudSelect.Selected), "error", err)
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
		cfg.UI.Notifications.NotifyWhenFocused = notifyWhenFocused.Checked
		cfg.UI.Notifications.Events.IncomingMessage = notifyIncomingMessage.Checked
		cfg.UI.Notifications.Events.NodeDiscovered = notifyNodeDiscovered.Checked
		cfg.UI.Notifications.Events.ConnectionStatus = notifyConnectionStatus.Checked
		cfg.UI.Notifications.Events.UpdateAvailable = notifyUpdateAvailable.Checked

		saveConfig := func(clearDatabase bool) {
			settingsLogger.Info("applying settings", "clear_database", clearDatabase, "connector", cfg.Connection.Connector)
			if clearDatabase {
				if dep.Actions.OnClearDB == nil {
					settingsLogger.Warn("settings save failed: database clear action unavailable")
					status.SetText("Save failed: database clear is not available")

					return
				}
				if err := dep.Actions.OnClearDB(); err != nil {
					settingsLogger.Warn("settings save failed: database clear failed", "error", err)
					status.SetText("Save failed: database clear failed: " + err.Error())

					return
				}
				settingsLogger.Info("database cleared before transport switch")
			}
			if err := dep.Actions.OnSave(cfg); err != nil {
				var devWarning *app.AutostartDevBuildSkipWarning
				if errors.As(err, &devWarning) {
					settingsLogger.Info("settings saved with dev-build autostart skip", "autostart_enabled", devWarning.Enabled)
					current = cfg
					status.SetText("Saved")

					if devWarning.Enabled {
						window := currentWindowFn()
						if window == nil {
							settingsLogger.Warn("autostart dev-build info dialog skipped: active window unavailable")
						} else {
							showInfoDialogFn(
								"Autostart in dev build",
								"Autostart entry was not rewritten because dev builds do not support autorun sync. Other settings were saved.",
								window,
							)
						}
					}

					return
				}

				var warning *app.AutostartSyncWarning
				if errors.As(err, &warning) {
					settingsLogger.Info("settings saved with warning", "warning", warning.Error())
					current = cfg
					status.SetText("Saved with warning: " + warning.Error())

					return
				}
				settingsLogger.Warn("settings save failed", "error", err)
				status.SetText("Save failed: " + err.Error())

				return
			}
			settingsLogger.Info("settings saved successfully", "connector", cfg.Connection.Connector)
			current = cfg
			status.SetText("Saved")
		}

		if connector != current.Connection.Connector {
			settingsLogger.Info(
				"transport change requires confirmation",
				"from", current.Connection.Connector,
				"to", connector,
			)
			window := currentWindow()
			if window == nil {
				settingsLogger.Warn("settings save failed: active window unavailable for transport confirmation")
				status.SetText("Save failed: active window is unavailable")

				return
			}
			dialog.ShowConfirm(
				"Switch transport?",
				"Changing transport will clear the local database before reconnecting. Continue?",
				func(ok bool) {
					if !ok {
						settingsLogger.Info("transport switch canceled by user")
						status.SetText("Save canceled")

						return
					}
					settingsLogger.Info("transport switch confirmed by user")
					saveConfig(true)
				},
				window,
			)

			return
		}

		saveConfig(false)
	})
	saveButton.Importance = widget.HighImportance

	revertButton := widget.NewButton("Revert", func() {
		settingsLogger.Info("settings revert requested")
		applyConfigToForm(current)
		status.SetText("Unsaved changes reverted")
	})

	clearDBButton := widget.NewButton("Clear database", func() {
		settingsLogger.Info("clear database requested from settings UI")
		if dep.Actions.OnClearDB == nil {
			settingsLogger.Warn("clear database unavailable: action is not configured")
			status.SetText("Database clear is not available")

			return
		}
		if err := dep.Actions.OnClearDB(); err != nil {
			settingsLogger.Warn("database clear failed", "error", err)
			status.SetText("Database clear failed: " + err.Error())

			return
		}
		settingsLogger.Info("database cleared from settings UI")
		status.SetText("Database cleared")
	})
	if dep.Actions.OnClearDB == nil {
		clearDBButton.Disable()
	}

	clearCacheButton := widget.NewButton("Clear cache", func() {
		settingsLogger.Info("clear cache requested from settings UI")
		if dep.Actions.OnClearCache == nil {
			settingsLogger.Warn("clear cache unavailable: action is not configured")
			status.SetText("Cache clear is not available")

			return
		}
		if err := dep.Actions.OnClearCache(); err != nil {
			settingsLogger.Warn("cache clear failed", "error", err)
			status.SetText("Cache clear failed: " + err.Error())

			return
		}
		settingsLogger.Info("cache cleared from settings UI")
		status.SetText("Cache cleared")
	})
	if dep.Actions.OnClearCache == nil {
		clearCacheButton.Disable()
	}

	loggingForm := widget.NewForm(
		widget.NewFormItem("Log Level", levelSelect),
		widget.NewFormItem("Log to file", logToFile),
	)
	startupForm := widget.NewForm(
		widget.NewFormItem("Run on system startup", autostartEnabled),
		widget.NewFormItem("Startup mode", autostartModeSelect),
	)
	notificationsContent := container.NewVBox(
		notifyWhenFocused,
		notifyIncomingMessage,
		notifyNodeDiscovered,
		notifyConnectionStatus,
		notifyUpdateAvailable,
	)

	connectionBlock := widget.NewCard("Connection", "", container.NewVBox(
		connStatusLabel,
		connectionFields,
	))
	startupBlock := widget.NewCard("Startup", "", startupForm)
	notificationsBlock := widget.NewCard("Notifications", "", notificationsContent)
	loggingBlock := widget.NewCard("Logging", "", loggingForm)
	maintenanceBlock := widget.NewCard("Maintenance", "", container.NewGridWithColumns(2,
		clearDBButton,
		clearCacheButton,
	))

	logo := newLinkImage(resources.LogoTextResource(), fyne.NewSize(220, 80), func() {
		settingsLogger.Debug("opening source URL from settings logo", "url", app.SourceURL)
		if err := openExternalURL(app.SourceURL); err != nil {
			settingsLogger.Warn("open source URL failed", "url", app.SourceURL, "error", err)
			status.SetText("Failed to open source website: " + err.Error())
		}
	})

	sourceLink := newSafeHyperlink("Source", app.SourceURL, status)
	meshtasticLink := newSafeHyperlink("Meshtastic", app.MeshtasticURL, status)
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

	generalTab := newSettingsSubTabPage(startupBlock)
	connectionTab := newSettingsSubTabPage(connectionBlock)
	notificationsTab := newSettingsSubTabPage(notificationsBlock)
	maintenanceTab := newSettingsSubTabPage(loggingBlock, maintenanceBlock)
	aboutTab := newSettingsSubTabPage(versionBlock)

	subTabs := container.NewAppTabs(
		container.NewTabItem("General", generalTab),
		container.NewTabItem("Connection", connectionTab),
		container.NewTabItem("Notifications", notificationsTab),
		container.NewTabItem("Maintenance", maintenanceTab),
		container.NewTabItem("About", aboutTab),
	)
	subTabs.SetTabLocation(container.TabLocationTop)

	buttonRow := container.NewGridWithColumns(2, saveButton, revertButton)
	bottomBar := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(status),
		container.NewPadded(buttonRow),
	)

	return container.NewBorder(nil, bottomBar, nil, nil, subTabs)
}

func newSettingsSubTabPage(content ...fyne.CanvasObject) fyne.CanvasObject {
	page := container.NewVBox(content...)

	return container.NewVScroll(container.NewPadded(page))
}

func showBluetoothScanDialog(window fyne.Window, devices []DiscoveredBluetoothDevice, onSelect func(DiscoveredBluetoothDevice)) {
	settingsLogger.Debug("showing Bluetooth scan dialog", "device_count", len(devices))
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
				settingsLogger.Debug("Bluetooth scan dialog canceled")

				return
			}
			device, ok := bluetoothScanDeviceAt(devices, selected)
			if !ok {
				settingsLogger.Debug("Bluetooth scan dialog selection ignored: invalid index", "selected", selected)

				return
			}
			settingsLogger.Info("Bluetooth device selected", "address", strings.TrimSpace(device.Address), "name", strings.TrimSpace(device.Name))
			onSelect(device)
		},
		window,
	)
	scanDialog.Resize(fyne.NewSize(560, 420))
	scanDialog.Show()
}

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
