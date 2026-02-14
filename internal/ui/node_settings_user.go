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
