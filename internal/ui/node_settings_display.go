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

func newNodeDisplaySettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "device.display"
	nodeSettingsTabLogger.Debug("building node display settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading display settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	compassNorthTopBox := widget.NewCheck("", nil)
	use12HClockBox := widget.NewCheck("", nil)
	headingBoldBox := widget.NewCheck("", nil)
	unitsSelect := widget.NewSelect(nil, nil)
	screenOnSecsEntry := widget.NewEntry()
	carouselSecsEntry := widget.NewEntry()
	wakeOnTapOrMotionBox := widget.NewCheck("", nil)
	flipScreenBox := widget.NewCheck("", nil)
	displayModeSelect := widget.NewSelect(nil, nil)
	oledTypeSelect := widget.NewSelect(nil, nil)
	compassOrientationSelect := widget.NewSelect(nil, nil)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Always point north", compassNorthTopBox),
		widget.NewFormItem("Use 12-hour time format", use12HClockBox),
		widget.NewFormItem("Bold heading", headingBoldBox),
		widget.NewFormItem("Display units", unitsSelect),
		widget.NewFormItem("Screen on duration (seconds)", screenOnSecsEntry),
		widget.NewFormItem("Carousel interval (seconds)", carouselSecsEntry),
		widget.NewFormItem("Wake on tap or motion", wakeOnTapOrMotionBox),
		widget.NewFormItem("Flip screen", flipScreenBox),
		widget.NewFormItem("Display mode", displayModeSelect),
		widget.NewFormItem("OLED type", oledTypeSelect),
		widget.NewFormItem("Compass orientation", compassOrientationSelect),
	)

	var (
		baseline             app.NodeDisplaySettings
		baselineFormValues   nodeDisplaySettingsFormValues
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

	setForm := func(settings app.NodeDisplaySettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		compassNorthTopBox.SetChecked(settings.CompassNorthTop)
		use12HClockBox.SetChecked(settings.Use12HClock)
		headingBoldBox.SetChecked(settings.HeadingBold)
		nodeDisplaySetEnumSelect(unitsSelect, nodeDisplayUnitsOptions, settings.Units)
		screenOnSecsEntry.SetText(strconv.FormatUint(uint64(settings.ScreenOnSecs), 10))
		carouselSecsEntry.SetText(strconv.FormatUint(uint64(settings.AutoScreenCarouselSecs), 10))
		wakeOnTapOrMotionBox.SetChecked(settings.WakeOnTapOrMotion)
		flipScreenBox.SetChecked(settings.FlipScreen)
		nodeDisplaySetEnumSelect(displayModeSelect, nodeDisplayModeOptions, settings.DisplayMode)
		nodeDisplaySetEnumSelect(oledTypeSelect, nodeDisplayOledTypeOptions, settings.Oled)
		nodeDisplaySetEnumSelect(compassOrientationSelect, nodeDisplayCompassOrientationOptions, settings.CompassOrientation)
	}

	applyForm := func(settings app.NodeDisplaySettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeDisplaySettingsFormValues {
		return nodeDisplaySettingsFormValues{
			CompassNorthTop:        compassNorthTopBox.Checked,
			Use12HClock:            use12HClockBox.Checked,
			HeadingBold:            headingBoldBox.Checked,
			Units:                  strings.TrimSpace(unitsSelect.Selected),
			ScreenOnSecs:           strings.TrimSpace(screenOnSecsEntry.Text),
			AutoScreenCarouselSecs: strings.TrimSpace(carouselSecsEntry.Text),
			WakeOnTapOrMotion:      wakeOnTapOrMotionBox.Checked,
			FlipScreen:             flipScreenBox.Checked,
			DisplayMode:            strings.TrimSpace(displayModeSelect.Selected),
			OledType:               strings.TrimSpace(oledTypeSelect.Selected),
			CompassOrientation:     strings.TrimSpace(compassOrientationSelect.Selected),
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

	applyLoadedSettings := func(next app.NodeDisplaySettings) {
		mu.Lock()
		baseline = cloneNodeDisplaySettings(next)
		baselineFormValues = nodeDisplayFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeDisplaySettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeDisplaySettings, error) {
		screenOnSecs, err := parseNodeDisplayUint32Field("screen on duration", screenOnSecsEntry.Text)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}
		autoScreenCarouselSecs, err := parseNodeDisplayUint32Field("carousel interval", carouselSecsEntry.Text)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}
		units, err := nodeDisplayParseEnumLabel("display units", unitsSelect.Selected, nodeDisplayUnitsOptions)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}
		displayMode, err := nodeDisplayParseEnumLabel("display mode", displayModeSelect.Selected, nodeDisplayModeOptions)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}
		oled, err := nodeDisplayParseEnumLabel("OLED type", oledTypeSelect.Selected, nodeDisplayOledTypeOptions)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}
		compassOrientation, err := nodeDisplayParseEnumLabel("compass orientation", compassOrientationSelect.Selected, nodeDisplayCompassOrientationOptions)
		if err != nil {
			return app.NodeDisplaySettings{}, err
		}

		mu.Lock()
		next := cloneNodeDisplaySettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.CompassNorthTop = compassNorthTopBox.Checked
		next.Use12HClock = use12HClockBox.Checked
		next.HeadingBold = headingBoldBox.Checked
		next.Units = units
		next.ScreenOnSecs = screenOnSecs
		next.AutoScreenCarouselSecs = autoScreenCarouselSecs
		next.WakeOnTapOrMotion = wakeOnTapOrMotionBox.Checked
		next.FlipScreen = flipScreenBox.Checked
		next.DisplayMode = displayMode
		next.Oled = oled
		next.CompassOrientation = compassOrientation

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node display settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node display settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node display settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node display settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading display settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadDisplaySettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node display settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node display settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded display settings from device.", 2, 2)
			})
		}()
	}

	compassNorthTopBox.OnChanged = func(_ bool) { markDirty() }
	use12HClockBox.OnChanged = func(_ bool) { markDirty() }
	headingBoldBox.OnChanged = func(_ bool) { markDirty() }
	unitsSelect.OnChanged = func(_ string) { markDirty() }
	screenOnSecsEntry.OnChanged = func(_ string) { markDirty() }
	carouselSecsEntry.OnChanged = func(_ string) { markDirty() }
	wakeOnTapOrMotionBox.OnChanged = func(_ bool) { markDirty() }
	flipScreenBox.OnChanged = func(_ bool) { markDirty() }
	displayModeSelect.OnChanged = func(_ string) { markDirty() }
	oledTypeSelect.OnChanged = func(_ string) { markDirty() }
	compassOrientationSelect.OnChanged = func(_ string) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node display settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeDisplaySettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node display settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node display settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node display settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node display settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node display settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node display settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node display settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving display settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeDisplaySettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveDisplaySettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node display settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeDisplaySettings(settings)
				baselineFormValues = nodeDisplayFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeDisplaySettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node display settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved display settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node display settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeDisplaySettings{
		Units:              int32(generated.Config_DisplayConfig_METRIC),
		Oled:               int32(generated.Config_DisplayConfig_OLED_AUTO),
		DisplayMode:        int32(generated.Config_DisplayConfig_DEFAULT),
		CompassOrientation: int32(generated.Config_DisplayConfig_DEGREES_0),
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Display settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Display settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node display settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node display settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Display settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node display settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeDisplaySettingsFormValues struct {
	CompassNorthTop        bool
	Use12HClock            bool
	HeadingBold            bool
	Units                  string
	ScreenOnSecs           string
	AutoScreenCarouselSecs string
	WakeOnTapOrMotion      bool
	FlipScreen             bool
	DisplayMode            string
	OledType               string
	CompassOrientation     string
}

type nodeDisplayEnumOption struct {
	Label string
	Value int32
}

var nodeDisplayUnitsOptions = []nodeDisplayEnumOption{
	{Label: "Metric", Value: int32(generated.Config_DisplayConfig_METRIC)},
	{Label: "Imperial", Value: int32(generated.Config_DisplayConfig_IMPERIAL)},
}

var nodeDisplayModeOptions = []nodeDisplayEnumOption{
	{Label: "Default", Value: int32(generated.Config_DisplayConfig_DEFAULT)},
	{Label: "Two-color", Value: int32(generated.Config_DisplayConfig_TWOCOLOR)},
	{Label: "Inverted", Value: int32(generated.Config_DisplayConfig_INVERTED)},
	{Label: "Color", Value: int32(generated.Config_DisplayConfig_COLOR)},
}

var nodeDisplayOledTypeOptions = []nodeDisplayEnumOption{
	{Label: "Auto", Value: int32(generated.Config_DisplayConfig_OLED_AUTO)},
	{Label: "SSD1306", Value: int32(generated.Config_DisplayConfig_OLED_SSD1306)},
	{Label: "SH1106", Value: int32(generated.Config_DisplayConfig_OLED_SH1106)},
	{Label: "SH1107", Value: int32(generated.Config_DisplayConfig_OLED_SH1107)},
	{Label: "SH1107 128x128", Value: int32(generated.Config_DisplayConfig_OLED_SH1107_128_128)},
}

var nodeDisplayCompassOrientationOptions = []nodeDisplayEnumOption{
	{Label: "0 deg", Value: int32(generated.Config_DisplayConfig_DEGREES_0)},
	{Label: "90 deg", Value: int32(generated.Config_DisplayConfig_DEGREES_90)},
	{Label: "180 deg", Value: int32(generated.Config_DisplayConfig_DEGREES_180)},
	{Label: "270 deg", Value: int32(generated.Config_DisplayConfig_DEGREES_270)},
	{Label: "0 deg inverted", Value: int32(generated.Config_DisplayConfig_DEGREES_0_INVERTED)},
	{Label: "90 deg inverted", Value: int32(generated.Config_DisplayConfig_DEGREES_90_INVERTED)},
	{Label: "180 deg inverted", Value: int32(generated.Config_DisplayConfig_DEGREES_180_INVERTED)},
	{Label: "270 deg inverted", Value: int32(generated.Config_DisplayConfig_DEGREES_270_INVERTED)},
}

func nodeDisplayFormValuesFromSettings(settings app.NodeDisplaySettings) nodeDisplaySettingsFormValues {
	return nodeDisplaySettingsFormValues{
		CompassNorthTop:        settings.CompassNorthTop,
		Use12HClock:            settings.Use12HClock,
		HeadingBold:            settings.HeadingBold,
		Units:                  nodeDisplayEnumLabel(settings.Units, nodeDisplayUnitsOptions),
		ScreenOnSecs:           strconv.FormatUint(uint64(settings.ScreenOnSecs), 10),
		AutoScreenCarouselSecs: strconv.FormatUint(uint64(settings.AutoScreenCarouselSecs), 10),
		WakeOnTapOrMotion:      settings.WakeOnTapOrMotion,
		FlipScreen:             settings.FlipScreen,
		DisplayMode:            nodeDisplayEnumLabel(settings.DisplayMode, nodeDisplayModeOptions),
		OledType:               nodeDisplayEnumLabel(settings.Oled, nodeDisplayOledTypeOptions),
		CompassOrientation:     nodeDisplayEnumLabel(settings.CompassOrientation, nodeDisplayCompassOrientationOptions),
	}
}

func cloneNodeDisplaySettings(settings app.NodeDisplaySettings) app.NodeDisplaySettings {
	return settings
}

func parseNodeDisplayUint32Field(fieldName, raw string) (uint32, error) {
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

func nodeDisplaySetEnumSelect(selectWidget *widget.Select, options []nodeDisplayEnumOption, value int32) {
	optionLabels := nodeDisplayEnumOptionsLabels(options)
	selected := nodeDisplayEnumLabel(value, options)
	if strings.HasPrefix(selected, "Unknown (") {
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeDisplayEnumLabel(value int32, options []nodeDisplayEnumOption) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return nodeDisplayUnknownEnumLabel(value)
}

func nodeDisplayEnumOptionsLabels(options []nodeDisplayEnumOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func nodeDisplayUnknownEnumLabel(value int32) string {
	return fmt.Sprintf("Unknown (%d)", value)
}

func nodeDisplayParseEnumLabel(fieldName, selected string, options []nodeDisplayEnumOption) (int32, error) {
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
