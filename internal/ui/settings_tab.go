package ui

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/resources"
)

const (
	connectorOptionIP                = "IP"
	connectorOptionSerial            = "Serial"
	connectorOptionBluetoothDisabled = "Bluetooth (disabled)"
)

var defaultSerialBaudOptions = []string{"9600", "19200", "38400", "57600", "115200", "230400", "460800", "921600"}

func newSettingsTab(dep Dependencies, connStatusLabel *widget.Label) fyne.CanvasObject {
	current := dep.Config
	current.ApplyDefaults()

	connectorSelect := widget.NewSelect([]string{
		connectorOptionIP,
		connectorOptionSerial,
		connectorOptionBluetoothDisabled,
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

	status := widget.NewLabel("")

	serialPortSelect := widget.NewSelect(nil, nil)
	serialPortSelect.PlaceHolder = "Select serial port"
	serialPortSelect.SetSelected(current.Connection.SerialPort)

	serialBaudSelect := widget.NewSelect(uniqueValues(append(defaultSerialBaudOptions, strconv.Itoa(current.Connection.SerialBaud))), nil)
	serialBaudSelect.SetSelected(strconv.Itoa(current.Connection.SerialBaud))

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

	connectionFields := container.New(layout.NewFormLayout(),
		connectorLabel, connectorSelect,
		ipHostLabel, hostEntry,
		serialPortLabel, serialPortRow,
		serialBaudLabel, serialBaudSelect,
	)

	setConnectorFields := func(connector config.ConnectorType) {
		showIP := connector == config.ConnectorIP
		showSerial := connector == config.ConnectorSerial

		setVisible(showIP, ipHostLabel, hostEntry)
		setVisible(showSerial, serialPortLabel, serialPortRow, serialBaudLabel, serialBaudSelect)
	}

	connectorSelect.OnChanged = func(value string) {
		next := connectorTypeFromOption(value)
		if next == config.ConnectorBluetooth {
			status.SetText("Bluetooth connector is not implemented")
			connectorSelect.SetSelected(connectorOptionFromType(current.Connection.Connector))
			return
		}

		setConnectorFields(next)
		if next == config.ConnectorSerial {
			refreshPorts()
		}
		status.SetText("")
	}
	setConnectorFields(current.Connection.Connector)
	if current.Connection.Connector == config.ConnectorSerial {
		refreshPorts()
	}

	saveButton := widget.NewButton("Save", func() {
		connector := connectorTypeFromOption(connectorSelect.Selected)
		if connector == config.ConnectorBluetooth {
			status.SetText("Bluetooth connector is not implemented")
			return
		}

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
		cfg.Logging.Level = levelSelect.Selected
		cfg.Logging.LogToFile = logToFile.Checked

		if err := dep.OnSave(cfg); err != nil {
			status.SetText("Save failed: " + err.Error())
			return
		}
		current = cfg
		status.SetText("Saved")
	})
	saveButton.Importance = widget.HighImportance

	clearDBButton := widget.NewButton("Clear database", func() {
		if dep.OnClearDB == nil {
			status.SetText("Database clear is not available")
			return
		}
		if err := dep.OnClearDB(); err != nil {
			status.SetText("Database clear failed: " + err.Error())
			return
		}
		status.SetText("Database cleared")
	})
	if dep.OnClearDB == nil {
		clearDBButton.Disable()
	}

	loggingForm := widget.NewForm(
		widget.NewFormItem("Log Level", levelSelect),
		widget.NewFormItem("Log to file", logToFile),
	)

	connectionBlock := widget.NewCard("Connection", "", container.NewVBox(
		connStatusLabel,
		connectionFields,
	))
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
		loggingBlock,
		maintenanceBlock,
		saveButton,
		versionBlock,
		status,
	)

	return container.NewVScroll(content)
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

func connectorOptionFromType(connector config.ConnectorType) string {
	switch connector {
	case config.ConnectorIP:
		return connectorOptionIP
	case config.ConnectorSerial:
		return connectorOptionSerial
	case config.ConnectorBluetooth:
		return connectorOptionBluetoothDisabled
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
	case connectorOptionBluetoothDisabled:
		return config.ConnectorBluetooth
	default:
		return config.ConnectorIP
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
