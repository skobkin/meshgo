package ui

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

const (
	nodeMQTTDefaultAddress          = "mqtt.meshtastic.org"
	nodeMQTTAddressMaxLen           = 63
	nodeMQTTUsernameMaxLen          = 63
	nodeMQTTPasswordMaxLen          = 63
	nodeMQTTRootMaxLen              = 31
	nodeMQTTMinMapReportIntervalSec = 3600
	nodeMQTTMapPrecisionMin         = 12
	nodeMQTTMapPrecisionMax         = 15
	nodeMQTTMapPrecisionDefault     = 14
	nodeMQTTPrecisionMetersFactor   = 23905787.925008

	nodeMQTTMapInterval1Hour  uint32 = 1 * 60 * 60  // 1 hour
	nodeMQTTMapInterval2Hours uint32 = 2 * 60 * 60  // 2 hours
	nodeMQTTMapInterval3Hours uint32 = 3 * 60 * 60  // 3 hours
	nodeMQTTMapInterval4Hours uint32 = 4 * 60 * 60  // 4 hours
	nodeMQTTMapInterval5Hours uint32 = 5 * 60 * 60  // 5 hours
	nodeMQTTMapInterval6Hours uint32 = 6 * 60 * 60  // 6 hours
	nodeMQTTMapInterval12Hour uint32 = 12 * 60 * 60 // 12 hours
	nodeMQTTMapInterval18Hour uint32 = 18 * 60 * 60 // 18 hours
	nodeMQTTMapInterval24Hour uint32 = 24 * 60 * 60 // 24 hours
	nodeMQTTMapInterval36Hour uint32 = 36 * 60 * 60 // 36 hours
	nodeMQTTMapInterval48Hour uint32 = 48 * 60 * 60 // 48 hours
	nodeMQTTMapInterval72Hour uint32 = 72 * 60 * 60 // 72 hours
)

func newNodeMQTTSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "module.mqtt"
	nodeSettingsTabLogger.Debug("building node MQTT settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading MQTT settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	enabledBox := widget.NewCheck("", nil)
	addressEntry := widget.NewEntry()
	addressEntry.SetPlaceHolder(nodeMQTTDefaultAddress)
	usernameEntry := widget.NewEntry()
	passwordEntry := widget.NewEntry()
	passwordEntry.Password = true
	encryptionEnabledBox := widget.NewCheck("", nil)
	jsonEnabledBox := widget.NewCheck("", nil)
	tlsEnabledBox := widget.NewCheck("", nil)
	rootEntry := widget.NewEntry()
	rootEntry.SetPlaceHolder("msh")
	proxyToClientEnabledBox := widget.NewCheck("", nil)

	mapReportingEnabledBox := widget.NewCheck("Enabled", nil)
	mapReportShouldReportLocationBox := widget.NewCheck("Consent to share location", nil)
	mapReportPositionPrecisionSelect := widget.NewSelect(nil, nil)
	mapReportPublishIntervalSecsSelect := widget.NewSelect(nil, nil)

	mqttForm := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("MQTT enabled", enabledBox),
		widget.NewFormItem("Address", addressEntry),
		widget.NewFormItem("Username", usernameEntry),
		widget.NewFormItem("Password", passwordEntry),
		widget.NewFormItem("Encryption enabled", encryptionEnabledBox),
		widget.NewFormItem("JSON output enabled", jsonEnabledBox),
		widget.NewFormItem("TLS enabled", tlsEnabledBox),
		widget.NewFormItem("Root topic", rootEntry),
		widget.NewFormItem("Proxy to client enabled", proxyToClientEnabledBox),
	)
	mapReportingForm := widget.NewForm(
		widget.NewFormItem("Map reporting", container.NewGridWithColumns(2, mapReportingEnabledBox, mapReportShouldReportLocationBox)),
	)
	mapReportingDetailsForm := widget.NewForm(
		widget.NewFormItem("Position precision", mapReportPositionPrecisionSelect),
		widget.NewFormItem("Publish interval", mapReportPublishIntervalSecsSelect),
	)
	mapReportingContent := container.NewVBox(
		widget.NewLabel("Map reporting"),
		mapReportingForm,
		mapReportingDetailsForm,
	)

	var (
		baseline              app.NodeMQTTSettings
		baselineFormValues    nodeMQTTSettingsFormValues
		dirty                 bool
		saving                bool
		initialReloadStarted  atomic.Bool
		mu                    sync.Mutex
		applyingForm          atomic.Bool
		mapDetailsVisibleInit bool
		mapDetailsVisible     bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}

	setForm := func(settings app.NodeMQTTSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		enabledBox.SetChecked(settings.Enabled)
		addressEntry.SetText(settings.Address)
		usernameEntry.SetText(settings.Username)
		passwordEntry.SetText(settings.Password)
		encryptionEnabledBox.SetChecked(settings.EncryptionEnabled)
		jsonEnabledBox.SetChecked(settings.JSONEnabled)
		tlsEnabledBox.SetChecked(settings.TLSEnabled)
		rootEntry.SetText(settings.Root)
		proxyToClientEnabledBox.SetChecked(settings.ProxyToClientEnabled)
		mapReportingEnabledBox.SetChecked(settings.MapReportingEnabled)
		mapReportShouldReportLocationBox.SetChecked(settings.MapReportShouldReportLocation)
		nodeMQTTSetMapPrecisionSelect(mapReportPositionPrecisionSelect, settings.MapReportPositionPrecision)
		nodeMQTTSetMapIntervalSelect(mapReportPublishIntervalSecsSelect, settings.MapReportPublishIntervalSecs)
	}

	applyForm := func(settings app.NodeMQTTSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodeMQTTSettingsFormValues {
		return nodeMQTTSettingsFormValues{
			Enabled:                       enabledBox.Checked,
			Address:                       addressEntry.Text,
			Username:                      usernameEntry.Text,
			Password:                      passwordEntry.Text,
			EncryptionEnabled:             encryptionEnabledBox.Checked,
			JSONEnabled:                   jsonEnabledBox.Checked,
			TLSEnabled:                    tlsEnabledBox.Checked,
			Root:                          rootEntry.Text,
			ProxyToClientEnabled:          proxyToClientEnabledBox.Checked,
			MapReportingEnabled:           mapReportingEnabledBox.Checked,
			MapReportShouldReportLocation: mapReportShouldReportLocationBox.Checked,
			MapReportPositionPrecision:    strings.TrimSpace(mapReportPositionPrecisionSelect.Selected),
			MapReportPublishIntervalSecs:  strings.TrimSpace(mapReportPublishIntervalSecsSelect.Selected),
		}
	}

	updateFieldAvailability := func() {
		mu.Lock()
		isSaving := saving
		mu.Unlock()

		enforceTLS := nodeMQTTTLSRequired(addressEntry.Text, proxyToClientEnabledBox.Checked)
		if enforceTLS && !tlsEnabledBox.Checked {
			applyingForm.Store(true)
			tlsEnabledBox.SetChecked(true)
			applyingForm.Store(false)
		}
		if isSaving || enforceTLS {
			tlsEnabledBox.Disable()
		} else {
			tlsEnabledBox.Enable()
		}

		mapReportingEnabled := mapReportingEnabledBox.Checked
		showMapDetails := mapReportingEnabled && mapReportShouldReportLocationBox.Checked

		if !isSaving && mapReportingEnabled {
			mapReportShouldReportLocationBox.Enable()
		} else {
			mapReportShouldReportLocationBox.Disable()
		}

		if !isSaving && showMapDetails {
			mapReportPositionPrecisionSelect.Enable()
			mapReportPublishIntervalSecsSelect.Enable()
		} else {
			mapReportPositionPrecisionSelect.Disable()
			mapReportPublishIntervalSecsSelect.Disable()
		}
		if showMapDetails {
			mapReportingDetailsForm.Show()
		} else {
			mapReportingDetailsForm.Hide()
		}

		needsRefresh := false
		if !mapDetailsVisibleInit || mapDetailsVisible != showMapDetails {
			mapDetailsVisibleInit = true
			mapDetailsVisible = showMapDetails
			needsRefresh = true
		}
		if needsRefresh {
			mapReportingContent.Refresh()
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

	applyLoadedSettings := func(next app.NodeMQTTSettings) {
		mu.Lock()
		baseline = cloneNodeMQTTSettings(next)
		baselineFormValues = nodeMQTTFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodeMQTTSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeMQTTSettings, error) {

		address := addressEntry.Text
		if err := validateNodeMQTTTextFieldLen("address", address, nodeMQTTAddressMaxLen); err != nil {
			return app.NodeMQTTSettings{}, err
		}
		username := usernameEntry.Text
		if err := validateNodeMQTTTextFieldLen("username", username, nodeMQTTUsernameMaxLen); err != nil {
			return app.NodeMQTTSettings{}, err
		}
		password := passwordEntry.Text
		if err := validateNodeMQTTTextFieldLen("password", password, nodeMQTTPasswordMaxLen); err != nil {
			return app.NodeMQTTSettings{}, err
		}
		root := rootEntry.Text
		if err := validateNodeMQTTTextFieldLen("root topic", root, nodeMQTTRootMaxLen); err != nil {
			return app.NodeMQTTSettings{}, err
		}

		mu.Lock()
		next := cloneNodeMQTTSettings(baseline)
		mu.Unlock()
		mapReportPositionPrecision := next.MapReportPositionPrecision
		if mapReportPositionPrecision == 0 {
			mapReportPositionPrecision = nodeMQTTMapPrecisionDefault
		}
		parsedMapPrecision, err := nodeMQTTParseMapPrecisionLabel("position precision", mapReportPositionPrecisionSelect.Selected)
		if err == nil {
			mapReportPositionPrecision = parsedMapPrecision
		} else if strings.TrimSpace(mapReportPositionPrecisionSelect.Selected) != "" {
			return app.NodeMQTTSettings{}, err
		}
		mapReportPublishIntervalSecs := next.MapReportPublishIntervalSecs
		if mapReportPublishIntervalSecs == 0 {
			mapReportPublishIntervalSecs = nodeMQTTMinMapReportIntervalSec
		}
		parsedMapInterval, err := nodeMQTTParseMapIntervalLabel("map reporting publish interval", mapReportPublishIntervalSecsSelect.Selected)
		if err == nil {
			mapReportPublishIntervalSecs = parsedMapInterval
		} else if strings.TrimSpace(mapReportPublishIntervalSecsSelect.Selected) != "" {
			return app.NodeMQTTSettings{}, err
		}

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.Enabled = enabledBox.Checked
		next.Address = address
		next.Username = username
		next.Password = password
		next.EncryptionEnabled = encryptionEnabledBox.Checked
		next.JSONEnabled = jsonEnabledBox.Checked
		next.TLSEnabled = tlsEnabledBox.Checked
		next.Root = root
		next.ProxyToClientEnabled = proxyToClientEnabledBox.Checked
		next.MapReportingEnabled = mapReportingEnabledBox.Checked
		next.MapReportShouldReportLocation = mapReportShouldReportLocationBox.Checked
		next.MapReportPositionPrecision = mapReportPositionPrecision
		next.MapReportPublishIntervalSecs = mapReportPublishIntervalSecs

		if next.MapReportingEnabled {
			if !next.MapReportShouldReportLocation {
				return app.NodeMQTTSettings{}, fmt.Errorf("map reporting requires location consent")
			}
			if next.MapReportPublishIntervalSecs < nodeMQTTMinMapReportIntervalSec {
				return app.NodeMQTTSettings{}, fmt.Errorf("map reporting publish interval must be at least %d seconds", nodeMQTTMinMapReportIntervalSec)
			}
			if next.MapReportPositionPrecision < nodeMQTTMapPrecisionMin || next.MapReportPositionPrecision > nodeMQTTMapPrecisionMax {
				return app.NodeMQTTSettings{}, fmt.Errorf("position precision must be between %d and %d", nodeMQTTMapPrecisionMin, nodeMQTTMapPrecisionMax)
			}
		}

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node MQTT settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node MQTT settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node MQTT settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node MQTT settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading MQTT settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadMQTTSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node MQTT settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				if loaded.MapReportPositionPrecision == 0 {
					loaded.MapReportPositionPrecision = nodeMQTTMapPrecisionDefault
				}
				if loaded.MapReportPublishIntervalSecs == 0 {
					loaded.MapReportPublishIntervalSecs = nodeMQTTMinMapReportIntervalSec
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node MQTT settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded MQTT settings from device.", 2, 2)
			})
		}()
	}

	enabledBox.OnChanged = func(_ bool) { markDirty() }
	addressEntry.OnChanged = func(_ string) {
		markDirty()
		updateFieldAvailability()
	}
	usernameEntry.OnChanged = func(_ string) { markDirty() }
	passwordEntry.OnChanged = func(_ string) { markDirty() }
	encryptionEnabledBox.OnChanged = func(_ bool) { markDirty() }
	jsonEnabledBox.OnChanged = func(_ bool) { markDirty() }
	tlsEnabledBox.OnChanged = func(_ bool) { markDirty() }
	rootEntry.OnChanged = func(_ string) { markDirty() }
	proxyToClientEnabledBox.OnChanged = func(_ bool) {
		markDirty()
		updateFieldAvailability()
	}
	mapReportingEnabledBox.OnChanged = func(_ bool) {
		markDirty()
		updateFieldAvailability()
	}
	mapReportShouldReportLocationBox.OnChanged = func(_ bool) {
		markDirty()
		updateFieldAvailability()
	}
	mapReportPositionPrecisionSelect.OnChanged = func(_ string) { markDirty() }
	mapReportPublishIntervalSecsSelect.OnChanged = func(_ string) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node MQTT settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeMQTTSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node MQTT settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node MQTT settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node MQTT settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node MQTT settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node MQTT settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node MQTT settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node MQTT settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving MQTT settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeMQTTSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveMQTTSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node MQTT settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeMQTTSettings(settings)
				baselineFormValues = nodeMQTTFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodeMQTTSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node MQTT settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved MQTT settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node MQTT settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeMQTTSettings{
		MapReportPositionPrecision:   nodeMQTTMapPrecisionDefault,
		MapReportPublishIntervalSecs: nodeMQTTMinMapReportIntervalSec,
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("MQTT settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("MQTT settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node MQTT settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node MQTT settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("MQTT module settings are loaded from and saved to the connected local node."),
		mqttForm,
		mapReportingContent,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node MQTT settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodeMQTTSettingsFormValues struct {
	Enabled                       bool
	Address                       string
	Username                      string
	Password                      string
	EncryptionEnabled             bool
	JSONEnabled                   bool
	TLSEnabled                    bool
	Root                          string
	ProxyToClientEnabled          bool
	MapReportingEnabled           bool
	MapReportShouldReportLocation bool
	MapReportPositionPrecision    string
	MapReportPublishIntervalSecs  string
}

func nodeMQTTFormValuesFromSettings(settings app.NodeMQTTSettings) nodeMQTTSettingsFormValues {
	return nodeMQTTSettingsFormValues{
		Enabled:                       settings.Enabled,
		Address:                       settings.Address,
		Username:                      settings.Username,
		Password:                      settings.Password,
		EncryptionEnabled:             settings.EncryptionEnabled,
		JSONEnabled:                   settings.JSONEnabled,
		TLSEnabled:                    settings.TLSEnabled,
		Root:                          settings.Root,
		ProxyToClientEnabled:          settings.ProxyToClientEnabled,
		MapReportingEnabled:           settings.MapReportingEnabled,
		MapReportShouldReportLocation: settings.MapReportShouldReportLocation,
		MapReportPositionPrecision:    nodeMQTTMapPrecisionSelectLabel(settings.MapReportPositionPrecision),
		MapReportPublishIntervalSecs:  nodeMQTTMapIntervalSelectLabel(settings.MapReportPublishIntervalSecs),
	}
}

func cloneNodeMQTTSettings(settings app.NodeMQTTSettings) app.NodeMQTTSettings {
	return settings
}

type nodeMQTTUint32Option struct {
	Label string
	Value uint32
}

var nodeMQTTMapPrecisionOptions = func() []nodeMQTTUint32Option {
	out := make([]nodeMQTTUint32Option, 0, nodeMQTTMapPrecisionMax-nodeMQTTMapPrecisionMin+1)
	for bits := nodeMQTTMapPrecisionMin; bits <= nodeMQTTMapPrecisionMax; bits++ {
		bitsValue := uint32(bits)
		out = append(out, nodeMQTTUint32Option{
			Label: nodeMQTTMapPrecisionKnownLabel(bitsValue),
			Value: bitsValue,
		})
	}

	return out
}()

var nodeMQTTMapIntervalOptions = []nodeMQTTUint32Option{
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval1Hour), Value: nodeMQTTMapInterval1Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval2Hours), Value: nodeMQTTMapInterval2Hours},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval3Hours), Value: nodeMQTTMapInterval3Hours},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval4Hours), Value: nodeMQTTMapInterval4Hours},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval5Hours), Value: nodeMQTTMapInterval5Hours},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval6Hours), Value: nodeMQTTMapInterval6Hours},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval12Hour), Value: nodeMQTTMapInterval12Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval18Hour), Value: nodeMQTTMapInterval18Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval24Hour), Value: nodeMQTTMapInterval24Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval36Hour), Value: nodeMQTTMapInterval36Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval48Hour), Value: nodeMQTTMapInterval48Hour},
	{Label: nodeMQTTMapIntervalKnownLabel(nodeMQTTMapInterval72Hour), Value: nodeMQTTMapInterval72Hour},
}

func nodeMQTTSetMapPrecisionSelect(selectWidget *widget.Select, value uint32) {
	if value == 0 {
		value = nodeMQTTMapPrecisionDefault
	}
	nodeMQTTSetUint32Select(selectWidget, nodeMQTTMapPrecisionOptions, value, nodeMQTTMapPrecisionCustomLabel)
}

func nodeMQTTSetMapIntervalSelect(selectWidget *widget.Select, value uint32) {
	if value == 0 {
		value = nodeMQTTMinMapReportIntervalSec
	}
	nodeMQTTSetUint32Select(selectWidget, nodeMQTTMapIntervalOptions, value, nodeMQTTMapIntervalCustomLabel)
}

func nodeMQTTSetUint32Select(
	selectWidget *widget.Select,
	options []nodeMQTTUint32Option,
	value uint32,
	customLabel func(uint32) string,
) {
	optionLabels := nodeMQTTUint32OptionLabels(options)
	selected := nodeMQTTUint32OptionLabel(value, options)
	if selected == "" {
		selected = customLabel(value)
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeMQTTMapPrecisionSelectLabel(value uint32) string {
	if value == 0 {
		value = nodeMQTTMapPrecisionDefault
	}
	label := nodeMQTTUint32OptionLabel(value, nodeMQTTMapPrecisionOptions)
	if label != "" {
		return label
	}

	return nodeMQTTMapPrecisionCustomLabel(value)
}

func nodeMQTTMapIntervalSelectLabel(value uint32) string {
	if value == 0 {
		value = nodeMQTTMinMapReportIntervalSec
	}
	label := nodeMQTTUint32OptionLabel(value, nodeMQTTMapIntervalOptions)
	if label != "" {
		return label
	}

	return nodeMQTTMapIntervalCustomLabel(value)
}

func nodeMQTTMapPrecisionKnownLabel(bits uint32) string {
	return fmt.Sprintf("%s (%d bits)", nodeMQTTFormatMetricDistance(nodeMQTTPrecisionBitsToMeters(bits)), bits)
}

func nodeMQTTMapPrecisionCustomLabel(bits uint32) string {
	return fmt.Sprintf("Custom (%d bits)", bits)
}

func nodeMQTTMapIntervalKnownLabel(seconds uint32) string {
	if seconds%3600 == 0 {
		hours := seconds / 3600
		if hours == 1 {
			return "1 hour"
		}

		return fmt.Sprintf("%d hours", hours)
	}
	if seconds%60 == 0 {
		minutes := seconds / 60
		if minutes == 1 {
			return "1 minute"
		}

		return fmt.Sprintf("%d minutes", minutes)
	}
	if seconds == 1 {
		return "1 second"
	}

	return fmt.Sprintf("%d seconds", seconds)
}

func nodeMQTTMapIntervalCustomLabel(seconds uint32) string {
	return fmt.Sprintf("Custom (%d seconds)", seconds)
}

func nodeMQTTParseMapPrecisionLabel(fieldName, selected string) (uint32, error) {
	return nodeMQTTParseUint32SelectLabel(fieldName, selected, nodeMQTTMapPrecisionOptions, "Custom (", " bits)")
}

func nodeMQTTParseMapIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeMQTTParseUint32SelectLabel(fieldName, selected, nodeMQTTMapIntervalOptions, "Custom (", " seconds)")
}

func nodeMQTTParseUint32SelectLabel(
	fieldName, selected string,
	options []nodeMQTTUint32Option,
	customPrefix, customSuffix string,
) (uint32, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return 0, fmt.Errorf("%s must be selected", fieldName)
	}
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}
	if strings.HasPrefix(selected, customPrefix) && strings.HasSuffix(selected, customSuffix) {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, customPrefix), customSuffix)
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return uint32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}

func nodeMQTTPrecisionBitsToMeters(bits uint32) float64 {
	return nodeMQTTPrecisionMetersFactor * math.Pow(0.5, float64(bits))
}

func nodeMQTTFormatMetricDistance(meters float64) string {
	if meters >= 1000 {
		return fmt.Sprintf("%.1f km", meters/1000)
	}

	return fmt.Sprintf("%.0f m", meters)
}

func nodeMQTTUint32OptionLabel(value uint32, options []nodeMQTTUint32Option) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return ""
}

func nodeMQTTUint32OptionLabels(options []nodeMQTTUint32Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func validateNodeMQTTTextFieldLen(fieldName, value string, maxLen int) error {
	if utf8.RuneCountInString(value) > maxLen {
		return fmt.Errorf("%s must be at most %d characters", fieldName, maxLen)
	}

	return nil
}

func nodeMQTTTLSRequired(address string, proxyToClientEnabled bool) bool {
	address = strings.TrimSpace(strings.ToLower(address))
	isDefaultAddress := address == "" || strings.Contains(address, nodeMQTTDefaultAddress)

	return isDefaultAddress && proxyToClientEnabled
}
