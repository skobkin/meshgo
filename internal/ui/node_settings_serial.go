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

func newNodeSerialSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.serial", "Loading serial settings…", "Serial settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeSerialSettings, error) {
			return dep.Actions.NodeSettings.LoadSerialSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeSerialSettings) error {
			return dep.Actions.NodeSettings.SaveSerialSettings(ctx, target, settings)
		},
		func(v app.NodeSerialSettings) app.NodeSerialSettings { return v },
		func(v app.NodeSerialSettings) string {
			return fmt.Sprintf("%s|%t|%t|%d|%d|%d|%d|%d|%t", v.NodeID, v.Enabled, v.EchoEnabled, v.RXGPIO, v.TXGPIO, v.Baud, v.Timeout, v.Mode, v.OverrideConsoleSerialPort)
		},
		buildNodeSerialSettingsForm,
	)
}

func buildNodeSerialSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeSerialSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	echoEnabled := newSettingsCheck(onChanged)
	rxGPIO := newNumberEntry(onChanged)
	txGPIO := newNumberEntry(onChanged)
	baud := widget.NewSelect(nil, nil)
	baud.OnChanged = func(string) { onChanged() }
	timeout := newNumberEntry(onChanged)
	mode := widget.NewSelect(nil, nil)
	mode.OnChanged = func(string) { onChanged() }
	overrideConsole := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Echo enabled", echoEnabled),
		widget.NewFormItem("RX GPIO", rxGPIO),
		widget.NewFormItem("TX GPIO", txGPIO),
		widget.NewFormItem("Baud", baud),
		widget.NewFormItem("Timeout", timeout),
		widget.NewFormItem("Mode", mode),
		widget.NewFormItem("Override console serial port", overrideConsole),
	)

	return nodeManagedSettingsForm[app.NodeSerialSettings]{
		content: form,
		set: func(v app.NodeSerialSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			echoEnabled.SetChecked(v.EchoEnabled)
			rxGPIO.SetText(strconv.FormatUint(uint64(v.RXGPIO), 10))
			txGPIO.SetText(strconv.FormatUint(uint64(v.TXGPIO), 10))
			nodeSettingsSetInt32Select(baud, nodeSettingsSerialBaudOptions, v.Baud, nodeSettingsCustomInt32Label)
			timeout.SetText(strconv.FormatUint(uint64(v.Timeout), 10))
			nodeSettingsSetInt32Select(mode, nodeSettingsSerialModeOptions, v.Mode, nodeSettingsCustomInt32Label)
			overrideConsole.SetChecked(v.OverrideConsoleSerialPort)
		},
		read: func(base app.NodeSerialSettings, target app.NodeSettingsTarget) (app.NodeSerialSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.EchoEnabled = echoEnabled.Checked
			base.RXGPIO, err = parseOptionalUint32(rxGPIO.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("RX GPIO", err)
			}
			base.TXGPIO, err = parseOptionalUint32(txGPIO.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("TX GPIO", err)
			}
			base.Baud, err = nodeSettingsParseInt32SelectLabel("baud", baud.Selected, nodeSettingsSerialBaudOptions)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("baud", err)
			}
			base.Timeout, err = parseOptionalUint32(timeout.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("timeout", err)
			}
			base.Mode, err = nodeSettingsParseInt32SelectLabel("mode", mode.Selected, nodeSettingsSerialModeOptions)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("mode", err)
			}
			base.OverrideConsoleSerialPort = overrideConsole.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, echoEnabled, rxGPIO, txGPIO, baud, timeout, mode, overrideConsole),
	}
}
