package ui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func newNodeStatusMessageSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.status_message", "Loading status message settings…", "Status message settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error) {
			return dep.Actions.NodeSettings.LoadStatusMessageSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStatusMessageSettings) error {
			return dep.Actions.NodeSettings.SaveStatusMessageSettings(ctx, target, settings)
		},
		func(v app.NodeStatusMessageSettings) app.NodeStatusMessageSettings { return v },
		func(v app.NodeStatusMessageSettings) string { return v.NodeID + "|" + v.NodeStatus },
		buildNodeStatusMessageSettingsForm,
	)
}

func buildNodeStatusMessageSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeStatusMessageSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewMultiLineEntry()
	status.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Status", status),
	)

	return nodeManagedSettingsForm[app.NodeStatusMessageSettings]{
		content: form,
		set: func(v app.NodeStatusMessageSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			status.SetText(v.NodeStatus)
		},
		read: func(base app.NodeStatusMessageSettings, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.NodeStatus = status.Text

			return base, nil
		},
		setSaving: disableWidgets(status),
	}
}
