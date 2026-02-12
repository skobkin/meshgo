package ui

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func newNodeTab(dep RuntimeDependencies) fyne.CanvasObject {
	nodeSettingsTabLogger.Info("building node settings tab")
	saveGate := &nodeSettingsSaveGate{}

	radioTabs := container.NewAppTabs(
		container.NewTabItem("LoRa", newSettingsPlaceholderPage("LoRa settings editing is planned.")),
		container.NewTabItem("Channels", newSettingsPlaceholderPage("Channels editor is planned.")),
		container.NewTabItem("Security", newSettingsPlaceholderPage("Security settings editing is planned.")),
	)
	radioTabs.SetTabLocation(container.TabLocationTop)

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

	status := widget.NewLabel("Loading local node user settings…")
	status.Wrapping = fyne.TextWrapWord
	connectionStateLabel := widget.NewLabel("")

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	longNameEntry := widget.NewEntry()
	shortNameEntry := widget.NewEntry()
	hamLicensedBox := widget.NewCheck("", nil)
	unmessageableBox := widget.NewCheck("", nil)

	saveButton := widget.NewButton("Save", nil)
	cancelButton := widget.NewButton("Cancel", nil)
	reloadButton := widget.NewButton("Reload", nil)

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
		if dep.Data.CurrentConnStatus == nil {
			return false
		}
		status, known := dep.Data.CurrentConnStatus()
		if !known {
			return false
		}

		return status.State == connectors.ConnectionStateConnected
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		if dep.Data.LocalNodeID == nil {
			return app.NodeSettingsTarget{}, false
		}
		nodeID := strings.TrimSpace(dep.Data.LocalNodeID())
		if nodeID == "" {
			return app.NodeSettingsTarget{}, false
		}

		return app.NodeSettingsTarget{NodeID: nodeID, IsLocal: true}, true
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

	updateConnectionLabel := func() {
		if isConnected() {
			connectionStateLabel.SetText("Device connection: connected")
		} else {
			connectionStateLabel.SetText("Device connection: disconnected")
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

		if dep.Actions.NodeSettings == nil {
			reloadButton.Disable()
		} else {
			reloadButton.Enable()
		}
	}

	refreshFromLocalStore := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Debug("local node settings are unavailable: local node ID is unknown")
			status.SetText("Local node ID is not available yet.")

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
			status.SetText("Loaded local node user settings.")
		}
		updateConnectionLabel()
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
		status.SetText("Local edits reverted.")
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node user settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node user settings save unavailable: service is not configured")
			status.SetText("Save is unavailable: node settings service is not configured.")

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node user settings save blocked: device is disconnected", "page_id", pageID)
			status.SetText("Save is unavailable while disconnected.")
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node user settings save failed: local node ID is unknown", "page_id", pageID)
			status.SetText("Save failed: local node ID is not known yet.")

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node user settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			status.SetText("Another settings save is in progress on a different page.")
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		current = currentFromForm()
		next := current
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node user settings", "page_id", pageID, "node_id", target.NodeID)
		status.SetText("Saving user settings…")
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
					status.SetText("Save failed: " + err.Error())
				} else {
					nodeSettingsTabLogger.Info("saved node user settings", "page_id", pageID, "node_id", target.NodeID)
					baseline = settings
					current = settings
					dirty = false
					status.SetText("Saved user settings.")
				}
				mu.Unlock()
				updateConnectionLabel()
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node user settings reload requested", "page_id", pageID)
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node user settings reload failed: local node ID is unknown", "page_id", pageID)
			status.SetText("Reload failed: local node ID is not known yet.")

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Info("node settings service is unavailable: reloading from local store", "page_id", pageID, "node_id", target.NodeID)
			refreshFromLocalStore()

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node user settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			status.SetText("Reload from device is unavailable while disconnected.")

			return
		}

		nodeSettingsTabLogger.Info("reloading node user settings from device", "page_id", pageID, "node_id", target.NodeID)
		status.SetText("Reloading user settings from device…")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadUserSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node user settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					status.SetText("Reload failed: " + err.Error())
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
				status.SetText("Reloaded user settings from device.")
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
					updateConnectionLabel()
					updateButtons()
				})
			}
		}()
	}
	updateConnectionLabel()
	updateButtons()

	return container.NewVBox(
		widget.NewLabel("User settings can be edited and saved per page. Only one settings save can run at a time."),
		connectionStateLabel,
		status,
		form,
		container.NewHBox(reloadButton, cancelButton, saveButton),
	)
}

func newSettingsPlaceholderPage(text string) fyne.CanvasObject {
	status := widget.NewLabel(text)
	status.Wrapping = fyne.TextWrapWord
	saveButton := widget.NewButton("Save", nil)
	saveButton.Disable()
	cancelButton := widget.NewButton("Cancel", nil)
	cancelButton.Disable()

	return container.NewVBox(
		widget.NewLabel("This settings page is scaffolded and will be implemented in a follow-up step."),
		status,
		container.NewHBox(cancelButton, saveButton),
	)
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

func orUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}

	return v
}
