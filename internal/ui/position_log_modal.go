package ui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/config"
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
			mapLinkProvider := currentMapLinkProvider(dep)
			body.Objects = []fyne.CanvasObject{newPositionLogTable(rows, mapLinkProvider, func(target *url.URL) {
				if target == nil {
					return
				}
				if err := openExternalURL(target.String()); err != nil {
					showErrorModal(dep, fmt.Errorf("open map link: %w", err))
				}
			})}
			body.Refresh()
		})
	}()
}

func currentMapLinkProvider(dep RuntimeDependencies) config.MapLinkProvider {
	if dep.Data.CurrentConfig != nil {
		current := dep.Data.CurrentConfig()

		return normalizeMapLinkProvider(current.UI.MapDisplay.MapLinkProvider)
	}

	return normalizeMapLinkProvider(dep.Data.Config.UI.MapDisplay.MapLinkProvider)
}

func newPositionLogTable(
	items []domain.NodePositionHistoryEntry,
	mapLinkProvider config.MapLinkProvider,
	onOpenMapLink func(target *url.URL),
) fyne.CanvasObject {
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
	rows := make([]positionLogRowView, 0, len(items))
	for _, item := range items {
		rows = append(rows, positionLogRowViewFromEntry(item, mapLinkProvider))
	}
	if len(rows) == 0 {
		rows = append(rows, positionLogRowView{
			Values: []string{"No position history yet", "", "", "", "", "", "", ""},
		})
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(rows) + 1, len(headers)
		},
		func() fyne.CanvasObject {
			rich := widget.NewRichText(&widget.TextSegment{Text: "", Style: widget.RichTextStyleInline})
			rich.Wrapping = fyne.TextWrapWord

			return rich
		},
		func(id widget.TableCellID, object fyne.CanvasObject) {
			rich, ok := object.(*widget.RichText)
			if !ok {
				return
			}
			if id.Row == 0 {
				rich.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: headers[id.Col], Style: widget.RichTextStyleStrong},
				}
				rich.Refresh()

				return
			}

			row := rows[id.Row-1]
			if id.Col >= len(row.Values) {
				rich.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: "", Style: widget.RichTextStyleInline},
				}
				rich.Refresh()

				return
			}
			style := widget.RichTextStyleInline
			if id.Col == 0 && row.LatitudeLink != nil {
				style.ColorName = theme.ColorNameHyperlink
				style.TextStyle = fyne.TextStyle{Underline: true}
			}
			rich.Segments = []widget.RichTextSegment{
				&widget.TextSegment{Text: row.Values[id.Col], Style: style},
			}
			rich.Refresh()
		},
	)
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 || id.Col != 0 || onOpenMapLink == nil {
			return
		}
		index := id.Row - 1
		if index < 0 || index >= len(rows) {
			return
		}
		onOpenMapLink(rows[index].LatitudeLink)
	}
	table.OnUnselected = func(widget.TableCellID) {}
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

type positionLogRowView struct {
	Values       []string
	LatitudeLink *url.URL
}

func positionLogRowViewFromEntry(item domain.NodePositionHistoryEntry, mapLinkProvider config.MapLinkProvider) positionLogRowView {
	return positionLogRowView{
		Values:       positionLogRow(item),
		LatitudeLink: positionLogLatitudeURL(item, mapLinkProvider),
	}
}

func positionLogLatitudeURL(item domain.NodePositionHistoryEntry, mapLinkProvider config.MapLinkProvider) *url.URL {
	if item.Latitude == nil || item.Longitude == nil {
		return nil
	}

	parsed, err := nodePositionMapURL(mapLinkProvider, *item.Latitude, *item.Longitude, item.Precision)
	if err != nil {
		return nil
	}

	return parsed
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
