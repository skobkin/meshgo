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

func newNodeCannedMessageSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.canned_message", "Loading canned message settings…", "Canned message settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error) {
			return dep.Actions.NodeSettings.LoadCannedMessageSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeCannedMessageSettings) error {
			return dep.Actions.NodeSettings.SaveCannedMessageSettings(ctx, target, settings)
		},
		func(v app.NodeCannedMessageSettings) app.NodeCannedMessageSettings { return v },
		func(v app.NodeCannedMessageSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%d|%d|%t|%t|%s|%t|%s",
				v.NodeID, v.Rotary1Enabled, v.InputBrokerPinA, v.InputBrokerPinB, v.InputBrokerPinPress,
				v.InputBrokerEventCW, v.InputBrokerEventCCW, v.InputBrokerEventPress, v.UpDown1Enabled,
				v.Enabled, v.AllowInputSource, v.SendBell, v.Messages)
		},
		buildNodeCannedMessageSettingsForm,
	)
}

func buildNodeCannedMessageSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeCannedMessageSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	rotary1Enabled := newSettingsCheck(onChanged)
	pinA := newNumberEntry(onChanged)
	pinB := newNumberEntry(onChanged)
	pinPress := newNumberEntry(onChanged)
	eventCW := newNumberEntry(onChanged)
	eventCCW := newNumberEntry(onChanged)
	eventPress := newNumberEntry(onChanged)
	upDown1Enabled := newSettingsCheck(onChanged)
	enabled := newSettingsCheck(onChanged)
	allowInputSource := widget.NewEntry()
	allowInputSource.OnChanged = func(string) { onChanged() }
	sendBell := newSettingsCheck(onChanged)
	messages := widget.NewMultiLineEntry()
	messages.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Rotary 1 enabled", rotary1Enabled),
		widget.NewFormItem("Input broker pin A", pinA),
		widget.NewFormItem("Input broker pin B", pinB),
		widget.NewFormItem("Input broker pin press", pinPress),
		widget.NewFormItem("Input broker event CW", eventCW),
		widget.NewFormItem("Input broker event CCW", eventCCW),
		widget.NewFormItem("Input broker event press", eventPress),
		widget.NewFormItem("Up/down 1 enabled", upDown1Enabled),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Allow input source", allowInputSource),
		widget.NewFormItem("Send bell", sendBell),
		widget.NewFormItem("Messages", messages),
	)

	return nodeManagedSettingsForm[app.NodeCannedMessageSettings]{
		content: form,
		set: func(v app.NodeCannedMessageSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			rotary1Enabled.SetChecked(v.Rotary1Enabled)
			pinA.SetText(strconv.FormatUint(uint64(v.InputBrokerPinA), 10))
			pinB.SetText(strconv.FormatUint(uint64(v.InputBrokerPinB), 10))
			pinPress.SetText(strconv.FormatUint(uint64(v.InputBrokerPinPress), 10))
			eventCW.SetText(strconv.FormatInt(int64(v.InputBrokerEventCW), 10))
			eventCCW.SetText(strconv.FormatInt(int64(v.InputBrokerEventCCW), 10))
			eventPress.SetText(strconv.FormatInt(int64(v.InputBrokerEventPress), 10))
			upDown1Enabled.SetChecked(v.UpDown1Enabled)
			enabled.SetChecked(v.Enabled)
			allowInputSource.SetText(v.AllowInputSource)
			sendBell.SetChecked(v.SendBell)
			messages.SetText(v.Messages)
		},
		read: func(base app.NodeCannedMessageSettings, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Rotary1Enabled = rotary1Enabled.Checked
			base.InputBrokerPinA, err = parseOptionalUint32(pinA.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin A", err)
			}
			base.InputBrokerPinB, err = parseOptionalUint32(pinB.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin B", err)
			}
			base.InputBrokerPinPress, err = parseOptionalUint32(pinPress.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin press", err)
			}
			base.InputBrokerEventCW, err = parseOptionalInt32(eventCW.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event CW", err)
			}
			base.InputBrokerEventCCW, err = parseOptionalInt32(eventCCW.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event CCW", err)
			}
			base.InputBrokerEventPress, err = parseOptionalInt32(eventPress.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event press", err)
			}
			base.UpDown1Enabled = upDown1Enabled.Checked
			base.Enabled = enabled.Checked
			base.AllowInputSource = strings.TrimSpace(allowInputSource.Text)
			base.SendBell = sendBell.Checked
			base.Messages = messages.Text

			return base, nil
		},
		setSaving: disableWidgets(rotary1Enabled, pinA, pinB, pinPress, eventCW, eventCCW, eventPress, upDown1Enabled, enabled, allowInputSource, sendBell, messages),
	}
}
