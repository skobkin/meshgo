package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func newNodeNeighborInfoSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.neighbor_info", "Loading neighbor info settings…", "Neighbor info settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
			return dep.Actions.NodeSettings.LoadNeighborInfoSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNeighborInfoSettings) error {
			return dep.Actions.NodeSettings.SaveNeighborInfoSettings(ctx, target, settings)
		},
		func(v app.NodeNeighborInfoSettings) app.NodeNeighborInfoSettings { return v },
		func(v app.NodeNeighborInfoSettings) string {
			return fmt.Sprintf("%s|%t|%d|%t", v.NodeID, v.Enabled, v.UpdateIntervalSecs, v.TransmitOverLoRa)
		},
		buildNodeNeighborInfoSettingsForm,
	)
}

func buildNodeNeighborInfoSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeNeighborInfoSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	updateInterval := widget.NewSelect(nil, nil)
	updateInterval.OnChanged = func(string) { onChanged() }
	transmitOverLoRa := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Update interval secs", updateInterval),
		widget.NewFormItem("Transmit over LoRa", transmitOverLoRa),
	)

	return nodeManagedSettingsForm[app.NodeNeighborInfoSettings]{
		content: form,
		set: func(v app.NodeNeighborInfoSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(updateInterval, nodeSettingsDetectionMinimumIntervalOptions, v.UpdateIntervalSecs, nodeSettingsCustomSecondsLabel)
			transmitOverLoRa.SetChecked(v.TransmitOverLoRa)
		},
		read: func(base app.NodeNeighborInfoSettings, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.UpdateIntervalSecs, err = nodeSettingsParseUint32SelectLabel("update interval secs", updateInterval.Selected, nodeSettingsDetectionMinimumIntervalOptions)
			if err != nil {
				return app.NodeNeighborInfoSettings{}, fieldParseError("update interval secs", err)
			}
			base.TransmitOverLoRa = transmitOverLoRa.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, updateInterval, transmitOverLoRa),
	}
}
