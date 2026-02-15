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
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func newNodeLoRaSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "radio.lora"
	nodeSettingsTabLogger.Debug("building node LoRa settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading LoRa settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	regionSelect := widget.NewSelect(nil, nil)
	useModemPresetBox := widget.NewCheck("", nil)
	modemPresetSelect := widget.NewSelect(nil, nil)
	bandwidthEntry := widget.NewEntry()
	spreadFactorEntry := widget.NewEntry()
	codingRateEntry := widget.NewEntry()
	ignoreMqttBox := widget.NewCheck("", nil)
	okToMqttBox := widget.NewCheck("", nil)
	txEnabledBox := widget.NewCheck("", nil)
	overrideDutyCycleBox := widget.NewCheck("", nil)
	hopLimitSelect := widget.NewSelect(nil, nil)
	frequencySlotEntry := widget.NewEntry()
	sx126xRxBoostedGainBox := widget.NewCheck("", nil)
	overrideFrequencyEntry := widget.NewEntry()
	txPowerEntry := widget.NewEntry()
	paFanDisabledBox := widget.NewCheck("", nil)

	modemPresetItem := widget.NewFormItem("Modem preset", modemPresetSelect)
	bandwidthItem := widget.NewFormItem("Bandwidth", bandwidthEntry)
	spreadFactorItem := widget.NewFormItem("Spread factor", spreadFactorEntry)
	codingRateItem := widget.NewFormItem("Coding rate", codingRateEntry)

	basicForm := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Region frequency plan", regionSelect),
		widget.NewFormItem("Use modem preset", useModemPresetBox),
	)
	presetForm := widget.NewForm(
		modemPresetItem,
	)
	manualForm := widget.NewForm(
		bandwidthItem,
		spreadFactorItem,
		codingRateItem,
	)
	advancedForm := widget.NewForm(
		widget.NewFormItem("Ignore MQTT", ignoreMqttBox),
		widget.NewFormItem("OK to MQTT", okToMqttBox),
		widget.NewFormItem("TX enabled", txEnabledBox),
		widget.NewFormItem("Override duty cycle", overrideDutyCycleBox),
		widget.NewFormItem("Hop limit", hopLimitSelect),
		widget.NewFormItem("Frequency slot", frequencySlotEntry),
		widget.NewFormItem("SX126X RX boosted gain", sx126xRxBoostedGainBox),
		widget.NewFormItem("Override frequency (MHz)", overrideFrequencyEntry),
		widget.NewFormItem("TX power (dBm)", txPowerEntry),
	)
	paFanForm := widget.NewForm(
		widget.NewFormItem("PA fan disabled", paFanDisabledBox),
	)
	formContent := container.NewVBox(
		basicForm,
		presetForm,
		manualForm,
		advancedForm,
		paFanForm,
	)

	var (
		baseline              app.NodeLoRaSettings
		baselineFormValues    nodeLoRaSettingsFormValues
		dirty                 bool
		saving                bool
		initialReloadStarted  atomic.Bool
		mu                    sync.Mutex
		applyingForm          atomic.Bool
		channelNumAuto        bool
		overrideFrequencyAuto bool
		primaryChannelTitle   string
		presetVisibilityInit  bool
		presetVisibilityMode  bool
		paFanVisibilityInit   bool
		paFanVisibleMode      bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}
	localBoardModel := func() string {
		if dep.Data.NodeStore == nil {
			return ""
		}
		node, known := localNodeSnapshot(dep.Data.NodeStore, dep.Data.LocalNodeID)
		if !known {
			return ""
		}

		return strings.TrimSpace(node.BoardModel)
	}
	if title, ok := nodeLoRaPrimaryTitleFromChatStore(dep.Data.ChatStore); ok {
		primaryChannelTitle = title
	}

	channelTitle := func(settings app.NodeLoRaSettings) string {
		mu.Lock()
		title := strings.TrimSpace(primaryChannelTitle)
		mu.Unlock()

		return nodeLoRaPrimaryChannelTitle(settings, title)
	}

	rawChannelNumText := func() string {
		text := strings.TrimSpace(frequencySlotEntry.Text)
		mu.Lock()
		auto := channelNumAuto
		mu.Unlock()
		if auto {
			return "0"
		}

		return text
	}

	rawOverrideFrequencyText := func() string {
		text := strings.TrimSpace(overrideFrequencyEntry.Text)
		mu.Lock()
		auto := overrideFrequencyAuto
		mu.Unlock()
		if auto {
			return "0"
		}

		return text
	}

	previewSettingsFromForm := func() (app.NodeLoRaSettings, bool) {
		mu.Lock()
		preview := cloneNodeLoRaSettings(baseline)
		mu.Unlock()

		region, err := nodeLoRaParseEnumLabel("region frequency plan", regionSelect.Selected, nodeLoRaRegionOptions)
		if err != nil {
			return app.NodeLoRaSettings{}, false
		}
		preview.Region = region
		preview.UsePreset = useModemPresetBox.Checked
		if preview.UsePreset {
			modemPreset, err := nodeLoRaParseEnumLabel("modem preset", modemPresetSelect.Selected, nodeLoRaModemPresetOptions)
			if err != nil {
				return app.NodeLoRaSettings{}, false
			}
			preview.ModemPreset = modemPreset
		} else {
			bandwidth, err := parseNodeLoRaUint32Field("bandwidth", bandwidthEntry.Text)
			if err != nil {
				return app.NodeLoRaSettings{}, false
			}
			preview.Bandwidth = bandwidth
		}
		channelNum, err := parseNodeLoRaUint32Field("frequency slot", rawChannelNumText())
		if err != nil {
			return app.NodeLoRaSettings{}, false
		}
		overrideFrequency, err := parseNodeLoRaFloat32Field("override frequency", rawOverrideFrequencyText())
		if err != nil {
			return app.NodeLoRaSettings{}, false
		}
		preview.ChannelNum = channelNum
		preview.OverrideFrequency = overrideFrequency

		return preview, true
	}

	refreshComputedDisplay := func() {
		mu.Lock()
		autoChannel := channelNumAuto
		autoOverride := overrideFrequencyAuto
		mu.Unlock()
		if !autoChannel && !autoOverride {
			return
		}

		preview, ok := previewSettingsFromForm()
		if !ok {
			return
		}

		title := channelTitle(preview)
		if autoChannel {
			effectiveChannelNum := nodeLoRaEffectiveChannelNum(preview, title)
			applyingForm.Store(true)
			frequencySlotEntry.SetText(strconv.FormatUint(uint64(effectiveChannelNum), 10))
			applyingForm.Store(false)
		}
		if autoOverride {
			effectiveFrequency := nodeLoRaEffectiveRadioFreq(preview, title)
			applyingForm.Store(true)
			overrideFrequencyEntry.SetText(strconv.FormatFloat(float64(effectiveFrequency), 'f', -1, 32))
			applyingForm.Store(false)
		}
	}

	setForm := func(settings app.NodeLoRaSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		nodeLoRaSetEnumSelect(regionSelect, nodeLoRaRegionOptions, settings.Region)
		useModemPresetBox.SetChecked(settings.UsePreset)
		nodeLoRaSetEnumSelect(modemPresetSelect, nodeLoRaModemPresetOptions, settings.ModemPreset)
		bandwidthEntry.SetText(strconv.FormatUint(uint64(settings.Bandwidth), 10))
		spreadFactorEntry.SetText(strconv.FormatUint(uint64(settings.SpreadFactor), 10))
		codingRateEntry.SetText(strconv.FormatUint(uint64(settings.CodingRate), 10))
		ignoreMqttBox.SetChecked(settings.IgnoreMqtt)
		okToMqttBox.SetChecked(settings.ConfigOkToMqtt)
		txEnabledBox.SetChecked(settings.TxEnabled)
		overrideDutyCycleBox.SetChecked(settings.OverrideDutyCycle)
		nodeLoRaSetHopLimitSelect(hopLimitSelect, settings.HopLimit)

		primaryTitle := channelTitle(settings)
		if settings.ChannelNum == 0 {
			channelNumAuto = true
			effectiveChannelNum := nodeLoRaEffectiveChannelNum(settings, primaryTitle)
			frequencySlotEntry.SetText(strconv.FormatUint(uint64(effectiveChannelNum), 10))
		} else {
			channelNumAuto = false
			frequencySlotEntry.SetText(strconv.FormatUint(uint64(settings.ChannelNum), 10))
		}
		sx126xRxBoostedGainBox.SetChecked(settings.Sx126XRxBoostedGain)
		if settings.OverrideFrequency == 0 {
			overrideFrequencyAuto = true
			effectiveFrequency := nodeLoRaEffectiveRadioFreq(settings, primaryTitle)
			overrideFrequencyEntry.SetText(strconv.FormatFloat(float64(effectiveFrequency), 'f', -1, 32))
		} else {
			overrideFrequencyAuto = false
			overrideFrequencyEntry.SetText(strconv.FormatFloat(float64(settings.OverrideFrequency), 'f', -1, 32))
		}
		txPowerEntry.SetText(strconv.FormatInt(int64(settings.TxPower), 10))
		paFanDisabledBox.SetChecked(settings.PaFanDisabled)
	}

	applyForm := func(settings app.NodeLoRaSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeLoRaSettingsFormValues {
		return nodeLoRaSettingsFormValues{
			Region:              strings.TrimSpace(regionSelect.Selected),
			UsePreset:           useModemPresetBox.Checked,
			ModemPreset:         strings.TrimSpace(modemPresetSelect.Selected),
			Bandwidth:           strings.TrimSpace(bandwidthEntry.Text),
			SpreadFactor:        strings.TrimSpace(spreadFactorEntry.Text),
			CodingRate:          strings.TrimSpace(codingRateEntry.Text),
			IgnoreMqtt:          ignoreMqttBox.Checked,
			ConfigOkToMqtt:      okToMqttBox.Checked,
			TxEnabled:           txEnabledBox.Checked,
			OverrideDutyCycle:   overrideDutyCycleBox.Checked,
			HopLimit:            strings.TrimSpace(hopLimitSelect.Selected),
			ChannelNum:          rawChannelNumText(),
			Sx126XRxBoostedGain: sx126xRxBoostedGainBox.Checked,
			OverrideFrequency:   rawOverrideFrequencyText(),
			TxPower:             strings.TrimSpace(txPowerEntry.Text),
			PaFanDisabled:       paFanDisabledBox.Checked,
		}
	}

	updateFieldAvailability := func() {
		mu.Lock()
		isSaving := saving
		mu.Unlock()

		if isSaving {
			useModemPresetBox.Disable()
		} else {
			useModemPresetBox.Enable()
		}

		showPreset := useModemPresetBox.Checked
		if !isSaving && showPreset {
			modemPresetSelect.Enable()
		} else {
			modemPresetSelect.Disable()
		}

		if !isSaving && !showPreset {
			bandwidthEntry.Enable()
			spreadFactorEntry.Enable()
			codingRateEntry.Enable()
		} else {
			bandwidthEntry.Disable()
			spreadFactorEntry.Disable()
			codingRateEntry.Disable()
		}

		if showPreset {
			presetForm.Show()
			manualForm.Hide()
		} else {
			presetForm.Hide()
			manualForm.Show()
		}
		showPaFan := nodeLoRaHasPaFan(localBoardModel())
		if showPaFan {
			paFanForm.Show()
			if isSaving {
				paFanDisabledBox.Disable()
			} else {
				paFanDisabledBox.Enable()
			}
		} else {
			paFanForm.Hide()
			paFanDisabledBox.Disable()
		}

		needsRefresh := false
		if !presetVisibilityInit || presetVisibilityMode != showPreset {
			presetVisibilityInit = true
			presetVisibilityMode = showPreset
			needsRefresh = true
		}
		if !paFanVisibilityInit || paFanVisibleMode != showPaFan {
			paFanVisibilityInit = true
			paFanVisibleMode = showPaFan
			needsRefresh = true
		}
		if needsRefresh {
			formContent.Refresh()
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
		formValues := readFormValues()
		mu.Lock()
		dirty = formValues != baselineFormValues
		mu.Unlock()
		updateButtons()
	}

	applyLoadedSettings := func(next app.NodeLoRaSettings) {
		mu.Lock()
		baseline = cloneNodeLoRaSettings(next)
		baselineFormValues = nodeLoRaFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeLoRaSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeLoRaSettings, error) {
		region, err := nodeLoRaParseEnumLabel("region frequency plan", regionSelect.Selected, nodeLoRaRegionOptions)
		if err != nil {
			return app.NodeLoRaSettings{}, err
		}
		hopLimit, err := parseNodeLoRaHopLimitField(hopLimitSelect.Selected)
		if err != nil {
			return app.NodeLoRaSettings{}, err
		}
		channelNum, err := parseNodeLoRaUint32Field("frequency slot", rawChannelNumText())
		if err != nil {
			return app.NodeLoRaSettings{}, err
		}
		overrideFrequency, err := parseNodeLoRaFloat32Field("override frequency", rawOverrideFrequencyText())
		if err != nil {
			return app.NodeLoRaSettings{}, err
		}
		txPower, err := parseNodeLoRaInt32Field("TX power", txPowerEntry.Text)
		if err != nil {
			return app.NodeLoRaSettings{}, err
		}

		mu.Lock()
		next := cloneNodeLoRaSettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.Region = region
		next.UsePreset = useModemPresetBox.Checked
		if useModemPresetBox.Checked {
			modemPreset, err := nodeLoRaParseEnumLabel("modem preset", modemPresetSelect.Selected, nodeLoRaModemPresetOptions)
			if err != nil {
				return app.NodeLoRaSettings{}, err
			}
			next.ModemPreset = modemPreset
		} else {
			bandwidth, err := parseNodeLoRaUint32Field("bandwidth", bandwidthEntry.Text)
			if err != nil {
				return app.NodeLoRaSettings{}, err
			}
			spreadFactor, err := parseNodeLoRaUint32Field("spread factor", spreadFactorEntry.Text)
			if err != nil {
				return app.NodeLoRaSettings{}, err
			}
			codingRate, err := parseNodeLoRaUint32Field("coding rate", codingRateEntry.Text)
			if err != nil {
				return app.NodeLoRaSettings{}, err
			}
			next.Bandwidth = bandwidth
			next.SpreadFactor = spreadFactor
			next.CodingRate = codingRate
		}
		next.IgnoreMqtt = ignoreMqttBox.Checked
		next.ConfigOkToMqtt = okToMqttBox.Checked
		next.TxEnabled = txEnabledBox.Checked
		next.OverrideDutyCycle = overrideDutyCycleBox.Checked
		next.HopLimit = hopLimit
		next.ChannelNum = channelNum
		next.Sx126XRxBoostedGain = sx126xRxBoostedGainBox.Checked
		next.OverrideFrequency = overrideFrequency
		next.TxPower = txPower
		next.PaFanDisabled = paFanDisabledBox.Checked
		if maxChannels := nodeLoRaNumChannels(next); maxChannels > 0 && channelNum > maxChannels {
			return app.NodeLoRaSettings{}, fmt.Errorf("frequency slot must be between 0 and %d for selected region/settings", maxChannels)
		}

		return next, nil
	}

	resolveReloadTarget := func(reportFailure bool) (app.NodeSettingsTarget, bool) {
		target, ok := localTarget()
		if !ok {
			if reportFailure {
				nodeSettingsTabLogger.Warn("node LoRa settings reload failed: local node ID is unknown", "page_id", pageID)
				controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)
			}

			return app.NodeSettingsTarget{}, false
		}
		if dep.Actions.NodeSettings == nil {
			if reportFailure {
				nodeSettingsTabLogger.Warn("node LoRa settings reload unavailable: service is not configured", "page_id", pageID)
				controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)
			}

			return app.NodeSettingsTarget{}, false
		}
		if !isConnected() {
			if reportFailure {
				nodeSettingsTabLogger.Info("node LoRa settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)
			}

			return app.NodeSettingsTarget{}, false
		}

		return target, true
	}

	startReloadFromDevice := func(target app.NodeSettingsTarget) {
		nodeSettingsTabLogger.Info("reloading node LoRa settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading LoRa settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadLoRaSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node LoRa settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node LoRa settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded LoRa settings from device.", 2, 2)
			})
		}()
	}

	reloadFromDevice := func(reportFailure bool) bool {
		target, ok := resolveReloadTarget(reportFailure)
		if !ok {
			return false
		}
		startReloadFromDevice(target)

		return true
	}

	tryStartInitialReload := func(trigger string) {
		if dep.Actions.NodeSettings == nil || initialReloadStarted.Load() {
			return
		}
		target, ok := resolveReloadTarget(false)
		if !ok {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node LoRa settings", "page_id", pageID, "trigger", trigger)
		startReloadFromDevice(target)
	}

	regionSelect.OnChanged = func(_ string) {
		refreshComputedDisplay()
		markDirty()
	}
	useModemPresetBox.OnChanged = func(_ bool) {
		refreshComputedDisplay()
		markDirty()
		updateFieldAvailability()
	}
	modemPresetSelect.OnChanged = func(_ string) {
		refreshComputedDisplay()
		markDirty()
	}
	bandwidthEntry.OnChanged = func(_ string) {
		refreshComputedDisplay()
		markDirty()
	}
	spreadFactorEntry.OnChanged = func(_ string) { markDirty() }
	codingRateEntry.OnChanged = func(_ string) { markDirty() }
	ignoreMqttBox.OnChanged = func(_ bool) { markDirty() }
	okToMqttBox.OnChanged = func(_ bool) { markDirty() }
	txEnabledBox.OnChanged = func(_ bool) { markDirty() }
	overrideDutyCycleBox.OnChanged = func(_ bool) { markDirty() }
	hopLimitSelect.OnChanged = func(_ string) { markDirty() }
	frequencySlotEntry.OnChanged = func(_ string) {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		channelNumAuto = false
		mu.Unlock()
		refreshComputedDisplay()
		markDirty()
	}
	sx126xRxBoostedGainBox.OnChanged = func(_ bool) { markDirty() }
	overrideFrequencyEntry.OnChanged = func(_ string) {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		overrideFrequencyAuto = false
		mu.Unlock()
		markDirty()
	}
	txPowerEntry.OnChanged = func(_ string) { markDirty() }
	paFanDisabledBox.OnChanged = func(_ bool) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node LoRa settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeLoRaSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node LoRa settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node LoRa settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node LoRa settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node LoRa settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node LoRa settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node LoRa settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node LoRa settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving LoRa settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeLoRaSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveLoRaSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node LoRa settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeLoRaSettings(settings)
				baselineFormValues = nodeLoRaFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeLoRaSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node LoRa settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved LoRa settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node LoRa settings reload requested", "page_id", pageID)
		reloadFromDevice(true)
	}

	initial := app.NodeLoRaSettings{}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("LoRa settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("LoRa settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node LoRa settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node LoRa settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()

		channelSub := dep.Data.Bus.Subscribe(connectors.TopicChannels)
		go func() {
			for raw := range channelSub {
				channels, ok := raw.(domain.ChannelList)
				if !ok {
					continue
				}
				title, ok := nodeLoRaPrimaryTitleFromChannels(channels)
				if !ok {
					continue
				}
				fyne.Do(func() {
					mu.Lock()
					primaryChannelTitle = title
					mu.Unlock()
					refreshComputedDisplay()
					markDirty()
				})
			}
		}()
	}
	if dep.Data.NodeStore != nil {
		nodeSettingsTabLogger.Debug("starting node LoRa settings page listener for local node store changes", "page_id", pageID)
		go func() {
			for range dep.Data.NodeStore.Changes() {
				fyne.Do(func() {
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("LoRa settings are loaded from and saved to the connected local node."),
		formContent,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		tryStartInitialReload("tab_opened")
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeLoRaSettingsFormValues struct {
	Region              string
	UsePreset           bool
	ModemPreset         string
	Bandwidth           string
	SpreadFactor        string
	CodingRate          string
	IgnoreMqtt          bool
	ConfigOkToMqtt      bool
	TxEnabled           bool
	OverrideDutyCycle   bool
	HopLimit            string
	ChannelNum          string
	Sx126XRxBoostedGain bool
	OverrideFrequency   string
	TxPower             string
	PaFanDisabled       bool
}

type nodeLoRaEnumOption struct {
	Label string
	Value int32
}

var nodeLoRaRegionOptions = []nodeLoRaEnumOption{
	{Label: "Unset", Value: int32(generated.Config_LoRaConfig_UNSET)},
	{Label: "United States (US)", Value: int32(generated.Config_LoRaConfig_US)},
	{Label: "Europe 433 MHz (EU_433)", Value: int32(generated.Config_LoRaConfig_EU_433)},
	{Label: "Europe 868 MHz (EU_868)", Value: int32(generated.Config_LoRaConfig_EU_868)},
	{Label: "China (CN)", Value: int32(generated.Config_LoRaConfig_CN)},
	{Label: "Japan (JP)", Value: int32(generated.Config_LoRaConfig_JP)},
	{Label: "Australia/New Zealand (ANZ)", Value: int32(generated.Config_LoRaConfig_ANZ)},
	{Label: "Korea (KR)", Value: int32(generated.Config_LoRaConfig_KR)},
	{Label: "Taiwan (TW)", Value: int32(generated.Config_LoRaConfig_TW)},
	{Label: "Russia (RU)", Value: int32(generated.Config_LoRaConfig_RU)},
	{Label: "India (IN)", Value: int32(generated.Config_LoRaConfig_IN)},
	{Label: "New Zealand 865 MHz (NZ_865)", Value: int32(generated.Config_LoRaConfig_NZ_865)},
	{Label: "Thailand (TH)", Value: int32(generated.Config_LoRaConfig_TH)},
	{Label: "2.4 GHz (LORA_24)", Value: int32(generated.Config_LoRaConfig_LORA_24)},
	{Label: "Ukraine 433 MHz (UA_433)", Value: int32(generated.Config_LoRaConfig_UA_433)},
	{Label: "Ukraine 868 MHz (UA_868)", Value: int32(generated.Config_LoRaConfig_UA_868)},
	{Label: "Malaysia 433 MHz (MY_433)", Value: int32(generated.Config_LoRaConfig_MY_433)},
	{Label: "Malaysia 919 MHz (MY_919)", Value: int32(generated.Config_LoRaConfig_MY_919)},
	{Label: "Singapore 923 MHz (SG_923)", Value: int32(generated.Config_LoRaConfig_SG_923)},
	{Label: "Philippines 433 MHz (PH_433)", Value: int32(generated.Config_LoRaConfig_PH_433)},
	{Label: "Philippines 868 MHz (PH_868)", Value: int32(generated.Config_LoRaConfig_PH_868)},
	{Label: "Philippines 915 MHz (PH_915)", Value: int32(generated.Config_LoRaConfig_PH_915)},
	{Label: "Australia/New Zealand 433 MHz (ANZ_433)", Value: int32(generated.Config_LoRaConfig_ANZ_433)},
	{Label: "Kazakhstan 433 MHz (KZ_433)", Value: int32(generated.Config_LoRaConfig_KZ_433)},
	{Label: "Kazakhstan 863 MHz (KZ_863)", Value: int32(generated.Config_LoRaConfig_KZ_863)},
	{Label: "Nepal 865 MHz (NP_865)", Value: int32(generated.Config_LoRaConfig_NP_865)},
	{Label: "Brazil 902 MHz (BR_902)", Value: int32(generated.Config_LoRaConfig_BR_902)},
}

var nodeLoRaModemPresetOptions = []nodeLoRaEnumOption{
	{Label: "Long Fast", Value: int32(generated.Config_LoRaConfig_LONG_FAST)},
	// Value 1 and 2 are preserved for firmware compatibility; upstream enum names are deprecated.
	{Label: "Long Slow", Value: 1},
	{Label: "Very Long Slow", Value: 2},
	{Label: "Medium Slow", Value: int32(generated.Config_LoRaConfig_MEDIUM_SLOW)},
	{Label: "Medium Fast", Value: int32(generated.Config_LoRaConfig_MEDIUM_FAST)},
	{Label: "Short Slow", Value: int32(generated.Config_LoRaConfig_SHORT_SLOW)},
	{Label: "Short Fast", Value: int32(generated.Config_LoRaConfig_SHORT_FAST)},
	{Label: "Long Moderate", Value: int32(generated.Config_LoRaConfig_LONG_MODERATE)},
	{Label: "Short Turbo", Value: int32(generated.Config_LoRaConfig_SHORT_TURBO)},
	{Label: "Long Turbo", Value: int32(generated.Config_LoRaConfig_LONG_TURBO)},
}

var nodeLoRaHopLimitOptions = []string{"0", "1", "2", "3", "4", "5", "6", "7"}

type nodeLoRaRegionInfo struct {
	StartMHz float32
	EndMHz   float32
	WideLoRa bool
}

var nodeLoRaRegionInfoByCode = map[int32]nodeLoRaRegionInfo{
	int32(generated.Config_LoRaConfig_UNSET):   {StartMHz: 902.0, EndMHz: 928.0},
	int32(generated.Config_LoRaConfig_US):      {StartMHz: 902.0, EndMHz: 928.0},
	int32(generated.Config_LoRaConfig_EU_433):  {StartMHz: 433.0, EndMHz: 434.0},
	int32(generated.Config_LoRaConfig_EU_868):  {StartMHz: 869.4, EndMHz: 869.65},
	int32(generated.Config_LoRaConfig_CN):      {StartMHz: 470.0, EndMHz: 510.0},
	int32(generated.Config_LoRaConfig_JP):      {StartMHz: 920.5, EndMHz: 923.5},
	int32(generated.Config_LoRaConfig_ANZ):     {StartMHz: 915.0, EndMHz: 928.0},
	int32(generated.Config_LoRaConfig_KR):      {StartMHz: 920.0, EndMHz: 923.0},
	int32(generated.Config_LoRaConfig_TW):      {StartMHz: 920.0, EndMHz: 925.0},
	int32(generated.Config_LoRaConfig_RU):      {StartMHz: 868.7, EndMHz: 869.2},
	int32(generated.Config_LoRaConfig_IN):      {StartMHz: 865.0, EndMHz: 867.0},
	int32(generated.Config_LoRaConfig_NZ_865):  {StartMHz: 864.0, EndMHz: 868.0},
	int32(generated.Config_LoRaConfig_TH):      {StartMHz: 920.0, EndMHz: 925.0},
	int32(generated.Config_LoRaConfig_LORA_24): {StartMHz: 2400.0, EndMHz: 2483.5, WideLoRa: true},
	int32(generated.Config_LoRaConfig_UA_433):  {StartMHz: 433.0, EndMHz: 434.7},
	int32(generated.Config_LoRaConfig_UA_868):  {StartMHz: 868.0, EndMHz: 868.6},
	int32(generated.Config_LoRaConfig_MY_433):  {StartMHz: 433.0, EndMHz: 435.0},
	int32(generated.Config_LoRaConfig_MY_919):  {StartMHz: 919.0, EndMHz: 924.0},
	int32(generated.Config_LoRaConfig_SG_923):  {StartMHz: 917.0, EndMHz: 925.0},
	int32(generated.Config_LoRaConfig_PH_433):  {StartMHz: 433.0, EndMHz: 434.7},
	int32(generated.Config_LoRaConfig_PH_868):  {StartMHz: 868.0, EndMHz: 869.4},
	int32(generated.Config_LoRaConfig_PH_915):  {StartMHz: 915.0, EndMHz: 918.0},
	int32(generated.Config_LoRaConfig_ANZ_433): {StartMHz: 433.05, EndMHz: 434.79},
	int32(generated.Config_LoRaConfig_KZ_433):  {StartMHz: 433.075, EndMHz: 434.775},
	int32(generated.Config_LoRaConfig_KZ_863):  {StartMHz: 863.0, EndMHz: 868.0, WideLoRa: true},
	int32(generated.Config_LoRaConfig_NP_865):  {StartMHz: 865.0, EndMHz: 868.0},
	int32(generated.Config_LoRaConfig_BR_902):  {StartMHz: 902.0, EndMHz: 907.5},
}

var nodeLoRaBandwidthByPresetMHz = map[int32]float32{
	// Values 1 and 2 are preserved for firmware compatibility; upstream enum names are deprecated.
	2: 0.0625,
	int32(generated.Config_LoRaConfig_LONG_TURBO):    0.5,
	int32(generated.Config_LoRaConfig_LONG_FAST):     0.25,
	int32(generated.Config_LoRaConfig_LONG_MODERATE): 0.125,
	1: 0.125,
	int32(generated.Config_LoRaConfig_MEDIUM_FAST): 0.25,
	int32(generated.Config_LoRaConfig_MEDIUM_SLOW): 0.25,
	int32(generated.Config_LoRaConfig_SHORT_FAST):  0.25,
	int32(generated.Config_LoRaConfig_SHORT_SLOW):  0.25,
	int32(generated.Config_LoRaConfig_SHORT_TURBO): 0.5,
}

func nodeLoRaPrimaryTitleFromChannels(channels domain.ChannelList) (string, bool) {
	for _, item := range channels.Items {
		if item.Index != 0 {
			continue
		}
		title := strings.TrimSpace(item.Title)
		if title == "" {
			return "", false
		}

		return title, true
	}

	return "", false
}

func nodeLoRaPrimaryTitleFromChatStore(store *domain.ChatStore) (string, bool) {
	title := strings.TrimSpace(domain.ChatTitleByKey(store, domain.ChatKeyForChannel(0)))
	if title == "" || strings.EqualFold(title, domain.ChatKeyForChannel(0)) {
		return "", false
	}

	return title, true
}

func nodeLoRaPrimaryChannelTitle(settings app.NodeLoRaSettings, knownTitle string) string {
	if title := strings.TrimSpace(knownTitle); title != "" {
		return title
	}
	if !settings.UsePreset {
		return "Custom"
	}

	switch settings.ModemPreset {
	case int32(generated.Config_LoRaConfig_SHORT_TURBO):
		return "ShortTurbo"
	case int32(generated.Config_LoRaConfig_SHORT_FAST):
		return "ShortFast"
	case int32(generated.Config_LoRaConfig_SHORT_SLOW):
		return "ShortSlow"
	case int32(generated.Config_LoRaConfig_MEDIUM_FAST):
		return "MediumFast"
	case int32(generated.Config_LoRaConfig_MEDIUM_SLOW):
		return "MediumSlow"
	case int32(generated.Config_LoRaConfig_LONG_FAST):
		return "LongFast"
	case 1:
		return "LongSlow"
	case int32(generated.Config_LoRaConfig_LONG_MODERATE):
		return "LongMod"
	case 2:
		return "VLongSlow"
	case int32(generated.Config_LoRaConfig_LONG_TURBO):
		return "LongTurbo"
	default:
		return "Invalid"
	}
}

func nodeLoRaNumChannels(settings app.NodeLoRaSettings) uint32 {
	region, ok := nodeLoRaRegionInfoByCode[settings.Region]
	if !ok {
		return 0
	}
	bandwidthMHz := nodeLoRaBandwidthMHz(settings, region)
	if bandwidthMHz <= 0 {
		return 1
	}
	channelCount := math.Floor(float64((region.EndMHz - region.StartMHz) / bandwidthMHz))
	if channelCount > 0 && channelCount <= float64(^uint32(0)) {
		return uint32(channelCount)
	}

	return 1
}

func nodeLoRaEffectiveChannelNum(settings app.NodeLoRaSettings, primaryChannelTitle string) uint32 {
	if settings.ChannelNum != 0 {
		return settings.ChannelNum
	}
	channelCount := nodeLoRaNumChannels(settings)
	if channelCount == 0 {
		return 0
	}
	hash := nodeLoRaDJB2(primaryChannelTitle)

	return (hash % channelCount) + 1
}

func nodeLoRaEffectiveRadioFreq(settings app.NodeLoRaSettings, primaryChannelTitle string) float32 {
	if settings.OverrideFrequency != 0 {
		return settings.OverrideFrequency + settings.FrequencyOffset
	}
	region, ok := nodeLoRaRegionInfoByCode[settings.Region]
	if !ok {
		return 0
	}
	bandwidthMHz := nodeLoRaBandwidthMHz(settings, region)
	channelNum := nodeLoRaEffectiveChannelNum(settings, primaryChannelTitle)
	if bandwidthMHz <= 0 || channelNum == 0 {
		return 0
	}

	return (region.StartMHz + bandwidthMHz/2) + (float32(channelNum)-1)*bandwidthMHz
}

func nodeLoRaBandwidthMHz(settings app.NodeLoRaSettings, region nodeLoRaRegionInfo) float32 {
	if settings.UsePreset {
		presetBandwidth, ok := nodeLoRaBandwidthByPresetMHz[settings.ModemPreset]
		if !ok {
			return 0
		}
		if region.WideLoRa {
			return presetBandwidth * 3.25
		}

		return presetBandwidth
	}
	switch settings.Bandwidth {
	case 31:
		return 0.03125
	case 62:
		return 0.0625
	case 200:
		return 0.203125
	case 400:
		return 0.40625
	case 800:
		return 0.8125
	case 1600:
		return 1.625
	default:
		return float32(settings.Bandwidth) / 1000.0
	}
}

func nodeLoRaDJB2(name string) uint32 {
	hash := uint32(5381)
	for _, ch := range name {
		hash += (hash << 5) + uint32(ch)
	}

	return hash
}

func nodeLoRaHasPaFan(boardModel string) bool {
	model := strings.TrimSpace(strings.ToUpper(boardModel))
	if model == "" || model == generated.HardwareModel_UNSET.String() {
		return true
	}

	switch model {
	case generated.HardwareModel_BETAFPV_2400_TX.String(),
		generated.HardwareModel_RADIOMASTER_900_BANDIT_NANO.String(),
		generated.HardwareModel_RADIOMASTER_900_BANDIT.String():
		return true
	default:
		return false
	}
}

func nodeLoRaFormValuesFromSettings(settings app.NodeLoRaSettings) nodeLoRaSettingsFormValues {
	return nodeLoRaSettingsFormValues{
		Region:              nodeLoRaEnumLabel(settings.Region, nodeLoRaRegionOptions),
		UsePreset:           settings.UsePreset,
		ModemPreset:         nodeLoRaEnumLabel(settings.ModemPreset, nodeLoRaModemPresetOptions),
		Bandwidth:           strconv.FormatUint(uint64(settings.Bandwidth), 10),
		SpreadFactor:        strconv.FormatUint(uint64(settings.SpreadFactor), 10),
		CodingRate:          strconv.FormatUint(uint64(settings.CodingRate), 10),
		IgnoreMqtt:          settings.IgnoreMqtt,
		ConfigOkToMqtt:      settings.ConfigOkToMqtt,
		TxEnabled:           settings.TxEnabled,
		OverrideDutyCycle:   settings.OverrideDutyCycle,
		HopLimit:            strconv.FormatUint(uint64(settings.HopLimit), 10),
		ChannelNum:          strconv.FormatUint(uint64(settings.ChannelNum), 10),
		Sx126XRxBoostedGain: settings.Sx126XRxBoostedGain,
		OverrideFrequency:   strconv.FormatFloat(float64(settings.OverrideFrequency), 'f', -1, 32),
		TxPower:             strconv.FormatInt(int64(settings.TxPower), 10),
		PaFanDisabled:       settings.PaFanDisabled,
	}
}

func cloneNodeLoRaSettings(settings app.NodeLoRaSettings) app.NodeLoRaSettings {
	out := settings
	out.IgnoreIncoming = cloneNodeLoRaUint32Slice(settings.IgnoreIncoming)

	return out
}

func parseNodeLoRaUint32Field(fieldName, raw string) (uint32, error) {
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

func parseNodeLoRaInt32Field(fieldName, raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", fieldName)
	}

	return int32(value), nil
}

func parseNodeLoRaFloat32Field(fieldName, raw string) (float32, error) {
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

func parseNodeLoRaHopLimitField(raw string) (uint32, error) {
	value, err := parseNodeLoRaUint32Field("hop limit", raw)
	if err != nil {
		return 0, err
	}
	if value > 7 {
		return 0, fmt.Errorf("hop limit must be between 0 and 7")
	}

	return value, nil
}

func nodeLoRaSetHopLimitSelect(selectWidget *widget.Select, value uint32) {
	options := append([]string(nil), nodeLoRaHopLimitOptions...)
	selected := strconv.FormatUint(uint64(value), 10)
	found := false
	for _, option := range options {
		if option == selected {
			found = true

			break
		}
	}
	if !found {
		options = append(options, selected)
	}
	selectWidget.SetOptions(options)
	selectWidget.SetSelected(selected)
}

func nodeLoRaSetEnumSelect(selectWidget *widget.Select, options []nodeLoRaEnumOption, value int32) {
	optionLabels := nodeLoRaEnumOptionsLabels(options)
	selected := nodeLoRaEnumLabel(value, options)
	if strings.HasPrefix(selected, "Unknown (") {
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeLoRaEnumLabel(value int32, options []nodeLoRaEnumOption) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return nodeLoRaUnknownEnumLabel(value)
}

func nodeLoRaEnumOptionsLabels(options []nodeLoRaEnumOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func nodeLoRaUnknownEnumLabel(value int32) string {
	return fmt.Sprintf("Unknown (%d)", value)
}

func nodeLoRaParseEnumLabel(fieldName, selected string, options []nodeLoRaEnumOption) (int32, error) {
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

func cloneNodeLoRaUint32Slice(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}

	out := make([]uint32, len(values))
	copy(out, values)

	return out
}
