package ui

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

const (
	nodeRangeTestSenderIntervalUnset     uint32 = 0    // unset
	nodeRangeTestSenderInterval15Seconds uint32 = 15   // 15 seconds
	nodeRangeTestSenderInterval30Seconds uint32 = 30   // 30 seconds
	nodeRangeTestSenderInterval45Seconds uint32 = 45   // 45 seconds
	nodeRangeTestSenderInterval1Minute   uint32 = 60   // 1 minute
	nodeRangeTestSenderInterval5Minutes  uint32 = 300  // 5 minutes
	nodeRangeTestSenderInterval10Minutes uint32 = 600  // 10 minutes
	nodeRangeTestSenderInterval15Minutes uint32 = 900  // 15 minutes
	nodeRangeTestSenderInterval30Minutes uint32 = 1800 // 30 minutes
	nodeRangeTestSenderInterval1Hour     uint32 = 3600 // 1 hour
)

func newNodeRangeTestSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "module.range_test"
	nodeSettingsTabLogger.Debug("building node range test settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading range test settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	enabledBox := widget.NewCheck("", nil)
	senderIntervalSelect := widget.NewSelect(nil, nil)
	saveCSVBox := widget.NewCheck("", nil)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Range test enabled", enabledBox),
		widget.NewFormItem("Sender message interval", senderIntervalSelect),
		widget.NewFormItem("Save CSV in storage (ESP32 only)", saveCSVBox),
	)

	var (
		baseline             app.NodeRangeTestSettings
		baselineFormValues   nodeRangeTestSettingsFormValues
		dirty                bool
		saving               bool
		initialReloadStarted atomic.Bool
		mu                   sync.Mutex
		applyingForm         atomic.Bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}

	setForm := func(settings app.NodeRangeTestSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		enabledBox.SetChecked(settings.Enabled)
		nodeRangeTestSetSenderIntervalSelect(senderIntervalSelect, settings.Sender)
		saveCSVBox.SetChecked(settings.Save)
	}

	applyForm := func(settings app.NodeRangeTestSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeRangeTestSettingsFormValues {
		return nodeRangeTestSettingsFormValues{
			Enabled: enabledBox.Checked,
			Sender:  strings.TrimSpace(senderIntervalSelect.Selected),
			Save:    saveCSVBox.Checked,
		}
	}

	updateButtons := func() {
		mu.Lock()
		activePage := ""
		if saveGate != nil {
			activePage = strings.TrimSpace(saveGate.ActivePage())
		}
		connected := isConnected()
		canSave := dep.Actions.NodeSettings != nil && connected && !saving && dirty && (activePage == "" || activePage == pageID)
		canCancel := !saving && dirty
		canReload := dep.Actions.NodeSettings != nil && !saving
		mu.Unlock()

		if canSave {
			saveButton.Enable()
		} else {
			saveButton.Disable()
		}
		if canCancel {
			cancelButton.Enable()
		} else {
			cancelButton.Disable()
		}
		if canReload {
			reloadButton.Enable()
		} else {
			reloadButton.Disable()
		}
	}

	markDirty := func() {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		dirty = readFormValues() != baselineFormValues
		mu.Unlock()
		updateButtons()
	}

	applyLoadedSettings := func(next app.NodeRangeTestSettings) {
		mu.Lock()
		baseline = cloneNodeRangeTestSettings(next)
		baselineFormValues = nodeRangeTestFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeRangeTestSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeRangeTestSettings, error) {
		sender, err := nodeRangeTestParseSenderIntervalLabel("sender message interval", senderIntervalSelect.Selected)
		if err != nil {
			return app.NodeRangeTestSettings{}, err
		}

		mu.Lock()
		next := cloneNodeRangeTestSettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.Enabled = enabledBox.Checked
		next.Sender = sender
		next.Save = saveCSVBox.Checked

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node range test settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node range test settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node range test settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node range test settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading range test settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadRangeTestSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node range test settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node range test settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded range test settings from device.", 2, 2)
			})
		}()
	}

	enabledBox.OnChanged = func(_ bool) { markDirty() }
	senderIntervalSelect.OnChanged = func(_ string) { markDirty() }
	saveCSVBox.OnChanged = func(_ bool) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node range test settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeRangeTestSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node range test settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node range test settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node range test settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node range test settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node range test settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node range test settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node range test settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving range test settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeRangeTestSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveRangeTestSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node range test settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeRangeTestSettings(settings)
				baselineFormValues = nodeRangeTestFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeRangeTestSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node range test settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved range test settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node range test settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeRangeTestSettings{}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Range test settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Range test settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node range test settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node range test settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Range test module settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node range test settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeRangeTestSettingsFormValues struct {
	Enabled bool
	Sender  string
	Save    bool
}

var nodeRangeTestSenderIntervalOptions = []nodeSettingsUint32Option{
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderIntervalUnset, "Unset"), Value: nodeRangeTestSenderIntervalUnset},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval15Seconds, "Unset"), Value: nodeRangeTestSenderInterval15Seconds},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval30Seconds, "Unset"), Value: nodeRangeTestSenderInterval30Seconds},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval45Seconds, "Unset"), Value: nodeRangeTestSenderInterval45Seconds},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval1Minute, "Unset"), Value: nodeRangeTestSenderInterval1Minute},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval5Minutes, "Unset"), Value: nodeRangeTestSenderInterval5Minutes},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval10Minutes, "Unset"), Value: nodeRangeTestSenderInterval10Minutes},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval15Minutes, "Unset"), Value: nodeRangeTestSenderInterval15Minutes},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval30Minutes, "Unset"), Value: nodeRangeTestSenderInterval30Minutes},
	{Label: nodeSettingsSecondsKnownLabel(nodeRangeTestSenderInterval1Hour, "Unset"), Value: nodeRangeTestSenderInterval1Hour},
}

func nodeRangeTestSetSenderIntervalSelect(selectWidget *widget.Select, value uint32) {
	nodeSettingsSetUint32Select(selectWidget, nodeRangeTestSenderIntervalOptions, value, nodeSettingsCustomSecondsLabel)
}

func nodeRangeTestSenderIntervalSelectLabel(value uint32) string {
	label := nodeSettingsUint32OptionLabel(value, nodeRangeTestSenderIntervalOptions)
	if label != "" {
		return label
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodeRangeTestParseSenderIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeSettingsParseUint32SelectLabel(
		fieldName,
		selected,
		nodeRangeTestSenderIntervalOptions,
		nodeSettingsCustomSecondsLabelSuffix,
	)
}

func nodeRangeTestFormValuesFromSettings(settings app.NodeRangeTestSettings) nodeRangeTestSettingsFormValues {
	return nodeRangeTestSettingsFormValues{
		Enabled: settings.Enabled,
		Sender:  nodeRangeTestSenderIntervalSelectLabel(settings.Sender),
		Save:    settings.Save,
	}
}

func cloneNodeRangeTestSettings(settings app.NodeRangeTestSettings) app.NodeRangeTestSettings {
	return settings
}
