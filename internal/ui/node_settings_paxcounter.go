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

func newNodePaxcounterSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.paxcounter", "Loading paxcounter settings…", "Paxcounter settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error) {
			return dep.Actions.NodeSettings.LoadPaxcounterSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodePaxcounterSettings) error {
			return dep.Actions.NodeSettings.SavePaxcounterSettings(ctx, target, settings)
		},
		func(v app.NodePaxcounterSettings) app.NodePaxcounterSettings { return v },
		func(v app.NodePaxcounterSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d", v.NodeID, v.Enabled, v.UpdateIntervalSecs, v.WifiThreshold, v.BLEThreshold)
		},
		buildNodePaxcounterSettingsForm,
	)
}

func buildNodePaxcounterSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodePaxcounterSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	updateInterval := widget.NewSelect(nil, nil)
	updateInterval.OnChanged = func(string) { onChanged() }
	wifiThreshold := newNumberEntry(onChanged)
	bleThreshold := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Update interval secs", updateInterval),
		widget.NewFormItem("WiFi threshold", wifiThreshold),
		widget.NewFormItem("BLE threshold", bleThreshold),
	)

	return nodeManagedSettingsForm[app.NodePaxcounterSettings]{
		content: form,
		set: func(v app.NodePaxcounterSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(updateInterval, nodeSettingsPaxcounterIntervalOptions, v.UpdateIntervalSecs, nodeSettingsCustomSecondsLabel)
			wifiThreshold.SetText(strconv.FormatInt(int64(v.WifiThreshold), 10))
			bleThreshold.SetText(strconv.FormatInt(int64(v.BLEThreshold), 10))
		},
		read: func(base app.NodePaxcounterSettings, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.UpdateIntervalSecs, err = nodeSettingsParseUint32SelectLabel("update interval secs", updateInterval.Selected, nodeSettingsPaxcounterIntervalOptions)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("update interval secs", err)
			}
			base.WifiThreshold, err = parseOptionalInt32(wifiThreshold.Text)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("WiFi threshold", err)
			}
			base.BLEThreshold, err = parseOptionalInt32(bleThreshold.Text)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("BLE threshold", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(enabled, updateInterval, wifiThreshold, bleThreshold),
	}
}
