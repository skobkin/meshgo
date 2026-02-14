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
	shutdownOnPowerLossEntry := widget.NewEntry()
	adcMultiplierOverrideBox := widget.NewCheck("", nil)
	adcMultiplierOverrideEntry := widget.NewEntry()
	waitBluetoothSecsEntry := widget.NewEntry()
	superDeepSleepSecsEntry := widget.NewEntry()
	minWakeSecsEntry := widget.NewEntry()
	deviceBatteryINAAddressEntry := widget.NewEntry()

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Enable power saving mode", powerSavingBox),
		widget.NewFormItem("Shutdown on power loss (seconds)", shutdownOnPowerLossEntry),
		widget.NewFormItem("ADC multiplier override", adcMultiplierOverrideBox),
		widget.NewFormItem("ADC multiplier override ratio", adcMultiplierOverrideEntry),
		widget.NewFormItem("Wait for Bluetooth duration (seconds)", waitBluetoothSecsEntry),
		widget.NewFormItem("Super deep sleep duration (seconds)", superDeepSleepSecsEntry),
		widget.NewFormItem("Minimum wake time (seconds)", minWakeSecsEntry),
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
		shutdownOnPowerLossEntry.SetText(strconv.FormatUint(uint64(settings.OnBatteryShutdownAfterSecs), 10))
		overrideEnabled := settings.AdcMultiplierOverride > 0
		adcMultiplierOverrideBox.SetChecked(overrideEnabled)
		if overrideEnabled {
			adcMultiplierOverrideEntry.SetText(strconv.FormatFloat(float64(settings.AdcMultiplierOverride), 'f', -1, 32))
		} else {
			adcMultiplierOverrideEntry.SetText("1")
		}
		waitBluetoothSecsEntry.SetText(strconv.FormatUint(uint64(settings.WaitBluetoothSecs), 10))
		superDeepSleepSecsEntry.SetText(strconv.FormatUint(uint64(settings.SdsSecs), 10))
		minWakeSecsEntry.SetText(strconv.FormatUint(uint64(settings.MinWakeSecs), 10))
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
			OnBatteryShutdownAfterSecs: strings.TrimSpace(shutdownOnPowerLossEntry.Text),
			AdcMultiplierOverride:      strings.TrimSpace(adcMultiplierOverrideEntry.Text),
			AdcMultiplierOverrideOn:    adcMultiplierOverrideBox.Checked,
			WaitBluetoothSecs:          strings.TrimSpace(waitBluetoothSecsEntry.Text),
			SdsSecs:                    strings.TrimSpace(superDeepSleepSecsEntry.Text),
			MinWakeSecs:                strings.TrimSpace(minWakeSecsEntry.Text),
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
		onBatteryShutdownAfterSecs, err := parseNodePowerUint32Field("shutdown on power loss", shutdownOnPowerLossEntry.Text)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		waitBluetoothSecs, err := parseNodePowerUint32Field("wait for Bluetooth duration", waitBluetoothSecsEntry.Text)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		sdsSecs, err := parseNodePowerUint32Field("super deep sleep duration", superDeepSleepSecsEntry.Text)
		if err != nil {
			return app.NodePowerSettings{}, err
		}
		minWakeSecs, err := parseNodePowerUint32Field("minimum wake time", minWakeSecsEntry.Text)
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
	shutdownOnPowerLossEntry.OnChanged = func(_ string) { markDirty() }
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
	waitBluetoothSecsEntry.OnChanged = func(_ string) { markDirty() }
	superDeepSleepSecsEntry.OnChanged = func(_ string) { markDirty() }
	minWakeSecsEntry.OnChanged = func(_ string) { markDirty() }
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
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
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
		OnBatteryShutdownAfterSecs: strconv.FormatUint(uint64(settings.OnBatteryShutdownAfterSecs), 10),
		AdcMultiplierOverrideOn:    overrideEnabled,
		AdcMultiplierOverride:      overrideValue,
		WaitBluetoothSecs:          strconv.FormatUint(uint64(settings.WaitBluetoothSecs), 10),
		SdsSecs:                    strconv.FormatUint(uint64(settings.SdsSecs), 10),
		MinWakeSecs:                strconv.FormatUint(uint64(settings.MinWakeSecs), 10),
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
