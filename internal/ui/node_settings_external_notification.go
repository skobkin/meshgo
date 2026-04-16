package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func newNodeExternalNotificationSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.external_notification", "Loading external notification settings…", "External notification settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error) {
			return dep.Actions.NodeSettings.LoadExternalNotificationSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeExternalNotificationSettings) error {
			return dep.Actions.NodeSettings.SaveExternalNotificationSettings(ctx, target, settings)
		},
		func(v app.NodeExternalNotificationSettings) app.NodeExternalNotificationSettings { return v },
		func(v app.NodeExternalNotificationSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%t|%t|%t|%t|%t|%t|%t|%t|%d|%s|%t",
				v.NodeID, v.Enabled, v.OutputMS, v.OutputGPIO, v.OutputVibraGPIO, v.OutputBuzzerGPIO, v.OutputActiveHigh,
				v.AlertMessageLED, v.AlertMessageVibra, v.AlertMessageBuzzer, v.AlertBellLED, v.AlertBellVibra, v.AlertBellBuzzer,
				v.UsePWMBuzzer, v.NagTimeoutSecs, v.Ringtone, v.UseI2SAsBuzzer)
		},
		buildNodeExternalNotificationSettingsForm,
	)
}

func buildNodeExternalNotificationSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeExternalNotificationSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	outputMS := widget.NewSelect(nil, nil)
	outputMS.OnChanged = func(string) { onChanged() }
	outputGPIO := newNumberEntry(onChanged)
	outputVibraGPIO := newNumberEntry(onChanged)
	outputBuzzerGPIO := newNumberEntry(onChanged)
	outputActiveHigh := newSettingsCheck(onChanged)
	alertMessageLED := newSettingsCheck(onChanged)
	alertMessageVibra := newSettingsCheck(onChanged)
	alertMessageBuzzer := newSettingsCheck(onChanged)
	alertBellLED := newSettingsCheck(onChanged)
	alertBellVibra := newSettingsCheck(onChanged)
	alertBellBuzzer := newSettingsCheck(onChanged)
	usePWMBuzzer := newSettingsCheck(onChanged)
	nagTimeout := widget.NewSelect(nil, nil)
	nagTimeout.OnChanged = func(string) { onChanged() }
	ringtone := widget.NewMultiLineEntry()
	ringtone.SetMinRowsVisible(4)
	ringtone.OnChanged = func(string) { onChanged() }
	useI2S := newSettingsCheck(onChanged)

	configForm := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("External notification enabled", enabled),
	)
	messageForm := widget.NewForm(
		widget.NewFormItem("Alert message LED", alertMessageLED),
		widget.NewFormItem("Alert message buzzer", alertMessageBuzzer),
		widget.NewFormItem("Alert message vibra", alertMessageVibra),
	)
	bellForm := widget.NewForm(
		widget.NewFormItem("Alert bell LED", alertBellLED),
		widget.NewFormItem("Alert bell buzzer", alertBellBuzzer),
		widget.NewFormItem("Alert bell vibra", alertBellVibra),
	)
	activeHighItem := widget.NewForm(
		widget.NewFormItem("Output LED active high", outputActiveHigh),
	)
	pwmItem := widget.NewForm(
		widget.NewFormItem("Use PWM buzzer", usePWMBuzzer),
	)
	advancedForm := widget.NewForm(
		widget.NewFormItem("Output LED GPIO", outputGPIO),
		widget.NewFormItem("Output buzzer GPIO", outputBuzzerGPIO),
		widget.NewFormItem("Output vibra GPIO", outputVibraGPIO),
		widget.NewFormItem("Output duration milliseconds", outputMS),
		widget.NewFormItem("Nag timeout seconds", nagTimeout),
		widget.NewFormItem("Ringtone", ringtone),
		widget.NewFormItem("Use I2S as buzzer", useI2S),
	)
	updateAdvancedVisibility := func() {
		if strings.TrimSpace(outputGPIO.Text) != "" && strings.TrimSpace(outputGPIO.Text) != "0" {
			activeHighItem.Show()
		} else {
			activeHighItem.Hide()
		}
		if strings.TrimSpace(outputBuzzerGPIO.Text) != "" && strings.TrimSpace(outputBuzzerGPIO.Text) != "0" {
			pwmItem.Show()
		} else {
			pwmItem.Hide()
		}
	}
	outputGPIO.OnChanged = func(string) {
		updateAdvancedVisibility()
		onChanged()
	}
	outputBuzzerGPIO.OnChanged = func(string) {
		updateAdvancedVisibility()
		onChanged()
	}
	content := container.NewVBox(
		widget.NewCard("External notification config", "", configForm),
		widget.NewCard("Notifications on message receipt", "", messageForm),
		widget.NewCard("Notifications on alert bell receipt", "", bellForm),
		widget.NewCard("Advanced", "", container.NewVBox(
			advancedForm,
			activeHighItem,
			pwmItem,
		)),
	)
	updateAdvancedVisibility()

	return nodeManagedSettingsForm[app.NodeExternalNotificationSettings]{
		content: content,
		set: func(v app.NodeExternalNotificationSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(outputMS, nodeSettingsOutputDurationOptions, v.OutputMS, nodeSettingsCustomMillisecondsLabel)
			outputGPIO.SetText(strconv.FormatUint(uint64(v.OutputGPIO), 10))
			outputVibraGPIO.SetText(strconv.FormatUint(uint64(v.OutputVibraGPIO), 10))
			outputBuzzerGPIO.SetText(strconv.FormatUint(uint64(v.OutputBuzzerGPIO), 10))
			outputActiveHigh.SetChecked(v.OutputActiveHigh)
			alertMessageLED.SetChecked(v.AlertMessageLED)
			alertMessageVibra.SetChecked(v.AlertMessageVibra)
			alertMessageBuzzer.SetChecked(v.AlertMessageBuzzer)
			alertBellLED.SetChecked(v.AlertBellLED)
			alertBellVibra.SetChecked(v.AlertBellVibra)
			alertBellBuzzer.SetChecked(v.AlertBellBuzzer)
			usePWMBuzzer.SetChecked(v.UsePWMBuzzer)
			nodeSettingsSetUint32Select(nagTimeout, nodeSettingsNagTimeoutOptions, v.NagTimeoutSecs, nodeSettingsCustomSecondsLabel)
			ringtone.SetText(v.Ringtone)
			useI2S.SetChecked(v.UseI2SAsBuzzer)
			updateAdvancedVisibility()
		},
		read: func(base app.NodeExternalNotificationSettings, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.OutputMS, err = nodeSettingsParseMillisecondsSelectLabel("output ms", outputMS.Selected, nodeSettingsOutputDurationOptions)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("output ms", err)
			}
			base.OutputGPIO, err = parseOptionalUint32(outputGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("output GPIO", err)
			}
			base.OutputVibraGPIO, err = parseOptionalUint32(outputVibraGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("vibra GPIO", err)
			}
			base.OutputBuzzerGPIO, err = parseOptionalUint32(outputBuzzerGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("buzzer GPIO", err)
			}
			base.OutputActiveHigh = outputActiveHigh.Checked
			base.AlertMessageLED = alertMessageLED.Checked
			base.AlertMessageVibra = alertMessageVibra.Checked
			base.AlertMessageBuzzer = alertMessageBuzzer.Checked
			base.AlertBellLED = alertBellLED.Checked
			base.AlertBellVibra = alertBellVibra.Checked
			base.AlertBellBuzzer = alertBellBuzzer.Checked
			base.UsePWMBuzzer = usePWMBuzzer.Checked
			base.NagTimeoutSecs, err = nodeSettingsParseUint32SelectLabel("nag timeout", nagTimeout.Selected, nodeSettingsNagTimeoutOptions)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("nag timeout", err)
			}
			base.Ringtone = ringtone.Text
			base.UseI2SAsBuzzer = useI2S.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, outputMS, outputGPIO, outputVibraGPIO, outputBuzzerGPIO, outputActiveHigh, alertMessageLED, alertMessageVibra, alertMessageBuzzer, alertBellLED, alertBellVibra, alertBellBuzzer, usePWMBuzzer, nagTimeout, ringtone, useI2S),
	}
}
