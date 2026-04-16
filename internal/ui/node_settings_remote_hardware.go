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

func newNodeRemoteHardwareSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.remote_hardware", "Loading remote hardware settings…", "Remote hardware settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error) {
			return dep.Actions.NodeSettings.LoadRemoteHardwareSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeRemoteHardwareSettings) error {
			return dep.Actions.NodeSettings.SaveRemoteHardwareSettings(ctx, target, settings)
		},
		cloneNodeRemoteHardwareSettings,
		func(v app.NodeRemoteHardwareSettings) string {
			parts := make([]string, 0, len(v.AvailablePins))
			for _, pin := range v.AvailablePins {
				parts = append(parts, strconv.FormatUint(uint64(pin), 10))
			}

			return fmt.Sprintf("%s|%t|%t|%s", v.NodeID, v.Enabled, v.AllowUndefinedPinAccess, strings.Join(parts, ","))
		},
		buildNodeRemoteHardwareSettingsForm,
	)
}

func buildNodeRemoteHardwareSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeRemoteHardwareSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	allowUndefined := newSettingsCheck(onChanged)
	availablePins := widget.NewEntry()
	availablePins.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Allow undefined pin access", allowUndefined),
		widget.NewFormItem("Available pins", availablePins),
	)

	return nodeManagedSettingsForm[app.NodeRemoteHardwareSettings]{
		content: form,
		set: func(v app.NodeRemoteHardwareSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			allowUndefined.SetChecked(v.AllowUndefinedPinAccess)
			parts := make([]string, 0, len(v.AvailablePins))
			for _, pin := range v.AvailablePins {
				parts = append(parts, strconv.FormatUint(uint64(pin), 10))
			}
			availablePins.SetText(strings.Join(parts, ", "))
		},
		read: func(base app.NodeRemoteHardwareSettings, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.Enabled = enabled.Checked
			base.AllowUndefinedPinAccess = allowUndefined.Checked
			pins, err := parseUint32List(availablePins.Text)
			if err != nil {
				return app.NodeRemoteHardwareSettings{}, fieldParseError("available pins", err)
			}
			base.AvailablePins = pins

			return base, nil
		},
		setSaving: disableWidgets(enabled, allowUndefined, availablePins),
	}
}
