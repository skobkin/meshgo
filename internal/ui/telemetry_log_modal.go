package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

func handleNodeTelemetryLogAction(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	if dep.Actions.NodeOverview == nil {
		showErrorModal(dep, fmt.Errorf("node overview actions are unavailable"))

		return
	}
	nodeID := strings.TrimSpace(node.NodeID)
	if nodeID == "" {
		return
	}
	if window == nil {
		window = currentRuntimeWindow(dep)
	}
	if window == nil {
		return
	}

	showTelemetryLogModal(window, dep, node)
}

func showTelemetryLogModal(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	loading := widget.NewLabel("Loading telemetry history...")
	body := container.NewStack(loading)
	closeButton := widget.NewButton("Close", nil)
	content := container.NewBorder(nil, closeButton, nil, nil, body)
	modal := widget.NewModalPopUp(content, window.Canvas())
	closeButton.OnTapped = modal.Hide
	modal.Resize(fyne.NewSize(1040, 560))
	modal.Show()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := dep.Actions.NodeOverview.ListTelemetryHistory(ctx, strings.TrimSpace(node.NodeID), 0)
		fyne.Do(func() {
			if err != nil {
				modal.Hide()
				showErrorModal(dep, fmt.Errorf("load telemetry history: %w", err))

				return
			}
			body.Objects = []fyne.CanvasObject{newTelemetryLogTable(rows)}
			body.Refresh()
		})
	}()
}

func newTelemetryLogTable(items []domain.NodeTelemetryHistoryEntry) fyne.CanvasObject {
	headers := []string{
		"Observed at",
		"Update",
		"From packet",
		"Battery",
		"Voltage",
		"Uptime",
		"Channel util",
		"TX air util",
		"Temperature",
		"Humidity",
		"Pressure",
		"Soil T",
		"Soil M",
		"Gas R",
		"Dew point",
		"Light",
		"UV light",
		"Radiation",
		"AQI",
		"Power V",
		"Power A",
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, telemetryLogRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, []string{
			"No telemetry history yet",
			"", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "",
		})
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(rows) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord

			return label
		},
		func(id widget.TableCellID, object fyne.CanvasObject) {
			label, ok := object.(*widget.Label)
			if !ok {
				return
			}
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}

				return
			}
			label.SetText(rows[id.Row-1][id.Col])
			label.TextStyle = fyne.TextStyle{}
		},
	)
	table.SetColumnWidth(0, 170)
	table.SetColumnWidth(1, 100)
	table.SetColumnWidth(2, 90)
	for col := 3; col < len(headers); col++ {
		table.SetColumnWidth(col, 90)
	}

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Telemetry log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(table),
	)
}

func telemetryLogRow(item domain.NodeTelemetryHistoryEntry) []string {
	return []string{
		telemetryLogTime(item.ObservedAt),
		telemetryLogUpdateType(item.UpdateType),
		yesNo(item.FromPacket),
		formatUint32(item.BatteryLevel, "%d%%"),
		formatFloat64(item.Voltage, "%.2f V"),
		overviewUptime(item.UptimeSeconds),
		formatFloat64(item.ChannelUtilization, "%.2f%%"),
		formatFloat64(item.AirUtilTx, "%.2f%%"),
		formatFloat64(item.Temperature, "%.1f C"),
		formatFloat64(item.Humidity, "%.1f%%"),
		formatFloat64(item.Pressure, "%.1f hPa"),
		formatFloat64(item.SoilTemperature, "%.1f C"),
		formatUint32(item.SoilMoisture, "%d%%"),
		formatFloat64(item.GasResistance, "%.2f MOhm"),
		formatDewPoint(item.Temperature, item.Humidity),
		formatFloat64(item.Lux, "%.1f lx"),
		formatFloat64(item.UVLux, "%.1f UVlx"),
		formatFloat64(item.Radiation, "%.2f uR/h"),
		formatFloat64(item.AirQualityIndex, "%.1f"),
		formatFloat64(item.PowerVoltage, "%.2f V"),
		formatFloat64(item.PowerCurrent, "%.3f A"),
	}
}

func telemetryLogTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}

	return value.Local().Format(time.DateTime)
}

func telemetryLogUpdateType(value domain.NodeUpdateType) string {
	value = domain.NodeUpdateType(strings.TrimSpace(string(value)))
	if value == "" {
		return "unknown"
	}

	return string(value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func formatUint32(value *uint32, format string) string {
	if value == nil {
		return "unknown"
	}

	return fmt.Sprintf(format, *value)
}

func formatFloat64(value *float64, format string) string {
	if value == nil {
		return "unknown"
	}

	return fmt.Sprintf(format, *value)
}

func formatDewPoint(temperature, humidity *float64) string {
	dewPoint, ok := calculateDewPointCelsius(temperature, humidity)
	if !ok {
		return "unknown"
	}

	return fmt.Sprintf("%.1f C", dewPoint)
}
