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

func newNodeAudioSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.audio", "Loading audio settings…", "Audio settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAudioSettings, error) {
			return dep.Actions.NodeSettings.LoadAudioSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAudioSettings) error {
			return dep.Actions.NodeSettings.SaveAudioSettings(ctx, target, settings)
		},
		func(v app.NodeAudioSettings) app.NodeAudioSettings { return v },
		func(v app.NodeAudioSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%d|%d", v.NodeID, v.Codec2Enabled, v.PTTPin, v.Bitrate, v.I2SWordSelect, v.I2SDataIn, v.I2SDataOut, v.I2SClock)
		},
		buildNodeAudioSettingsForm,
	)
}

func buildNodeAudioSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeAudioSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	codec2Enabled := newSettingsCheck(onChanged)
	pttPin := newNumberEntry(onChanged)
	bitrate := widget.NewSelect(nil, nil)
	bitrate.OnChanged = func(string) { onChanged() }
	i2sWs := newNumberEntry(onChanged)
	i2sSd := newNumberEntry(onChanged)
	i2sDin := newNumberEntry(onChanged)
	i2sSck := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Codec2 enabled", codec2Enabled),
		widget.NewFormItem("PTT pin", pttPin),
		widget.NewFormItem("Bitrate", bitrate),
		widget.NewFormItem("I2S WS", i2sWs),
		widget.NewFormItem("I2S SD", i2sSd),
		widget.NewFormItem("I2S DIN", i2sDin),
		widget.NewFormItem("I2S SCK", i2sSck),
	)

	return nodeManagedSettingsForm[app.NodeAudioSettings]{
		content: form,
		set: func(v app.NodeAudioSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			codec2Enabled.SetChecked(v.Codec2Enabled)
			pttPin.SetText(strconv.FormatUint(uint64(v.PTTPin), 10))
			nodeSettingsSetInt32Select(bitrate, nodeSettingsAudioBitrateOptions, v.Bitrate, nodeSettingsCustomInt32Label)
			i2sWs.SetText(strconv.FormatUint(uint64(v.I2SWordSelect), 10))
			i2sSd.SetText(strconv.FormatUint(uint64(v.I2SDataIn), 10))
			i2sDin.SetText(strconv.FormatUint(uint64(v.I2SDataOut), 10))
			i2sSck.SetText(strconv.FormatUint(uint64(v.I2SClock), 10))
		},
		read: func(base app.NodeAudioSettings, target app.NodeSettingsTarget) (app.NodeAudioSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Codec2Enabled = codec2Enabled.Checked
			base.PTTPin, err = parseOptionalUint32(pttPin.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("PTT pin", err)
			}
			base.Bitrate, err = nodeSettingsParseInt32SelectLabel("bitrate", bitrate.Selected, nodeSettingsAudioBitrateOptions)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("bitrate", err)
			}
			base.I2SWordSelect, err = parseOptionalUint32(i2sWs.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S WS", err)
			}
			base.I2SDataIn, err = parseOptionalUint32(i2sSd.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S SD", err)
			}
			base.I2SDataOut, err = parseOptionalUint32(i2sDin.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S DIN", err)
			}
			base.I2SClock, err = parseOptionalUint32(i2sSck.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S SCK", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(codec2Enabled, pttPin, bitrate, i2sWs, i2sSd, i2sDin, i2sSck),
	}
}
