package ui

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

const nodeSettingsOpTimeout = 12 * time.Second

var nodeSettingsTabLogger = slog.With("component", "ui.node_settings_tab")

type nodeSettingsSaveGate struct {
	mu   sync.Mutex
	page string
}

func (g *nodeSettingsSaveGate) TryAcquire(page string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if strings.TrimSpace(g.page) != "" {
		return false
	}
	g.page = strings.TrimSpace(page)

	return true
}

func (g *nodeSettingsSaveGate) Release(page string) {
	g.mu.Lock()
	if strings.TrimSpace(g.page) == strings.TrimSpace(page) {
		g.page = ""
	}
	g.mu.Unlock()
}

func (g *nodeSettingsSaveGate) ActivePage() string {
	g.mu.Lock()
	active := g.page
	g.mu.Unlock()

	return active
}

type nodeSettingsPageControls struct {
	saveButton   *widget.Button
	cancelButton *widget.Button
	reloadButton *widget.Button
	statusLabel  *widget.Label
	progressBar  *widget.ProgressBar
	root         fyne.CanvasObject
}

func newNodeSettingsPageControls(initialStatus string) *nodeSettingsPageControls {
	status := widget.NewLabel(strings.TrimSpace(initialStatus))
	status.Wrapping = fyne.TextWrapWord

	progress := widget.NewProgressBar()
	progress.SetValue(0)

	saveButton := widget.NewButton("Save", nil)
	cancelButton := widget.NewButton("Cancel", nil)
	reloadButton := widget.NewButton("Reload", nil)

	buttons := container.NewHBox(reloadButton, layout.NewSpacer(), cancelButton, saveButton)
	root := container.NewVBox(
		widget.NewSeparator(),
		progress,
		status,
		buttons,
	)

	return &nodeSettingsPageControls{
		saveButton:   saveButton,
		cancelButton: cancelButton,
		reloadButton: reloadButton,
		statusLabel:  status,
		progressBar:  progress,
		root:         root,
	}
}

func (c *nodeSettingsPageControls) SetStatus(text string, completed, total int) {
	if c == nil {
		return
	}
	if strings.TrimSpace(text) != "" {
		c.statusLabel.SetText(text)
	}
	c.progressBar.SetValue(nodeSettingsProgress(completed, total))
}

func nodeSettingsProgress(completed, total int) float64 {
	if total <= 0 {
		return 0
	}
	if completed <= 0 {
		return 0
	}
	if completed >= total {
		return 1
	}

	return float64(completed) / float64(total)
}

func wrapNodeSettingsPage(content fyne.CanvasObject, controls *nodeSettingsPageControls) fyne.CanvasObject {
	if controls == nil {
		return content
	}

	return container.NewBorder(nil, controls.root, nil, nil, container.NewVScroll(content))
}

func newNodeTab(dep RuntimeDependencies) fyne.CanvasObject {
	nodeSettingsTabLogger.Info("building node settings tab")
	saveGate := &nodeSettingsSaveGate{}
	securityPage, onSecurityTabOpened := newNodeSecuritySettingsPage(dep, saveGate)
	securityTab := container.NewTabItem("Security", securityPage)

	radioTabs := container.NewAppTabs(
		container.NewTabItem("LoRa", newSettingsPlaceholderPage("LoRa settings editing is planned.")),
		container.NewTabItem("Channels", newSettingsPlaceholderPage("Channels editor is planned.")),
		securityTab,
	)
	radioTabs.SetTabLocation(container.TabLocationTop)
	radioTabs.OnSelected = func(item *container.TabItem) {
		if onSecurityTabOpened == nil || item != securityTab {
			return
		}
		onSecurityTabOpened()
	}

	deviceTabs := container.NewAppTabs(
		container.NewTabItem("User", newNodeUserSettingsPage(dep, saveGate)),
		container.NewTabItem("Device", newSettingsPlaceholderPage("Device settings editing is planned.")),
		container.NewTabItem("Position", newSettingsPlaceholderPage("Position settings editing is planned.")),
		container.NewTabItem("Power", newSettingsPlaceholderPage("Power settings editing is planned.")),
		container.NewTabItem("Display", newSettingsPlaceholderPage("Display settings editing is planned.")),
		container.NewTabItem("Bluetooth", newSettingsPlaceholderPage("Bluetooth settings editing is planned.")),
	)
	deviceTabs.SetTabLocation(container.TabLocationTop)

	moduleTabs := container.NewAppTabs(
		container.NewTabItem("MQTT", newSettingsPlaceholderPage("MQTT module settings editing is planned.")),
		container.NewTabItem("Serial", newSettingsPlaceholderPage("Serial module settings editing is planned.")),
		container.NewTabItem("External notification", newSettingsPlaceholderPage("External notification module settings editing is planned.")),
		container.NewTabItem("Store & Forward", newSettingsPlaceholderPage("Store & Forward module settings editing is planned.")),
		container.NewTabItem("Range test", newSettingsPlaceholderPage("Range test module settings editing is planned.")),
		container.NewTabItem("Telemetry", newSettingsPlaceholderPage("Telemetry module settings editing is planned.")),
		container.NewTabItem("Neighbor Info", newSettingsPlaceholderPage("Neighbor Info module settings editing is planned.")),
		container.NewTabItem("Status Message", newSettingsPlaceholderPage("Status Message module settings editing is planned.")),
	)
	moduleTabs.SetTabLocation(container.TabLocationTop)

	importExportTab := newDisabledTopLevelPage("Import/Export is planned and currently disabled.")
	maintenanceTab := newDisabledTopLevelPage("Maintenance is planned and currently disabled.")

	topTabs := container.NewAppTabs(
		container.NewTabItem("Radio configuration", radioTabs),
		container.NewTabItem("Device configuration", deviceTabs),
		container.NewTabItem("Module configuration", moduleTabs),
		container.NewTabItem("Import/Export", importExportTab),
		container.NewTabItem("Maintenance", maintenanceTab),
	)
	topTabs.SetTabLocation(container.TabLocationTop)
	topTabs.DisableIndex(3)
	topTabs.DisableIndex(4)

	return topTabs
}

func newNodeUserSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) fyne.CanvasObject {
	const pageID = "device.user"
	nodeSettingsTabLogger.Debug("building node user settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading local node user settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	longNameEntry := widget.NewEntry()
	shortNameEntry := widget.NewEntry()
	hamLicensedBox := widget.NewCheck("", nil)
	unmessageableBox := widget.NewCheck("", nil)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Long Name", longNameEntry),
		widget.NewFormItem("Short Name", shortNameEntry),
		widget.NewFormItem("Licensed amateur radio (HAM)", hamLicensedBox),
		widget.NewFormItem("Unmessageable", unmessageableBox),
	)

	var (
		current      app.NodeUserSettings
		baseline     app.NodeUserSettings
		dirty        bool
		saving       bool
		mu           sync.Mutex
		applyingForm atomic.Bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}

	setForm := func(settings app.NodeUserSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		longNameEntry.SetText(settings.LongName)
		shortNameEntry.SetText(settings.ShortName)
		hamLicensedBox.SetChecked(settings.HamLicensed)
		unmessageableBox.SetChecked(settings.IsUnmessageable)
	}

	currentFromForm := func() app.NodeUserSettings {
		return app.NodeUserSettings{
			NodeID:          strings.TrimSpace(nodeIDLabel.Text),
			LongName:        strings.TrimSpace(longNameEntry.Text),
			ShortName:       strings.TrimSpace(shortNameEntry.Text),
			HamLicensed:     hamLicensedBox.Checked,
			IsUnmessageable: unmessageableBox.Checked,
		}
	}

	applyForm := func(settings app.NodeUserSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
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

	refreshFromLocalStore := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Debug("local node settings are unavailable: local node ID is unknown")
			controls.SetStatus("Local node ID is not available yet.", 0, 1)

			return
		}
		next := app.NodeUserSettings{
			NodeID: target.NodeID,
		}
		if dep.Data.NodeStore != nil {
			node, known := localNodeSnapshot(dep.Data.NodeStore, dep.Data.LocalNodeID)
			if known {
				next.LongName = strings.TrimSpace(node.LongName)
				next.ShortName = strings.TrimSpace(node.ShortName)
				next.IsUnmessageable = node.IsUnmessageable != nil && *node.IsUnmessageable
			}
		}

		shouldApply := false
		mu.Lock()
		if !dirty && !saving {
			baseline = next
			current = next
			shouldApply = true
		}
		mu.Unlock()
		if shouldApply {
			applyForm(next)
			nodeSettingsTabLogger.Debug("loaded node user settings from local store", "node_id", next.NodeID)
			controls.SetStatus("Loaded local node user settings.", 1, 1)
		}
		updateButtons()
	}

	markDirty := func() {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		current = currentFromForm()
		dirty = current != baseline
		mu.Unlock()
		updateButtons()
	}

	longNameEntry.OnChanged = func(_ string) { markDirty() }
	shortNameEntry.OnChanged = func(_ string) { markDirty() }
	hamLicensedBox.OnChanged = func(_ bool) { markDirty() }
	unmessageableBox.OnChanged = func(_ bool) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node user settings edit canceled", "page_id", pageID)
		var settings app.NodeUserSettings
		mu.Lock()
		settings = baseline
		current = settings
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node user settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node user settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node user settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node user settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node user settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		current = currentFromForm()
		next := current
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node user settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving user settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeUserSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveUserSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node user settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
				} else {
					nodeSettingsTabLogger.Info("saved node user settings", "page_id", pageID, "node_id", target.NodeID)
					baseline = settings
					current = settings
					dirty = false
					controls.SetStatus("Saved user settings.", 3, 3)
				}
				mu.Unlock()
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node user settings reload requested", "page_id", pageID)
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node user settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Info("node settings service is unavailable: reloading from local store", "page_id", pageID, "node_id", target.NodeID)
			refreshFromLocalStore()

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node user settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node user settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading user settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadUserSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node user settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				var settings app.NodeUserSettings
				mu.Lock()
				baseline = loaded
				current = loaded
				dirty = false
				settings = current
				mu.Unlock()
				applyForm(settings)
				nodeSettingsTabLogger.Info("reloaded node user settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded user settings from device.", 2, 2)
				updateButtons()
			})
		}()
	}

	refreshFromLocalStore()

	if dep.Data.NodeStore != nil {
		nodeSettingsTabLogger.Debug("starting node settings page listener for local node store changes", "page_id", pageID)
		go func() {
			for range dep.Data.NodeStore.Changes() {
				fyne.Do(func() {
					refreshFromLocalStore()
				})
			}
		}()
	}
	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("User settings can be edited and saved per page. Only one settings save can run at a time."),
		form,
	)

	return wrapNodeSettingsPage(content, controls)
}

func newNodeSecuritySettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "radio.security"
	nodeSettingsTabLogger.Debug("building node security settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading security settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	publicKeyEntry := widget.NewEntry()
	publicKeyEntry.Disable()
	privateKeyEntry := widget.NewEntry()
	privateKeyEntry.Password = true
	privateKeyEntry.Disable()
	publicKeyCopyButton := widget.NewButton("Copy", nil)
	privateKeyCopyButton := widget.NewButton("Copy", nil)
	adminKeysEntry := widget.NewMultiLineEntry()
	adminKeysEntry.SetMinRowsVisible(3)

	managedBox := widget.NewCheck("", nil)
	serialEnabledBox := widget.NewCheck("", nil)
	debugLogAPIBox := widget.NewCheck("", nil)
	adminChannelBox := widget.NewCheck("", nil)

	publicKeyField := container.NewBorder(nil, nil, nil, publicKeyCopyButton, publicKeyEntry)
	privateKeyField := container.NewBorder(nil, nil, nil, privateKeyCopyButton, privateKeyEntry)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Public key (read-only)", publicKeyField),
		widget.NewFormItem("Private key (read-only)", privateKeyField),
		widget.NewFormItem("Admin keys (base64, one key per line)", adminKeysEntry),
		widget.NewFormItem("Managed mode", managedBox),
		widget.NewFormItem("Serial console over Stream API", serialEnabledBox),
		widget.NewFormItem("Debug log over API", debugLogAPIBox),
		widget.NewFormItem("Legacy admin channel", adminChannelBox),
	)

	var (
		baseline             app.NodeSecuritySettings
		baselineAdminKeysRaw string
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

	updateKeyCopyButtons := func() {
		if strings.TrimSpace(publicKeyEntry.Text) == "" {
			publicKeyCopyButton.Disable()
		} else {
			publicKeyCopyButton.Enable()
		}
		if strings.TrimSpace(privateKeyEntry.Text) == "" {
			privateKeyCopyButton.Disable()
		} else {
			privateKeyCopyButton.Enable()
		}
	}

	setForm := func(settings app.NodeSecuritySettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		publicKeyEntry.SetText(encodeSecurityKey(settings.PublicKey))
		privateKeyEntry.SetText(encodeSecurityKey(settings.PrivateKey))
		adminKeysEntry.SetText(formatSecurityAdminKeys(settings.AdminKeys))
		managedBox.SetChecked(settings.IsManaged)
		serialEnabledBox.SetChecked(settings.SerialEnabled)
		debugLogAPIBox.SetChecked(settings.DebugLogAPIEnabled)
		adminChannelBox.SetChecked(settings.AdminChannelEnabled)
		updateKeyCopyButtons()
	}

	applyForm := func(settings app.NodeSecuritySettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	updateManagedBoxAvailability := func() {
		if saving {
			managedBox.Disable()

			return
		}
		hasAdminKeys := strings.TrimSpace(adminKeysEntry.Text) != ""
		if hasAdminKeys || managedBox.Checked {
			managedBox.Enable()
		} else {
			managedBox.Disable()
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
		updateManagedBoxAvailability()
	}

	markDirty := func() {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		dirty = managedBox.Checked != baseline.IsManaged ||
			serialEnabledBox.Checked != baseline.SerialEnabled ||
			debugLogAPIBox.Checked != baseline.DebugLogAPIEnabled ||
			adminChannelBox.Checked != baseline.AdminChannelEnabled ||
			strings.TrimSpace(adminKeysEntry.Text) != strings.TrimSpace(baselineAdminKeysRaw)
		mu.Unlock()
		updateButtons()
	}

	applyLoadedSettings := func(next app.NodeSecuritySettings) {
		mu.Lock()
		baseline = cloneNodeSecuritySettings(next)
		baselineAdminKeysRaw = formatSecurityAdminKeys(next.AdminKeys)
		dirty = false
		settings := cloneNodeSecuritySettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodeSecuritySettings, error) {
		adminKeys, err := parseSecurityAdminKeysInput(adminKeysEntry.Text)
		if err != nil {
			return app.NodeSecuritySettings{}, err
		}
		if managedBox.Checked && len(adminKeys) == 0 {
			return app.NodeSecuritySettings{}, fmt.Errorf("managed mode requires at least one admin key")
		}

		mu.Lock()
		next := cloneNodeSecuritySettings(baseline)
		mu.Unlock()

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.AdminKeys = cloneSecurityKeys(adminKeys)
		next.IsManaged = managedBox.Checked
		next.SerialEnabled = serialEnabledBox.Checked
		next.DebugLogAPIEnabled = debugLogAPIBox.Checked
		next.AdminChannelEnabled = adminChannelBox.Checked

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node security settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node security settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node security settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node security settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading security settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadSecuritySettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node security settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node security settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded security settings from device.", 2, 2)
			})
		}()
	}

	adminKeysEntry.OnChanged = func(_ string) { markDirty() }
	managedBox.OnChanged = func(_ bool) { markDirty() }
	serialEnabledBox.OnChanged = func(_ bool) { markDirty() }
	debugLogAPIBox.OnChanged = func(_ bool) { markDirty() }
	adminChannelBox.OnChanged = func(_ bool) { markDirty() }
	publicKeyCopyButton.OnTapped = func() {
		if err := copyTextToClipboard(publicKeyEntry.Text); err != nil {
			controls.SetStatus("Copy failed: "+err.Error(), 0, 1)

			return
		}
		controls.SetStatus("Public key copied.", 1, 1)
	}
	privateKeyCopyButton.OnTapped = func() {
		if err := copyTextToClipboard(privateKeyEntry.Text); err != nil {
			controls.SetStatus("Copy failed: "+err.Error(), 0, 1)

			return
		}
		controls.SetStatus("Private key copied.", 1, 1)
	}

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node security settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodeSecuritySettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node security settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node security settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node security settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node security settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node security settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node security settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node security settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving security settings…", 1, 3)
		updateButtons()

		go func(settings app.NodeSecuritySettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveSecuritySettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node security settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodeSecuritySettings(settings)
				baselineAdminKeysRaw = formatSecurityAdminKeys(settings.AdminKeys)
				dirty = false
				applied := cloneNodeSecuritySettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node security settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved security settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node security settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodeSecuritySettings{}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Security settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Security settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node security settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node security settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	adminKeysHint := widget.NewLabel("Use base64-encoded admin public keys, one key per line. Up to 3 keys are supported.")
	adminKeysHint.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		widget.NewLabel("Security settings are loaded from and saved to the connected local node."),
		adminKeysHint,
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node security settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

func newSettingsPlaceholderPage(text string) fyne.CanvasObject {
	controls := newNodeSettingsPageControls(text)
	controls.saveButton.Disable()
	controls.cancelButton.Disable()
	controls.reloadButton.Disable()

	content := container.NewVBox(
		widget.NewLabel("This settings page is scaffolded and will be implemented in a follow-up step."),
	)

	return wrapNodeSettingsPage(content, controls)
}

func newDisabledTopLevelPage(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord

	return container.NewVBox(
		widget.NewLabel("Disabled"),
		label,
	)
}

func localNodeSnapshot(store *domain.NodeStore, localNodeID func() string) (domain.Node, bool) {
	if localNodeID == nil {
		return domain.Node{}, false
	}
	id := strings.TrimSpace(localNodeID())
	if id == "" {
		return domain.Node{}, false
	}
	if store == nil {
		return domain.Node{NodeID: id}, false
	}

	node, ok := store.Get(id)
	if !ok {
		return domain.Node{NodeID: id}, false
	}
	if strings.TrimSpace(node.NodeID) == "" {
		node.NodeID = id
	}

	return node, true
}

func isNodeSettingsConnected(dep RuntimeDependencies) bool {
	if dep.Data.CurrentConnStatus == nil {
		return false
	}
	status, known := dep.Data.CurrentConnStatus()
	if !known {
		return false
	}

	return status.State == connectors.ConnectionStateConnected
}

func localNodeSettingsTarget(dep RuntimeDependencies) (app.NodeSettingsTarget, bool) {
	if dep.Data.LocalNodeID == nil {
		return app.NodeSettingsTarget{}, false
	}
	nodeID := strings.TrimSpace(dep.Data.LocalNodeID())
	if nodeID == "" {
		return app.NodeSettingsTarget{}, false
	}

	return app.NodeSettingsTarget{NodeID: nodeID, IsLocal: true}, true
}

func encodeSecurityKey(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	return base64.StdEncoding.EncodeToString(raw)
}

func formatSecurityAdminKeys(keys [][]byte) string {
	if len(keys) == 0 {
		return ""
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		encoded := encodeSecurityKey(key)
		if encoded == "" {
			continue
		}
		parts = append(parts, encoded)
	}

	return strings.Join(parts, "\n")
}

func parseSecurityAdminKeysInput(input string) ([][]byte, error) {
	chunks := strings.FieldsFunc(input, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';'
	})

	keys := make([][]byte, 0, len(chunks))
	seen := make(map[string]struct{}, len(chunks))
	for index, chunk := range chunks {
		keyRaw := strings.TrimSpace(chunk)
		if keyRaw == "" {
			continue
		}
		if len(keys) >= 3 {
			return nil, fmt.Errorf("no more than 3 admin keys are allowed")
		}
		decoded, err := decodeBase64Key(keyRaw)
		if err != nil {
			return nil, fmt.Errorf("admin key #%d is not valid base64", index+1)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("admin key #%d must decode to 32 bytes", index+1)
		}
		keyID := string(decoded)
		if _, exists := seen[keyID]; exists {
			return nil, fmt.Errorf("admin key #%d duplicates a previous key", index+1)
		}
		seen[keyID] = struct{}{}
		keys = append(keys, decoded)
	}

	return keys, nil
}

func decodeBase64Key(raw string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err == nil {
		return decoded, nil
	}

	decoded, rawErr := base64.RawStdEncoding.DecodeString(raw)
	if rawErr == nil {
		return decoded, nil
	}

	return nil, err
}

func copyTextToClipboard(value string) error {
	app := fyne.CurrentApp()
	if app == nil || app.Clipboard() == nil {
		return fmt.Errorf("clipboard is unavailable")
	}

	app.Clipboard().SetContent(value)

	return nil
}

func cloneNodeSecuritySettings(settings app.NodeSecuritySettings) app.NodeSecuritySettings {
	out := settings
	out.PublicKey = append([]byte(nil), settings.PublicKey...)
	out.PrivateKey = append([]byte(nil), settings.PrivateKey...)
	out.AdminKeys = cloneSecurityKeys(settings.AdminKeys)

	return out
}

func cloneSecurityKeys(keys [][]byte) [][]byte {
	if len(keys) == 0 {
		return nil
	}

	out := make([][]byte, 0, len(keys))
	for _, key := range keys {
		out = append(out, append([]byte(nil), key...))
	}

	return out
}

func orUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}

	return v
}
