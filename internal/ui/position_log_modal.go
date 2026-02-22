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

func handleNodePositionLogAction(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
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

	showPositionLogModal(window, dep, node)
}

func showPositionLogModal(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	loading := widget.NewLabel("Loading position history...")
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
		rows, err := dep.Actions.NodeOverview.ListPositionHistory(ctx, strings.TrimSpace(node.NodeID), 0)
		fyne.Do(func() {
			if err != nil {
				modal.Hide()
				showErrorModal(dep, fmt.Errorf("load position history: %w", err))

				return
			}
			body.Objects = []fyne.CanvasObject{newPositionLogTable(rows)}
			body.Refresh()
		})
	}()
}

func newPositionLogTable(items []domain.NodePositionHistoryEntry) fyne.CanvasObject {
	headers := []string{
		"Latitude",
		"Longitude",
		"Altitude",
		"Precision",
		"Channel",
		"Update",
		"Observed at",
		// "From packet",
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, positionLogRow(item))
	}
	if len(rows) == 0 {
		rows = append(rows, []string{
			"No position history yet",
			"", "", "", "", "", "", "",
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
	table.SetColumnWidth(0, 110)
	table.SetColumnWidth(1, 110)
	table.SetColumnWidth(2, 110)
	table.SetColumnWidth(3, 110)
	table.SetColumnWidth(4, 90)  // Channel
	table.SetColumnWidth(5, 100) // Update
	table.SetColumnWidth(6, 170) // Observed at

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Position log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(table),
	)
}

func positionLogRow(item domain.NodePositionHistoryEntry) []string {
	return []string{
		formatFloat64(item.Latitude, "%.6f"),
		formatFloat64(item.Longitude, "%.6f"),
		formatInt32(item.Altitude, "%d m"),
		formatPositionPrecision(item.Precision),
		formatUint32(item.Channel, "%d"),
		telemetryLogUpdateType(item.UpdateType),
		telemetryLogTime(item.ObservedAt),
		// yesNo(item.FromPacket),
	}
}

func formatInt32(value *int32, format string) string {
	if value == nil {
		return "unknown"
	}

	return fmt.Sprintf(format, *value)
}

func formatPositionPrecision(value *uint32) string {
	if value == nil {
		return "unknown"
	}

	return nodeChannelPositionPrecisionLabel(*value)
}
