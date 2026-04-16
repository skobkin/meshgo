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

func newNodeDetectionSensorSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.detection_sensor", "Loading detection sensor settings…", "Detection sensor settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error) {
			return dep.Actions.NodeSettings.LoadDetectionSensorSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeDetectionSensorSettings) error {
			return dep.Actions.NodeSettings.SaveDetectionSensorSettings(ctx, target, settings)
		},
		func(v app.NodeDetectionSensorSettings) app.NodeDetectionSensorSettings { return v },
		func(v app.NodeDetectionSensorSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%t|%s|%d|%d|%t", v.NodeID, v.Enabled, v.MinimumBroadcastSecs, v.StateBroadcastSecs, v.SendBell, v.Name, v.MonitorPin, v.DetectionTriggerType, v.UsePullup)
		},
		buildNodeDetectionSensorSettingsForm,
	)
}

func buildNodeDetectionSensorSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeDetectionSensorSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	minimumBroadcast := widget.NewSelect(nil, nil)
	minimumBroadcast.OnChanged = func(string) { onChanged() }
	stateBroadcast := widget.NewSelect(nil, nil)
	stateBroadcast.OnChanged = func(string) { onChanged() }
	sendBell := newSettingsCheck(onChanged)
	name := widget.NewEntry()
	name.OnChanged = func(string) { onChanged() }
	monitorPin := newNumberEntry(onChanged)
	triggerType := widget.NewSelect(nil, nil)
	triggerType.OnChanged = func(string) { onChanged() }
	usePullup := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Minimum broadcast secs", minimumBroadcast),
		widget.NewFormItem("State broadcast secs", stateBroadcast),
		widget.NewFormItem("Send bell", sendBell),
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Monitor pin", monitorPin),
		widget.NewFormItem("Detection trigger type", triggerType),
		widget.NewFormItem("Use pullup", usePullup),
	)

	return nodeManagedSettingsForm[app.NodeDetectionSensorSettings]{
		content: form,
		set: func(v app.NodeDetectionSensorSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(minimumBroadcast, nodeSettingsDetectionMinimumIntervalOptions, v.MinimumBroadcastSecs, nodeSettingsCustomSecondsLabel)
			nodeSettingsSetUint32Select(stateBroadcast, nodeSettingsDetectionStateIntervalOptions, v.StateBroadcastSecs, nodeSettingsCustomSecondsLabel)
			sendBell.SetChecked(v.SendBell)
			name.SetText(v.Name)
			monitorPin.SetText(strconv.FormatUint(uint64(v.MonitorPin), 10))
			nodeSettingsSetInt32Select(triggerType, nodeSettingsDetectionTriggerTypeOptions, v.DetectionTriggerType, nodeSettingsCustomInt32Label)
			usePullup.SetChecked(v.UsePullup)
		},
		read: func(base app.NodeDetectionSensorSettings, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.MinimumBroadcastSecs, err = nodeSettingsParseUint32SelectLabel("minimum broadcast secs", minimumBroadcast.Selected, nodeSettingsDetectionMinimumIntervalOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("minimum broadcast secs", err)
			}
			base.StateBroadcastSecs, err = nodeSettingsParseUint32SelectLabel("state broadcast secs", stateBroadcast.Selected, nodeSettingsDetectionStateIntervalOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("state broadcast secs", err)
			}
			base.SendBell = sendBell.Checked
			base.Name = strings.TrimSpace(name.Text)
			base.MonitorPin, err = parseOptionalUint32(monitorPin.Text)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("monitor pin", err)
			}
			base.DetectionTriggerType, err = nodeSettingsParseInt32SelectLabel("detection trigger type", triggerType.Selected, nodeSettingsDetectionTriggerTypeOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("detection trigger type", err)
			}
			base.UsePullup = usePullup.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, minimumBroadcast, stateBroadcast, sendBell, name, monitorPin, triggerType, usePullup),
	}
}
