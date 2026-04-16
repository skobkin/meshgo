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

func newNodeAmbientLightingSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.ambient_lighting", "Loading ambient lighting settings…", "Ambient lighting settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
			return dep.Actions.NodeSettings.LoadAmbientLightingSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAmbientLightingSettings) error {
			return dep.Actions.NodeSettings.SaveAmbientLightingSettings(ctx, target, settings)
		},
		func(v app.NodeAmbientLightingSettings) app.NodeAmbientLightingSettings { return v },
		func(v app.NodeAmbientLightingSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d", v.NodeID, v.LEDState, v.Current, v.Red, v.Green, v.Blue)
		},
		buildNodeAmbientLightingSettingsForm,
	)
}

func buildNodeAmbientLightingSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeAmbientLightingSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	ledState := newSettingsCheck(onChanged)
	current := newNumberEntry(onChanged)
	red := newNumberEntry(onChanged)
	green := newNumberEntry(onChanged)
	blue := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("LED state", ledState),
		widget.NewFormItem("Current", current),
		widget.NewFormItem("Red", red),
		widget.NewFormItem("Green", green),
		widget.NewFormItem("Blue", blue),
	)

	return nodeManagedSettingsForm[app.NodeAmbientLightingSettings]{
		content: form,
		set: func(v app.NodeAmbientLightingSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			ledState.SetChecked(v.LEDState)
			current.SetText(strconv.FormatUint(uint64(v.Current), 10))
			red.SetText(strconv.FormatUint(uint64(v.Red), 10))
			green.SetText(strconv.FormatUint(uint64(v.Green), 10))
			blue.SetText(strconv.FormatUint(uint64(v.Blue), 10))
		},
		read: func(base app.NodeAmbientLightingSettings, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.LEDState = ledState.Checked
			base.Current, err = parseOptionalUint32(current.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("current", err)
			}
			base.Red, err = parseOptionalUint32(red.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("red", err)
			}
			base.Green, err = parseOptionalUint32(green.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("green", err)
			}
			base.Blue, err = parseOptionalUint32(blue.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("blue", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(ledState, current, red, green, blue),
	}
}
