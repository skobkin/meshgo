package ui

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
)

func newNodePowerSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "device.power"
	nodeSettingsTabLogger.Debug("building node power settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading power settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	powerSavingBox := widget.NewCheck("", nil)
	shutdownOnPowerLossSelect := widget.NewSelect(nil, nil)
	adcMultiplierOverrideBox := widget.NewCheck("", nil)
	adcMultiplierOverrideEntry := widget.NewEntry()
	waitBluetoothSecsSelect := widget.NewSelect(nil, nil)
	superDeepSleepSecsSelect := widget.NewSelect(nil, nil)
	minWakeSecsSelect := widget.NewSelect(nil, nil)
	deviceBatteryINAAddressEntry := widget.NewEntry()

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Enable power saving mode", powerSavingBox),
		widget.NewFormItem("Shutdown on power loss", shutdownOnPowerLossSelect),
		widget.NewFormItem("ADC multiplier override", adcMultiplierOverrideBox),
		widget.NewFormItem("ADC multiplier override ratio", adcMultiplierOverrideEntry),
		widget.NewFormItem("Wait for Bluetooth duration", waitBluetoothSecsSelect),
		widget.NewFormItem("Super deep sleep duration", superDeepSleepSecsSelect),
		widget.NewFormItem("Minimum wake time", minWakeSecsSelect),
		widget.NewFormItem("Battery INA 2xx I2C address", deviceBatteryINAAddressEntry),
	)

	var (
		baseline             app.NodePowerSettings
		baselineFormValues   nodePowerSettingsFormValues
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

	setForm := func(settings app.NodePowerSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		powerSavingBox.SetChecked(settings.IsPowerSaving)
		nodeSettingsSetUint32Select(
			shutdownOnPowerLossSelect,
			nodePowerAllIntervalOptions,
			settings.OnBatteryShutdownAfterSecs,
			nodePowerAllIntervalCustomLabel,
		)
		overrideEnabled := settings.AdcMultiplierOverride > 0
		adcMultiplierOverrideBox.SetChecked(overrideEnabled)
		if overrideEnabled {
			adcMultiplierOverrideEntry.SetText(strconv.FormatFloat(float64(settings.AdcMultiplierOverride), 'f', -1, 32))
		} else {
			adcMultiplierOverrideEntry.SetText("1")
		}
		nodeSettingsSetUint32Select(
			waitBluetoothSecsSelect,
			nodeSettingsNagTimeoutOptions,
			settings.WaitBluetoothSecs,
			nodeSettingsCustomSecondsLabel,
		)
		nodeSettingsSetUint32Select(
			superDeepSleepSecsSelect,
			nodePowerAllIntervalOptions,
			settings.SdsSecs,
			nodePowerAllIntervalCustomLabel,
		)
		nodeSettingsSetUint32Select(
			minWakeSecsSelect,
			nodeSettingsNagTimeoutOptions,
			settings.MinWakeSecs,
			nodeSettingsCustomSecondsLabel,
		)
		deviceBatteryINAAddressEntry.SetText(strconv.FormatUint(uint64(settings.DeviceBatteryInaAddress), 10))
	}

	applyForm := func(settings app.NodePowerSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodePowerSettingsFormValues {
		return nodePowerSettingsFormValues{
			IsPowerSaving:              powerSavingBox.Checked,
			OnBatteryShutdownAfterSecs: strings.TrimSpace(shutdownOnPowerLossSelect.Selected),
			AdcMultiplierOverride:      strings.TrimSpace(adcMultiplierOverrideEntry.Text),
			AdcMultiplierOverrideOn:    adcMultiplierOverrideBox.Checked,
			WaitBluetoothSecs:          strings.TrimSpace(waitBluetoothSecsSelect.Selected),
			SdsSecs:                    strings.TrimSpace(superDeepSleepSecsSelect.Selected),
			MinWakeSecs:                strings.TrimSpace(minWakeSecsSelect.Selected),
			DeviceBatteryInaAddress:    strings.TrimSpace(deviceBatteryINAAddressEntry.Text),
		}
	}

	updateFieldAvailability := func() {
		mu.Lock()
		isSaving := saving
		mu.Unlock()

		if !isSaving && adcMultiplierOverrideBox.Checked {
			adcMultiplierOverrideEntry.Enable()
		} else {
			adcMultiplierOverrideEntry.Disable()
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

	applyLoadedSettings := func(next app.NodePowerSettings) {
		mu.Lock()
		baseline = cloneNodePowerSettings(next)
		baselineFormValues = nodePowerFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodePowerSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodePowerSettings, error) {
		onBatteryShutdownAfterSecs, err := nodePowerParseAllIntervalSelectLabel("shutdown on power loss", shutdownOnPowerLossSelect.Selected)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		waitBluetoothSecs, err := nodeSettingsParseUint32SelectLabel(
			"wait for Bluetooth duration",
			waitBluetoothSecsSelect.Selected,
			nodeSettingsNagTimeoutOptions,
		)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		sdsSecs, err := nodePowerParseAllIntervalSelectLabel("super deep sleep duration", superDeepSleepSecsSelect.Selected)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		minWakeSecs, err := nodeSettingsParseUint32SelectLabel(
			"minimum wake time",
			minWakeSecsSelect.Selected,
			nodeSettingsNagTimeoutOptions,
		)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		deviceBatteryInaAddress, err := parseNodePowerUint32Field("battery INA 2xx I2C address", deviceBatteryINAAddressEntry.Text)
		if err != nil {
			return app.NodePowerSettings{}, err
		}

		mu.Lock()
		next := cloneNodePowerSettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.IsPowerSaving = powerSavingBox.Checked
		next.OnBatteryShutdownAfterSecs = onBatteryShutdownAfterSecs
		if adcMultiplierOverrideBox.Checked {
			override, err := parseNodePowerFloat32Field("ADC multiplier override ratio", adcMultiplierOverrideEntry.Text)
			if err != nil {
				return app.NodePowerSettings{}, err
			}
			if override <= 0 {
				return app.NodePowerSettings{}, fmt.Errorf("ADC multiplier override ratio must be greater than zero")
			}
			next.AdcMultiplierOverride = override
		} else {
			next.AdcMultiplierOverride = 0
		}
		next.WaitBluetoothSecs = waitBluetoothSecs
		next.SdsSecs = sdsSecs
		next.MinWakeSecs = minWakeSecs
		next.DeviceBatteryInaAddress = deviceBatteryInaAddress

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node power settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node power settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node power settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node power settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading power settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadPowerSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node power settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node power settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded power settings from device.", 2, 2)
			})
		}()
	}

	powerSavingBox.OnChanged = func(_ bool) { markDirty() }
	shutdownOnPowerLossSelect.OnChanged = func(_ string) { markDirty() }
	adcMultiplierOverrideBox.OnChanged = func(checked bool) {
		if checked {
			raw := strings.TrimSpace(adcMultiplierOverrideEntry.Text)
			if raw == "" || raw == "0" {
				adcMultiplierOverrideEntry.SetText("1")
			}
		}
		markDirty()
		updateFieldAvailability()
	}
	adcMultiplierOverrideEntry.OnChanged = func(_ string) { markDirty() }
	waitBluetoothSecsSelect.OnChanged = func(_ string) { markDirty() }
	superDeepSleepSecsSelect.OnChanged = func(_ string) { markDirty() }
	minWakeSecsSelect.OnChanged = func(_ string) { markDirty() }
	deviceBatteryINAAddressEntry.OnChanged = func(_ string) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node power settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodePowerSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node power settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node power settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node power settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node power settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node power settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node power settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node power settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving power settings…", 1, 3)
		updateButtons()

		go func(settings app.NodePowerSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SavePowerSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node power settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodePowerSettings(settings)
				baselineFormValues = nodePowerFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodePowerSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node power settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved power settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node power settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodePowerSettings{
		AdcMultiplierOverride: 0,
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Power settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Power settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node power settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(bus.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node power settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Power settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node power settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodePowerSettingsFormValues struct {
	IsPowerSaving              bool
	OnBatteryShutdownAfterSecs string
	AdcMultiplierOverrideOn    bool
	AdcMultiplierOverride      string
	WaitBluetoothSecs          string
	SdsSecs                    string
	MinWakeSecs                string
	DeviceBatteryInaAddress    string
}

func nodePowerFormValuesFromSettings(settings app.NodePowerSettings) nodePowerSettingsFormValues {
	overrideEnabled := settings.AdcMultiplierOverride > 0
	overrideValue := "1"
	if overrideEnabled {
		overrideValue = strconv.FormatFloat(float64(settings.AdcMultiplierOverride), 'f', -1, 32)
	}

	return nodePowerSettingsFormValues{
		IsPowerSaving:              settings.IsPowerSaving,
		OnBatteryShutdownAfterSecs: nodePowerAllIntervalLabel(settings.OnBatteryShutdownAfterSecs),
		AdcMultiplierOverrideOn:    overrideEnabled,
		AdcMultiplierOverride:      overrideValue,
		WaitBluetoothSecs:          nodeSettingsSelectCurrentLabel(settings.WaitBluetoothSecs, nodeSettingsNagTimeoutOptions, nodeSettingsCustomSecondsLabel),
		SdsSecs:                    nodePowerAllIntervalLabel(settings.SdsSecs),
		MinWakeSecs:                nodeSettingsSelectCurrentLabel(settings.MinWakeSecs, nodeSettingsNagTimeoutOptions, nodeSettingsCustomSecondsLabel),
		DeviceBatteryInaAddress:    strconv.FormatUint(uint64(settings.DeviceBatteryInaAddress), 10),
	}
}

func cloneNodePowerSettings(settings app.NodePowerSettings) app.NodePowerSettings {
	return settings
}

func parseNodePowerUint32Field(fieldName, raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a non-negative integer", fieldName)
	}

	return uint32(value), nil
}

func parseNodePowerFloat32Field(fieldName, raw string) (float32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", fieldName)
	}

	return float32(value), nil
}

var nodePowerAllIntervalOptions = []nodeSettingsUint32Option{
	{Label: "Unset", Value: 0},
	{Label: nodeSettingsSecondsKnownLabel(1, ""), Value: 1},
	{Label: nodeSettingsSecondsKnownLabel(2, ""), Value: 2},
	{Label: nodeSettingsSecondsKnownLabel(3, ""), Value: 3},
	{Label: nodeSettingsSecondsKnownLabel(4, ""), Value: 4},
	{Label: nodeSettingsSecondsKnownLabel(5, ""), Value: 5},
	{Label: nodeSettingsSecondsKnownLabel(10, ""), Value: 10},
	{Label: nodeSettingsSecondsKnownLabel(15, ""), Value: 15},
	{Label: nodeSettingsSecondsKnownLabel(20, ""), Value: 20},
	{Label: nodeSettingsSecondsKnownLabel(30, ""), Value: 30},
	{Label: nodeSettingsSecondsKnownLabel(45, ""), Value: 45},
	{Label: nodeSettingsSecondsKnownLabel(60, ""), Value: 60},
	{Label: nodeSettingsSecondsKnownLabel(2*60, ""), Value: 2 * 60},
	{Label: nodeSettingsSecondsKnownLabel(5*60, ""), Value: 5 * 60},
	{Label: nodeSettingsSecondsKnownLabel(10*60, ""), Value: 10 * 60},
	{Label: nodeSettingsSecondsKnownLabel(15*60, ""), Value: 15 * 60},
	{Label: nodeSettingsSecondsKnownLabel(30*60, ""), Value: 30 * 60},
	{Label: nodeSettingsSecondsKnownLabel(60*60, ""), Value: 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(2*60*60, ""), Value: 2 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(3*60*60, ""), Value: 3 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(4*60*60, ""), Value: 4 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(5*60*60, ""), Value: 5 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(6*60*60, ""), Value: 6 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(12*60*60, ""), Value: 12 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(18*60*60, ""), Value: 18 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(24*60*60, ""), Value: 24 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(36*60*60, ""), Value: 36 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(48*60*60, ""), Value: 48 * 60 * 60},
	{Label: nodeSettingsSecondsKnownLabel(72*60*60, ""), Value: 72 * 60 * 60},
	{Label: "Always on", Value: math.MaxInt32},
}

func nodePowerAllIntervalCustomLabel(value uint32) string {
	if value == math.MaxInt32 {
		return "Always on"
	}
	if value == 0 {
		return "Unset"
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodePowerAllIntervalLabel(value uint32) string {
	return nodeSettingsSelectCurrentLabel(value, nodePowerAllIntervalOptions, nodePowerAllIntervalCustomLabel)
}

func nodePowerParseAllIntervalSelectLabel(fieldName string, selected string) (uint32, error) {
	selected = strings.TrimSpace(selected)
	if strings.EqualFold(selected, "Always on") {
		return math.MaxInt32, nil
	}
	if strings.EqualFold(selected, "Unset") {
		return 0, nil
	}

	return nodeSettingsParseUint32SelectLabel(fieldName, selected, nodePowerAllIntervalOptions)
}
