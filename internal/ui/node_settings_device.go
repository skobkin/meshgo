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

const (
	nodeDeviceNodeInfoIntervalUnset  uint32 = 0            // unset
	nodeDeviceNodeInfoInterval3Hour  uint32 = 3 * 60 * 60  // 3 hours
	nodeDeviceNodeInfoInterval4Hour  uint32 = 4 * 60 * 60  // 4 hours
	nodeDeviceNodeInfoInterval5Hour  uint32 = 5 * 60 * 60  // 5 hours
	nodeDeviceNodeInfoInterval6Hour  uint32 = 6 * 60 * 60  // 6 hours
	nodeDeviceNodeInfoInterval12Hour uint32 = 12 * 60 * 60 // 12 hours
	nodeDeviceNodeInfoInterval18Hour uint32 = 18 * 60 * 60 // 18 hours
	nodeDeviceNodeInfoInterval24Hour uint32 = 24 * 60 * 60 // 24 hours
	nodeDeviceNodeInfoInterval36Hour uint32 = 36 * 60 * 60 // 36 hours
	nodeDeviceNodeInfoInterval48Hour uint32 = 48 * 60 * 60 // 48 hours
	nodeDeviceNodeInfoInterval72Hour uint32 = 72 * 60 * 60 // 72 hours
)

func newNodeDeviceSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "device.device"
	nodeSettingsTabLogger.Debug("building node device settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading device settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	roleSelect := widget.NewSelect(nil, nil)
	rebroadcastModeSelect := widget.NewSelect(nil, nil)
	buzzerModeSelect := widget.NewSelect(nil, nil)
	nodeInfoBroadcastSecsSelect := widget.NewSelect(nil, nil)
	buttonGPIOEntry := widget.NewEntry()
	buzzerGPIOEntry := widget.NewEntry()
	tzdefEntry := widget.NewEntry()
	doubleTapAsButtonPressBox := widget.NewCheck("", nil)
	disableTripleClickBox := widget.NewCheck("", nil)
	ledHeartbeatDisabledBox := widget.NewCheck("", nil)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Role", roleSelect),
		widget.NewFormItem("Rebroadcast mode", rebroadcastModeSelect),
		widget.NewFormItem("Node info broadcast interval", nodeInfoBroadcastSecsSelect),
		widget.NewFormItem("Button GPIO", buttonGPIOEntry),
		widget.NewFormItem("Buzzer GPIO", buzzerGPIOEntry),
		widget.NewFormItem("Timezone (POSIX TZDEF)", tzdefEntry),
		widget.NewFormItem("Double tap as button press", doubleTapAsButtonPressBox),
		widget.NewFormItem("Disable triple-click shortcut", disableTripleClickBox),
		widget.NewFormItem("Disable LED heartbeat", ledHeartbeatDisabledBox),
		widget.NewFormItem("Buzzer mode", buzzerModeSelect),
	)

	var (
		baseline             app.NodeDeviceSettings
		baselineFormValues   nodeDeviceSettingsFormValues
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

	setForm := func(settings app.NodeDeviceSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		nodeDeviceSetNodeInfoBroadcastIntervalSelect(nodeInfoBroadcastSecsSelect, settings.NodeInfoBroadcastSecs)
		buttonGPIOEntry.SetText(strconv.FormatUint(uint64(settings.ButtonGPIO), 10))
		buzzerGPIOEntry.SetText(strconv.FormatUint(uint64(settings.BuzzerGPIO), 10))
		tzdefEntry.SetText(strings.TrimSpace(settings.Tzdef))
		doubleTapAsButtonPressBox.SetChecked(settings.DoubleTapAsButtonPress)
		disableTripleClickBox.SetChecked(settings.DisableTripleClick)
		ledHeartbeatDisabledBox.SetChecked(settings.LedHeartbeatDisabled)
		nodeDeviceSetEnumSelect(roleSelect, nodeDeviceRoleOptions, settings.Role)
		nodeDeviceSetEnumSelect(rebroadcastModeSelect, nodeDeviceRebroadcastModeOptions, settings.RebroadcastMode)
		nodeDeviceSetEnumSelect(buzzerModeSelect, nodeDeviceBuzzerModeOptions, settings.BuzzerMode)
	}

	applyForm := func(settings app.NodeDeviceSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeDeviceSettingsFormValues {
		return nodeDeviceSettingsFormValues{
			Role:                   strings.TrimSpace(roleSelect.Selected),
			RebroadcastMode:        strings.TrimSpace(rebroadcastModeSelect.Selected),
			NodeInfoBroadcastSecs:  strings.TrimSpace(nodeInfoBroadcastSecsSelect.Selected),
			ButtonGPIO:             strings.TrimSpace(buttonGPIOEntry.Text),
			BuzzerGPIO:             strings.TrimSpace(buzzerGPIOEntry.Text),
			Tzdef:                  strings.TrimSpace(tzdefEntry.Text),
			DoubleTapAsButtonPress: doubleTapAsButtonPressBox.Checked,
			DisableTripleClick:     disableTripleClickBox.Checked,
			LedHeartbeatDisabled:   ledHeartbeatDisabledBox.Checked,
			BuzzerMode:             strings.TrimSpace(buzzerModeSelect.Selected),
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

	applyLoadedSettings := func(next app.NodeDeviceSettings) {
		mu.Lock()
		baseline = cloneNodeDeviceSettings(next)
		baselineFormValues = nodeDeviceFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeDeviceSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeDeviceSettings, error) {
		role, err := nodeDeviceParseEnumLabel("role", roleSelect.Selected, nodeDeviceRoleOptions)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}
		rebroadcastMode, err := nodeDeviceParseEnumLabel("rebroadcast mode", rebroadcastModeSelect.Selected, nodeDeviceRebroadcastModeOptions)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}
		buzzerMode, err := nodeDeviceParseEnumLabel("buzzer mode", buzzerModeSelect.Selected, nodeDeviceBuzzerModeOptions)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}
		nodeInfoBroadcastSecs, err := nodeDeviceParseNodeInfoBroadcastIntervalLabel("node info broadcast interval", nodeInfoBroadcastSecsSelect.Selected)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}
		buttonGPIO, err := parseNodeDeviceUint32Field("button GPIO", buttonGPIOEntry.Text)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}
		buzzerGPIO, err := parseNodeDeviceUint32Field("buzzer GPIO", buzzerGPIOEntry.Text)
		if err != nil {
			return app.NodeDeviceSettings{}, err
		}

		mu.Lock()
		next := cloneNodeDeviceSettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.Role = role
		next.RebroadcastMode = rebroadcastMode
		next.NodeInfoBroadcastSecs = nodeInfoBroadcastSecs
		next.ButtonGPIO = buttonGPIO
		next.BuzzerGPIO = buzzerGPIO
		next.Tzdef = strings.TrimSpace(tzdefEntry.Text)
		next.DoubleTapAsButtonPress = doubleTapAsButtonPressBox.Checked
		next.DisableTripleClick = disableTripleClickBox.Checked
		next.LedHeartbeatDisabled = ledHeartbeatDisabledBox.Checked
		next.BuzzerMode = buzzerMode

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node device settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node device settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node device settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node device settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading device settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadDeviceSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node device settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node device settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded device settings from device.", 2, 2)
			})
		}()
	}

	roleSelect.OnChanged = func(_ string) { markDirty() }
	rebroadcastModeSelect.OnChanged = func(_ string) { markDirty() }
	buzzerModeSelect.OnChanged = func(_ string) { markDirty() }
	nodeInfoBroadcastSecsSelect.OnChanged = func(_ string) { markDirty() }
	buttonGPIOEntry.OnChanged = func(_ string) { markDirty() }
	buzzerGPIOEntry.OnChanged = func(_ string) { markDirty() }
	tzdefEntry.OnChanged = func(_ string) { markDirty() }
	doubleTapAsButtonPressBox.OnChanged = func(_ bool) { markDirty() }
	disableTripleClickBox.OnChanged = func(_ bool) { markDirty() }
	ledHeartbeatDisabledBox.OnChanged = func(_ bool) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node device settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeDeviceSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node device settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node device settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node device settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node device settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node device settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node device settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node device settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving device settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeDeviceSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveDeviceSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node device settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeDeviceSettings(settings)
				baselineFormValues = nodeDeviceFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeDeviceSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node device settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved device settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node device settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeDeviceSettings{
		Role:            int32(generated.Config_DeviceConfig_CLIENT),
		RebroadcastMode: int32(generated.Config_DeviceConfig_ALL),
		BuzzerMode:      int32(generated.Config_DeviceConfig_ALL_ENABLED),
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Device settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Device settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node device settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node device settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Device settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node device settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeDeviceSettingsFormValues struct {
	Role                   string
	RebroadcastMode        string
	NodeInfoBroadcastSecs  string
	ButtonGPIO             string
	BuzzerGPIO             string
	Tzdef                  string
	DoubleTapAsButtonPress bool
	DisableTripleClick     bool
	LedHeartbeatDisabled   bool
	BuzzerMode             string
}

type nodeDeviceEnumOption struct {
	Label string
	Value int32
}

var nodeDeviceRoleOptions = []nodeDeviceEnumOption{
	{Label: "Client", Value: int32(generated.Config_DeviceConfig_CLIENT)},
	{Label: "Client mute", Value: int32(generated.Config_DeviceConfig_CLIENT_MUTE)},
	{Label: "Router", Value: int32(generated.Config_DeviceConfig_ROUTER)},
	{Label: "Tracker", Value: int32(generated.Config_DeviceConfig_TRACKER)},
	{Label: "Sensor", Value: int32(generated.Config_DeviceConfig_SENSOR)},
	{Label: "TAK", Value: int32(generated.Config_DeviceConfig_TAK)},
	{Label: "Client hidden", Value: int32(generated.Config_DeviceConfig_CLIENT_HIDDEN)},
	{Label: "Lost and found", Value: int32(generated.Config_DeviceConfig_LOST_AND_FOUND)},
	{Label: "TAK tracker", Value: int32(generated.Config_DeviceConfig_TAK_TRACKER)},
	{Label: "Router late", Value: int32(generated.Config_DeviceConfig_ROUTER_LATE)},
	{Label: "Client base", Value: int32(generated.Config_DeviceConfig_CLIENT_BASE)},
}

var nodeDeviceRebroadcastModeOptions = []nodeDeviceEnumOption{
	{Label: "All", Value: int32(generated.Config_DeviceConfig_ALL)},
	{Label: "All (skip decoding)", Value: int32(generated.Config_DeviceConfig_ALL_SKIP_DECODING)},
	{Label: "Local only", Value: int32(generated.Config_DeviceConfig_LOCAL_ONLY)},
	{Label: "Known only", Value: int32(generated.Config_DeviceConfig_KNOWN_ONLY)},
	{Label: "None", Value: int32(generated.Config_DeviceConfig_NONE)},
	{Label: "Core portnums only", Value: int32(generated.Config_DeviceConfig_CORE_PORTNUMS_ONLY)},
}

var nodeDeviceBuzzerModeOptions = []nodeDeviceEnumOption{
	{Label: "All enabled", Value: int32(generated.Config_DeviceConfig_ALL_ENABLED)},
	{Label: "Disabled", Value: int32(generated.Config_DeviceConfig_DISABLED)},
	{Label: "Notifications only", Value: int32(generated.Config_DeviceConfig_NOTIFICATIONS_ONLY)},
	{Label: "System only", Value: int32(generated.Config_DeviceConfig_SYSTEM_ONLY)},
	{Label: "Direct messages only", Value: int32(generated.Config_DeviceConfig_DIRECT_MSG_ONLY)},
}

var nodeDeviceNodeInfoBroadcastIntervalOptions = []nodeSettingsUint32Option{
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoIntervalUnset, "Unset"), Value: nodeDeviceNodeInfoIntervalUnset},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval3Hour, "Unset"), Value: nodeDeviceNodeInfoInterval3Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval4Hour, "Unset"), Value: nodeDeviceNodeInfoInterval4Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval5Hour, "Unset"), Value: nodeDeviceNodeInfoInterval5Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval6Hour, "Unset"), Value: nodeDeviceNodeInfoInterval6Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval12Hour, "Unset"), Value: nodeDeviceNodeInfoInterval12Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval18Hour, "Unset"), Value: nodeDeviceNodeInfoInterval18Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval24Hour, "Unset"), Value: nodeDeviceNodeInfoInterval24Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval36Hour, "Unset"), Value: nodeDeviceNodeInfoInterval36Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval48Hour, "Unset"), Value: nodeDeviceNodeInfoInterval48Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodeDeviceNodeInfoInterval72Hour, "Unset"), Value: nodeDeviceNodeInfoInterval72Hour},
}

func nodeDeviceFormValuesFromSettings(settings app.NodeDeviceSettings) nodeDeviceSettingsFormValues {
	return nodeDeviceSettingsFormValues{
		Role:                   nodeDeviceEnumLabel(settings.Role, nodeDeviceRoleOptions),
		RebroadcastMode:        nodeDeviceEnumLabel(settings.RebroadcastMode, nodeDeviceRebroadcastModeOptions),
		NodeInfoBroadcastSecs:  nodeDeviceNodeInfoBroadcastIntervalSelectLabel(settings.NodeInfoBroadcastSecs),
		ButtonGPIO:             strconv.FormatUint(uint64(settings.ButtonGPIO), 10),
		BuzzerGPIO:             strconv.FormatUint(uint64(settings.BuzzerGPIO), 10),
		Tzdef:                  strings.TrimSpace(settings.Tzdef),
		DoubleTapAsButtonPress: settings.DoubleTapAsButtonPress,
		DisableTripleClick:     settings.DisableTripleClick,
		LedHeartbeatDisabled:   settings.LedHeartbeatDisabled,
		BuzzerMode:             nodeDeviceEnumLabel(settings.BuzzerMode, nodeDeviceBuzzerModeOptions),
	}
}

func cloneNodeDeviceSettings(settings app.NodeDeviceSettings) app.NodeDeviceSettings {
	return settings
}

func parseNodeDeviceUint32Field(fieldName, raw string) (uint32, error) {
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

func nodeDeviceSetNodeInfoBroadcastIntervalSelect(selectWidget *widget.Select, value uint32) {
	nodeSettingsSetUint32Select(
		selectWidget,
		nodeDeviceNodeInfoBroadcastIntervalOptions,
		value,
		nodeSettingsCustomSecondsLabel,
	)
}

func nodeDeviceNodeInfoBroadcastIntervalSelectLabel(value uint32) string {
	label := nodeSettingsUint32OptionLabel(value, nodeDeviceNodeInfoBroadcastIntervalOptions)
	if label != "" {
		return label
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodeDeviceParseNodeInfoBroadcastIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeSettingsParseUint32SelectLabel(
		fieldName,
		selected,
		nodeDeviceNodeInfoBroadcastIntervalOptions,
		nodeSettingsCustomSecondsLabelSuffix,
	)
}

func nodeDeviceSetEnumSelect(selectWidget *widget.Select, options []nodeDeviceEnumOption, value int32) {
	optionLabels := nodeDeviceEnumOptionsLabels(options)
	selected := nodeDeviceEnumLabel(value, options)
	if strings.HasPrefix(selected, "Unknown (") {
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeDeviceEnumLabel(value int32, options []nodeDeviceEnumOption) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return nodeDeviceUnknownEnumLabel(value)
}

func nodeDeviceEnumOptionsLabels(options []nodeDeviceEnumOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func nodeDeviceUnknownEnumLabel(value int32) string {
	return fmt.Sprintf("Unknown (%d)", value)
}

func nodeDeviceParseEnumLabel(fieldName, selected string, options []nodeDeviceEnumOption) (int32, error) {
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
