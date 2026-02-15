package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func newNodeBluetoothSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "device.bluetooth"
	nodeSettingsTabLogger.Debug("building node bluetooth settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading bluetooth settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	enabledBox := widget.NewCheck("", nil)
	pairingModeSelect := widget.NewSelect(nil, nil)
	fixedPINEntry := widget.NewEntry()
	fixedPINEntry.SetPlaceHolder("123456")

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Bluetooth enabled", enabledBox),
		widget.NewFormItem("Pairing mode", pairingModeSelect),
		widget.NewFormItem("Fixed PIN", fixedPINEntry),
	)

	var (
		baseline             app.NodeBluetoothSettings
		baselineFormValues   nodeBluetoothSettingsFormValues
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

	setForm := func(settings app.NodeBluetoothSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		enabledBox.SetChecked(settings.Enabled)
		nodeBluetoothSetEnumSelect(pairingModeSelect, nodeBluetoothPairingModeOptions, settings.Mode)
		fixedPINEntry.SetText(strconv.FormatUint(uint64(settings.FixedPIN), 10))
	}

	applyForm := func(settings app.NodeBluetoothSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeBluetoothSettingsFormValues {
		return nodeBluetoothSettingsFormValues{
			Enabled:  enabledBox.Checked,
			Mode:     strings.TrimSpace(pairingModeSelect.Selected),
			FixedPIN: strings.TrimSpace(fixedPINEntry.Text),
		}
	}

	updateFieldAvailability := func() {
		mu.Lock()
		isSaving := saving
		mu.Unlock()

		mode, err := nodeBluetoothParseEnumLabel("pairing mode", pairingModeSelect.Selected, nodeBluetoothPairingModeOptions)
		if !isSaving && err == nil && mode == int32(generated.Config_BluetoothConfig_FIXED_PIN) {
			fixedPINEntry.Enable()
		} else {
			fixedPINEntry.Disable()
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
		updateFieldAvailability()
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

	applyLoadedSettings := func(next app.NodeBluetoothSettings) {
		mu.Lock()
		baseline = cloneNodeBluetoothSettings(next)
		baselineFormValues = nodeBluetoothFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeBluetoothSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeBluetoothSettings, error) {
		mode, err := nodeBluetoothParseEnumLabel("pairing mode", pairingModeSelect.Selected, nodeBluetoothPairingModeOptions)
		if err != nil {
			return app.NodeBluetoothSettings{}, err
		}
		fixedPIN, err := parseNodeBluetoothFixedPINField("fixed PIN", fixedPINEntry.Text)
		if err != nil {
			return app.NodeBluetoothSettings{}, err
		}
		if mode == int32(generated.Config_BluetoothConfig_FIXED_PIN) && fixedPIN == 0 {
			return app.NodeBluetoothSettings{}, fmt.Errorf("fixed PIN must be 6 digits when pairing mode is Fixed PIN")
		}

		mu.Lock()
		next := cloneNodeBluetoothSettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.Enabled = enabledBox.Checked
		next.Mode = mode
		next.FixedPIN = fixedPIN

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node bluetooth settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node bluetooth settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node bluetooth settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node bluetooth settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading bluetooth settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadBluetoothSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node bluetooth settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node bluetooth settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded bluetooth settings from device.", 2, 2)
			})
		}()
	}

	enabledBox.OnChanged = func(_ bool) { markDirty() }
	pairingModeSelect.OnChanged = func(_ string) {
		markDirty()
		updateFieldAvailability()
	}
	fixedPINEntry.OnChanged = func(_ string) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node bluetooth settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeBluetoothSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node bluetooth settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node bluetooth settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node bluetooth settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node bluetooth settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node bluetooth settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node bluetooth settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node bluetooth settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving bluetooth settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeBluetoothSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveBluetoothSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node bluetooth settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeBluetoothSettings(settings)
				baselineFormValues = nodeBluetoothFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeBluetoothSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node bluetooth settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved bluetooth settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node bluetooth settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeBluetoothSettings{
		Mode: int32(generated.Config_BluetoothConfig_RANDOM_PIN),
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Bluetooth settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Bluetooth settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node bluetooth settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node bluetooth settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Bluetooth settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node bluetooth settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeBluetoothSettingsFormValues struct {
	Enabled  bool
	Mode     string
	FixedPIN string
}

type nodeBluetoothEnumOption struct {
	Label string
	Value int32
}

var nodeBluetoothPairingModeOptions = []nodeBluetoothEnumOption{
	{Label: "Random PIN", Value: int32(generated.Config_BluetoothConfig_RANDOM_PIN)},
	{Label: "Fixed PIN", Value: int32(generated.Config_BluetoothConfig_FIXED_PIN)},
	{Label: "No PIN", Value: int32(generated.Config_BluetoothConfig_NO_PIN)},
}

func nodeBluetoothFormValuesFromSettings(settings app.NodeBluetoothSettings) nodeBluetoothSettingsFormValues {
	return nodeBluetoothSettingsFormValues{
		Enabled:  settings.Enabled,
		Mode:     nodeBluetoothEnumLabel(settings.Mode, nodeBluetoothPairingModeOptions),
		FixedPIN: strconv.FormatUint(uint64(settings.FixedPIN), 10),
	}
}

func cloneNodeBluetoothSettings(settings app.NodeBluetoothSettings) app.NodeBluetoothSettings {
	return settings
}

func parseNodeBluetoothFixedPINField(fieldName, raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a non-negative integer", fieldName)
	}
	if value == 0 {
		return 0, nil
	}
	if len(raw) != 6 || value > 999999 {
		return 0, fmt.Errorf("%s must be 6 digits", fieldName)
	}

	return uint32(value), nil
}

func nodeBluetoothSetEnumSelect(selectWidget *widget.Select, options []nodeBluetoothEnumOption, value int32) {
	optionLabels := nodeBluetoothEnumOptionsLabels(options)
	selected := nodeBluetoothEnumLabel(value, options)
	if strings.HasPrefix(selected, "Unknown (") {
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeBluetoothEnumLabel(value int32, options []nodeBluetoothEnumOption) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return nodeBluetoothUnknownEnumLabel(value)
}

func nodeBluetoothEnumOptionsLabels(options []nodeBluetoothEnumOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func nodeBluetoothUnknownEnumLabel(value int32) string {
	return fmt.Sprintf("Unknown (%d)", value)
}

func nodeBluetoothParseEnumLabel(fieldName, selected string, options []nodeBluetoothEnumOption) (int32, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return 0, fmt.Errorf("%s must be selected", fieldName)
	}
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}

	const (
		prefix = "Unknown ("
		suffix = ")"
	)
	if strings.HasPrefix(selected, prefix) && strings.HasSuffix(selected, suffix) {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, prefix), suffix)
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return int32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}
