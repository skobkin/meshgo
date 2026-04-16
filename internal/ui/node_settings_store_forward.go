package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func newNodeStoreForwardSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.store_forward", "Loading Store & Forward settings…", "Store & Forward settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error) {
			return dep.Actions.NodeSettings.LoadStoreForwardSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStoreForwardSettings) error {
			return dep.Actions.NodeSettings.SaveStoreForwardSettings(ctx, target, settings)
		},
		func(v app.NodeStoreForwardSettings) app.NodeStoreForwardSettings { return v },
		func(v app.NodeStoreForwardSettings) string {
			return fmt.Sprintf("%s|%t|%t|%d|%d|%d|%t", v.NodeID, v.Enabled, v.Heartbeat, v.Records, v.HistoryReturnMax, v.HistoryReturnWindow, v.IsServer)
		},
		buildNodeStoreForwardSettingsForm,
	)
}

func buildNodeStoreForwardSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeStoreForwardSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	heartbeat := newSettingsCheck(onChanged)
	records := newNumberEntry(onChanged)
	historyReturnMax := newNumberEntry(onChanged)
	historyReturnWindow := newNumberEntry(onChanged)
	isServer := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Heartbeat", heartbeat),
		widget.NewFormItem("Records", records),
		widget.NewFormItem("History return max", historyReturnMax),
		widget.NewFormItem("History return window", historyReturnWindow),
		widget.NewFormItem("Server mode", isServer),
	)

	return nodeManagedSettingsForm[app.NodeStoreForwardSettings]{
		content: form,
		set: func(v app.NodeStoreForwardSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			heartbeat.SetChecked(v.Heartbeat)
			records.SetText(strconv.FormatUint(uint64(v.Records), 10))
			historyReturnMax.SetText(strconv.FormatUint(uint64(v.HistoryReturnMax), 10))
			historyReturnWindow.SetText(strconv.FormatUint(uint64(v.HistoryReturnWindow), 10))
			isServer.SetChecked(v.IsServer)
		},
		read: func(base app.NodeStoreForwardSettings, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.Heartbeat = heartbeat.Checked
			base.Records, err = parseOptionalUint32(records.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("records", err)
			}
			base.HistoryReturnMax, err = parseOptionalUint32(historyReturnMax.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("history return max", err)
			}
			base.HistoryReturnWindow, err = parseOptionalUint32(historyReturnWindow.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("history return window", err)
			}
			base.IsServer = isServer.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, heartbeat, records, historyReturnMax, historyReturnWindow, isServer),
	}
}
