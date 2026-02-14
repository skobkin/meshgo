package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

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
		publicKeyEntry.SetText(encodeNodeSettingsKeyBase64(settings.PublicKey))
		privateKeyEntry.SetText(encodeNodeSettingsKeyBase64(settings.PrivateKey))
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
		next.AdminKeys = cloneNodeSettingsKeyBytesList(adminKeys)
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

func formatSecurityAdminKeys(keys [][]byte) string {
	return formatNodeSettingsKeysPerLine(keys)
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
		decoded, err := decodeNodeSettingsKeyBase64(keyRaw)
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

func cloneNodeSecuritySettings(settings app.NodeSecuritySettings) app.NodeSecuritySettings {
	out := settings
	out.PublicKey = cloneNodeSettingsKeyBytes(settings.PublicKey)
	out.PrivateKey = cloneNodeSettingsKeyBytes(settings.PrivateKey)
	out.AdminKeys = cloneNodeSettingsKeyBytesList(settings.AdminKeys)

	return out
}
