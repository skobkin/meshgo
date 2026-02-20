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
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/resources"
)

const (
	nodeChannelsSettingsPageID      = "radio.channels"
	nodeChannelsDefaultNameMaxBytes = 11
	nodeChannelsEditTimeoutScale    = 3
	nodeChannelsIndicatorIconSize   = 32
	nodeChannelsFallbackTitle       = "LongFast"
)

var nodeChannelPositionPrecisionOptions = []uint32{0, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 32}

func newNodeChannelsSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = nodeChannelsSettingsPageID
	nodeSettingsTabLogger.Debug("building node channels settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading channel settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}
	summaryLabel := widget.NewLabel("No channels loaded")
	statusHint := widget.NewLabel("Reorder, add, edit, and delete channels locally, then click Save to upload to the device.")
	statusHint.Wrapping = fyne.TextWrapWord
	rows := container.NewVBox()
	addButton := widget.NewButtonWithIcon("Add channel", theme.ContentAddIcon(), nil)
	clearButton := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), nil)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Channels", summaryLabel),
	)

	var (
		mu                   sync.Mutex
		baseline             []app.NodeChannelSettings
		draft                []app.NodeChannelSettings
		maxSlots             = app.NodeChannelMaxSlots
		presetTitle          = nodeChannelsFallbackTitle
		dirty                bool
		saving               bool
		initialReloadStarted atomic.Bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}

	updateSummary := func() {
		mu.Lock()
		count := len(draft)
		slots := maxSlots
		mu.Unlock()

		summaryLabel.SetText(fmt.Sprintf("%d of %d channels configured", count, slots))
	}

	var refreshDraft func([]app.NodeChannelSettings)

	applyForm := func(items []app.NodeChannelSettings) {
		mu.Lock()
		draftCopy := cloneNodeChannelSettings(items)
		displayPresetTitle := strings.TrimSpace(presetTitle)
		mu.Unlock()

		rows.Objects = nil
		variant := currentThemeVariant()
		if displayPresetTitle == "" {
			displayPresetTitle = nodeChannelsFallbackTitle
		}

		for index, ch := range draftCopy {
			rowIndex := index
			rowSettings := cloneNodeChannelSettingsEntry(ch)

			row := buildNodeChannelRow(
				rowIndex,
				rowSettings,
				len(draftCopy),
				displayPresetTitle,
				variant,
				func() {
					mu.Lock()
					if rowIndex <= 0 || rowIndex >= len(draft) || saving {
						mu.Unlock()

						return
					}
					draft[rowIndex-1], draft[rowIndex] = draft[rowIndex], draft[rowIndex-1]
					next := cloneNodeChannelSettings(draft)
					mu.Unlock()
					refreshDraft(next)
				},
				func() {
					mu.Lock()
					if rowIndex < 0 || rowIndex+1 >= len(draft) || saving {
						mu.Unlock()

						return
					}
					draft[rowIndex], draft[rowIndex+1] = draft[rowIndex+1], draft[rowIndex]
					next := cloneNodeChannelSettings(draft)
					mu.Unlock()
					refreshDraft(next)
				},
				func() {
					mu.Lock()
					if rowIndex < 0 || rowIndex >= len(draft) || saving {
						mu.Unlock()

						return
					}
					current := cloneNodeChannelSettingsEntry(draft[rowIndex])
					mu.Unlock()

					showNodeChannelEditDialog(dep, "Edit channel", current, func(updated app.NodeChannelSettings) {
						mu.Lock()
						if rowIndex < 0 || rowIndex >= len(draft) {
							mu.Unlock()

							return
						}
						draft[rowIndex] = cloneNodeChannelSettingsEntry(updated)
						next := cloneNodeChannelSettings(draft)
						mu.Unlock()
						refreshDraft(next)
					})
				},
				func() {
					mu.Lock()
					if rowIndex < 0 || rowIndex >= len(draft) || saving {
						mu.Unlock()

						return
					}
					draft = append(draft[:rowIndex], draft[rowIndex+1:]...)
					next := cloneNodeChannelSettings(draft)
					mu.Unlock()
					refreshDraft(next)
				},
			)
			rows.Add(row)
		}
		rows.Refresh()
		updateSummary()
	}

	refreshDraft = func(next []app.NodeChannelSettings) {
		mu.Lock()
		draft = cloneNodeChannelSettings(next)
		dirty = !nodeChannelSettingsListEqual(baseline, draft)
		mu.Unlock()
		applyForm(next)
		updateButtonsForNodeChannels(
			dep,
			saveGate,
			&mu,
			&dirty,
			&saving,
			&draft,
			&maxSlots,
			saveButton,
			cancelButton,
			reloadButton,
			addButton,
			clearButton,
			isConnected,
		)
	}

	applyLoadedSettings := func(next app.NodeChannelSettingsList, nextPresetTitle string, preserveDraft bool) {
		mu.Lock()
		if next.MaxSlots > 0 {
			maxSlots = next.MaxSlots
		} else {
			maxSlots = app.NodeChannelMaxSlots
		}
		presetTitle = strings.TrimSpace(nextPresetTitle)
		if presetTitle == "" {
			presetTitle = nodeChannelsFallbackTitle
		}
		baseline = cloneNodeChannelSettings(next.Channels)
		if !preserveDraft {
			draft = cloneNodeChannelSettings(next.Channels)
		}
		dirty = !nodeChannelSettingsListEqual(baseline, draft)
		currentDraft := cloneNodeChannelSettings(draft)
		mu.Unlock()

		nodeIDLabel.SetText(orUnknown(next.NodeID))
		applyForm(currentDraft)
		updateButtonsForNodeChannels(
			dep,
			saveGate,
			&mu,
			&dirty,
			&saving,
			&draft,
			&maxSlots,
			saveButton,
			cancelButton,
			reloadButton,
			addButton,
			clearButton,
			isConnected,
		)
	}

	reloadFromDevice := func(setStatus bool, preserveDraft bool) {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node channels settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node channels settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)
			updateButtonsForNodeChannels(
				dep,
				saveGate,
				&mu,
				&dirty,
				&saving,
				&draft,
				&maxSlots,
				saveButton,
				cancelButton,
				reloadButton,
				addButton,
				clearButton,
				isConnected,
			)

			return
		}

		if setStatus {
			controls.SetStatus("Reloading channel settings from device…", 1, 2)
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout*nodeChannelsEditTimeoutScale)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadChannelSettings(ctx, target)
			displayPresetTitle := nodeChannelsFallbackTitle
			if err == nil {
				loraSettings, loraErr := dep.Actions.NodeSettings.LoadLoRaSettings(ctx, target)
				if loraErr != nil {
					nodeSettingsTabLogger.Debug("node channels settings fallback title uses default preset", "page_id", pageID, "node_id", target.NodeID, "error", loraErr)
				} else {
					displayPresetTitle = strings.TrimSpace(nodeLoRaPrimaryChannelTitle(loraSettings, ""))
					if displayPresetTitle == "" {
						displayPresetTitle = nodeChannelsFallbackTitle
					}
				}
			}
			fyne.Do(func() {
				if err != nil {
					if setStatus {
						controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					}
					updateButtonsForNodeChannels(
						dep,
						saveGate,
						&mu,
						&dirty,
						&saving,
						&draft,
						&maxSlots,
						saveButton,
						cancelButton,
						reloadButton,
						addButton,
						clearButton,
						isConnected,
					)

					return
				}
				applyLoadedSettings(loaded, displayPresetTitle, preserveDraft)
				if setStatus {
					controls.SetStatus("Reloaded channel settings from device.", 2, 2)
				}
			})
		}()
	}

	tryStartInitialReload := func(reason string) {
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting initial node channels settings load", "page_id", pageID, "reason", reason)
		reloadFromDevice(true, false)
	}

	addButton.OnTapped = func() {
		mu.Lock()
		if saving {
			mu.Unlock()

			return
		}
		if len(draft) >= maxSlots {
			mu.Unlock()
			controls.SetStatus(fmt.Sprintf("No free channel slots left (%d max).", maxSlots), 0, 1)

			return
		}
		initial := defaultNodeChannelSettings()
		mu.Unlock()

		showNodeChannelEditDialog(dep, "Add channel", initial, func(next app.NodeChannelSettings) {
			mu.Lock()
			if saving || len(draft) >= maxSlots {
				mu.Unlock()

				return
			}
			draft = append(draft, cloneNodeChannelSettingsEntry(next))
			nextDraft := cloneNodeChannelSettings(draft)
			mu.Unlock()
			refreshDraft(nextDraft)
		})
	}

	clearButton.OnTapped = func() {
		mu.Lock()
		if saving || len(draft) == 0 {
			mu.Unlock()

			return
		}
		draft = nil
		nextDraft := cloneNodeChannelSettings(draft)
		mu.Unlock()
		refreshDraft(nextDraft)
		controls.SetStatus("Cleared local channel list.", 1, 1)
	}

	cancelButton.OnTapped = func() {
		mu.Lock()
		draft = cloneNodeChannelSettings(baseline)
		dirty = false
		nextDraft := cloneNodeChannelSettings(draft)
		mu.Unlock()
		applyForm(nextDraft)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtonsForNodeChannels(
			dep,
			saveGate,
			&mu,
			&dirty,
			&saving,
			&draft,
			&maxSlots,
			saveButton,
			cancelButton,
			reloadButton,
			addButton,
			clearButton,
			isConnected,
		)
	}

	saveButton.OnTapped = func() {
		if dep.Actions.NodeSettings == nil {
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtonsForNodeChannels(
				dep,
				saveGate,
				&mu,
				&dirty,
				&saving,
				&draft,
				&maxSlots,
				saveButton,
				cancelButton,
				reloadButton,
				addButton,
				clearButton,
				isConnected,
			)

			return
		}
		target, ok := localTarget()
		if !ok {
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtonsForNodeChannels(
				dep,
				saveGate,
				&mu,
				&dirty,
				&saving,
				&draft,
				&maxSlots,
				saveButton,
				cancelButton,
				reloadButton,
				addButton,
				clearButton,
				isConnected,
			)

			return
		}

		mu.Lock()
		payload := app.NodeChannelSettingsList{
			NodeID:   strings.TrimSpace(target.NodeID),
			MaxSlots: maxSlots,
			Channels: cloneNodeChannelSettings(draft),
		}
		saving = true
		mu.Unlock()
		controls.SetStatus("Saving channel settings…", 1, 3)
		updateButtonsForNodeChannels(
			dep,
			saveGate,
			&mu,
			&dirty,
			&saving,
			&draft,
			&maxSlots,
			saveButton,
			cancelButton,
			reloadButton,
			addButton,
			clearButton,
			isConnected,
		)

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout*nodeChannelsEditTimeoutScale)
			defer cancel()
			err := dep.Actions.NodeSettings.SaveChannelSettings(ctx, target, payload)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					mu.Unlock()
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					showErrorModal(dep, fmt.Errorf("channel upload failed: %w", err))
					reloadFromDevice(false, true)
					updateButtonsForNodeChannels(
						dep,
						saveGate,
						&mu,
						&dirty,
						&saving,
						&draft,
						&maxSlots,
						saveButton,
						cancelButton,
						reloadButton,
						addButton,
						clearButton,
						isConnected,
					)

					return
				}
				baseline = cloneNodeChannelSettings(payload.Channels)
				draft = cloneNodeChannelSettings(payload.Channels)
				dirty = false
				nextDraft := cloneNodeChannelSettings(draft)
				mu.Unlock()
				applyForm(nextDraft)
				controls.SetStatus("Saved channel settings.", 3, 3)
				updateButtonsForNodeChannels(
					dep,
					saveGate,
					&mu,
					&dirty,
					&saving,
					&draft,
					&maxSlots,
					saveButton,
					cancelButton,
					reloadButton,
					addButton,
					clearButton,
					isConnected,
				)
			})
		}()
	}

	reloadButton.OnTapped = func() {
		reloadFromDevice(true, false)
	}

	initial := app.NodeChannelSettingsList{MaxSlots: app.NodeChannelMaxSlots}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial, nodeChannelsFallbackTitle, false)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Channel settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Channel settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		connSub := dep.Data.Bus.Subscribe(bus.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					updateButtonsForNodeChannels(
						dep,
						saveGate,
						&mu,
						&dirty,
						&saving,
						&draft,
						&maxSlots,
						saveButton,
						cancelButton,
						reloadButton,
						addButton,
						clearButton,
						isConnected,
					)
				})
			}
		}()
	}
	if dep.Data.NodeStore != nil {
		go func() {
			for range dep.Data.NodeStore.Changes() {
				fyne.Do(func() {
					updateButtonsForNodeChannels(
						dep,
						saveGate,
						&mu,
						&dirty,
						&saving,
						&draft,
						&maxSlots,
						saveButton,
						cancelButton,
						reloadButton,
						addButton,
						clearButton,
						isConnected,
					)
				})
			}
		}()
	}

	content := container.NewVBox(
		form,
		statusHint,
		container.NewHBox(layout.NewSpacer(), addButton, clearButton),
		widget.NewSeparator(),
		rows,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		tryStartInitialReload("tab_opened")
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

func updateButtonsForNodeChannels(
	dep RuntimeDependencies,
	saveGate *nodeSettingsSaveGate,
	mu *sync.Mutex,
	dirty *bool,
	saving *bool,
	draft *[]app.NodeChannelSettings,
	maxSlots *int,
	saveButton *widget.Button,
	cancelButton *widget.Button,
	reloadButton *widget.Button,
	addButton *widget.Button,
	clearButton *widget.Button,
	isConnected func() bool,
) {
	mu.Lock()
	activePage := ""
	if saveGate != nil {
		activePage = strings.TrimSpace(saveGate.ActivePage())
	}
	isDirty := *dirty
	isSaving := *saving
	currentCount := len(*draft)
	slots := *maxSlots
	canAdd := dep.Actions.NodeSettings != nil && !isSaving && currentCount < slots
	canSave := dep.Actions.NodeSettings != nil && isConnected() && !isSaving && isDirty && (activePage == "" || activePage == nodeChannelsSettingsPageID)
	canCancel := !isSaving && isDirty
	canReload := dep.Actions.NodeSettings != nil && !isSaving
	canClear := dep.Actions.NodeSettings != nil && !isSaving && currentCount > 0
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
	if canAdd {
		addButton.Enable()
	} else {
		addButton.Disable()
	}
	if canClear {
		clearButton.Enable()
	} else {
		clearButton.Disable()
	}
}

func buildNodeChannelRow(
	index int,
	channel app.NodeChannelSettings,
	total int,
	presetTitle string,
	variant fyne.ThemeVariant,
	onMoveUp func(),
	onMoveDown func(),
	onEdit func(),
	onDelete func(),
) fyne.CanvasObject {
	isPrimary := index == 0
	title := nodeChannelTitle(fmt.Sprintf("%d. %s", index+1, nodeChannelDisplayTitle(channel, index, presetTitle)), isPrimary)

	icons := buildNodeChannelIndicatorIcons(channel, variant, isPrimary)
	indicators := container.New(layout.NewCenterLayout(), container.NewHBox(icons...))

	moveUpButton := widget.NewButtonWithIcon("", theme.MoveUpIcon(), onMoveUp)
	moveDownButton := widget.NewButtonWithIcon("", theme.MoveDownIcon(), onMoveDown)
	editButton := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), onEdit)
	deleteButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), onDelete)

	if index == 0 {
		moveUpButton.Disable()
	}
	if index >= total-1 {
		moveDownButton.Disable()
	}

	left := container.NewBorder(nil, nil, nil, indicators, title)
	actions := container.NewHBox(moveUpButton, moveDownButton, editButton, deleteButton)

	return widget.NewCard("", "", container.NewBorder(nil, nil, nil, actions, left))
}

func buildNodeChannelIndicatorIcons(channel app.NodeChannelSettings, variant fyne.ThemeVariant, isPrimary bool) []fyne.CanvasObject {
	icons := make([]fyne.CanvasObject, 0, 6)
	if isPrimary {
		icons = append(icons, nodeChannelIcon(resources.UIIconBadgePrimary, variant))
	}
	if channel.PositionPrecision > 0 {
		icons = append(icons, nodeChannelIcon(resources.UIIconMapNodeMarker, variant))
	}
	if channel.UplinkEnabled {
		icons = append(icons, nodeChannelIcon(resources.UIIconCloudUpload, variant))
	}
	if channel.DownlinkEnabled {
		icons = append(icons, nodeChannelIcon(resources.UIIconCloudDownload, variant))
	}
	if channel.Muted {
		icons = append(icons, nodeChannelIcon(resources.UIIconSpeakerMute, variant))
	}
	icons = append(icons, nodeChannelIcon(nodeChannelEncryptionIcon(channel), variant))

	return icons
}

func nodeChannelTitle(text string, isPrimary bool) fyne.CanvasObject {
	style := widget.RichTextStyleInline
	if isPrimary {
		style.TextStyle = fyne.TextStyle{Bold: true}
		style.ColorName = theme.ColorNameSuccess
	}

	return widget.NewRichText(&widget.TextSegment{
		Text:  text,
		Style: style,
	})
}

func nodeChannelIcon(icon resources.UIIcon, variant fyne.ThemeVariant) fyne.CanvasObject {
	res := resources.UIIconResource(icon, variant)
	size := fyne.NewSquareSize(nodeChannelsIndicatorIconSize)
	if res == nil {
		return container.NewGridWrap(size, widget.NewIcon(theme.BrokenImageIcon()))
	}

	return container.NewGridWrap(size, widget.NewIcon(res))
}

func nodeChannelEncryptionIcon(channel app.NodeChannelSettings) resources.UIIcon {
	lowEntropy := len(channel.PSK) <= 1
	precise := channel.PositionPrecision == 32
	if !lowEntropy {
		return resources.UIIconLockGreen
	}
	if precise && channel.UplinkEnabled {
		return resources.UIIconLockRedWarning
	}
	if precise {
		return resources.UIIconLockRed
	}

	return resources.UIIconLockYellow
}

func showNodeChannelEditDialog(
	dep RuntimeDependencies,
	title string,
	initial app.NodeChannelSettings,
	onSave func(app.NodeChannelSettings),
) {
	window := currentRuntimeWindow(dep)
	if window == nil {
		return
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(strings.TrimSpace(initial.Name))
	pskEntry := widget.NewEntry()
	pskEntry.Password = true
	pskEntry.SetText(encodeNodeSettingsKeyBase64(initial.PSK))
	pskCopyButton := widget.NewButton("Copy", nil)
	uplinkBox := widget.NewCheck("", nil)
	uplinkBox.SetChecked(initial.UplinkEnabled)
	downlinkBox := widget.NewCheck("", nil)
	downlinkBox.SetChecked(initial.DownlinkEnabled)
	muteBox := widget.NewCheck("", nil)
	muteBox.SetChecked(initial.Muted)
	positionOptions := nodeChannelPositionPrecisionLabels()
	initialPositionLabel := nodeChannelPositionPrecisionLabel(initial.PositionPrecision)
	if !containsString(positionOptions, initialPositionLabel) {
		positionOptions = append(positionOptions, initialPositionLabel)
	}
	positionSelect := widget.NewSelect(positionOptions, nil)
	positionSelect.SetSelected(initialPositionLabel)
	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord
	pskField := container.NewBorder(nil, nil, nil, pskCopyButton, pskEntry)

	updatePSKCopyButton := func() {
		if strings.TrimSpace(pskEntry.Text) == "" {
			pskCopyButton.Disable()
		} else {
			pskCopyButton.Enable()
		}
	}

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("PSK (base64)", pskField),
		widget.NewFormItem("Uplink", uplinkBox),
		widget.NewFormItem("Downlink", downlinkBox),
		widget.NewFormItem("Muted", muteBox),
		widget.NewFormItem("Position precision", positionSelect),
	)
	hint := widget.NewLabel("Name max 11 bytes. PSK must decode to 0, 1, 16, or 32 bytes.")
	hint.Wrapping = fyne.TextWrapWord

	var modal dialog.Dialog
	cancelButton := widget.NewButton("Cancel", func() {
		if modal != nil {
			modal.Hide()
		}
	})
	saveButton := widget.NewButton("Save", func() {
		next, err := buildNodeChannelFromForm(nameEntry.Text, pskEntry.Text, uplinkBox.Checked, downlinkBox.Checked, muteBox.Checked, positionSelect.Selected)
		if err != nil {
			errorLabel.SetText("Validation failed: " + err.Error())

			return
		}
		next.ID = initial.ID
		errorLabel.SetText("")
		if modal != nil {
			modal.Hide()
		}
		onSave(next)
	})
	pskEntry.OnChanged = func(_ string) {
		updatePSKCopyButton()
	}
	pskCopyButton.OnTapped = func() {
		if err := copyTextToClipboard(pskEntry.Text); err != nil {
			errorLabel.SetText("Copy failed: " + err.Error())

			return
		}
		errorLabel.SetText("PSK copied.")
	}
	updatePSKCopyButton()
	buttons := container.NewHBox(layout.NewSpacer(), cancelButton, saveButton)
	content := container.NewVBox(form, hint, errorLabel, buttons)
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(520, 360))

	modal = dialog.NewCustomWithoutButtons(strings.TrimSpace(title), scroll, window)
	modal.Resize(fyne.NewSize(560, 420))
	modal.Show()
}

func buildNodeChannelFromForm(
	nameText string,
	pskText string,
	uplink bool,
	downlink bool,
	muted bool,
	positionLabel string,
) (app.NodeChannelSettings, error) {
	name := strings.TrimSpace(nameText)
	if len([]byte(name)) > nodeChannelsDefaultNameMaxBytes {
		return app.NodeChannelSettings{}, fmt.Errorf("name must be at most %d bytes", nodeChannelsDefaultNameMaxBytes)
	}
	psk, err := parseNodeChannelPSK(pskText)
	if err != nil {
		return app.NodeChannelSettings{}, err
	}
	position, err := parseNodeChannelPositionPrecision(positionLabel)
	if err != nil {
		return app.NodeChannelSettings{}, err
	}

	return app.NodeChannelSettings{
		Name:              name,
		PSK:               psk,
		UplinkEnabled:     uplink,
		DownlinkEnabled:   downlink,
		Muted:             muted,
		PositionPrecision: position,
	}, nil
}

func parseNodeChannelPSK(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	decoded, err := decodeNodeSettingsKeyBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("PSK is not valid base64")
	}
	switch len(decoded) {
	case 0, 1, 16, 32:
		return cloneNodeSettingsKeyBytes(decoded), nil
	default:
		return nil, fmt.Errorf("PSK must decode to 0, 1, 16, or 32 bytes")
	}
}

func parseNodeChannelPositionPrecision(label string) (uint32, error) {
	label = strings.TrimSpace(label)
	for _, option := range nodeChannelPositionPrecisionOptions {
		if nodeChannelPositionPrecisionLabel(option) == label {
			return option, nil
		}
	}
	if open := strings.LastIndex(label, "("); open >= 0 && strings.HasSuffix(label, ")") {
		raw := strings.TrimSpace(label[open+1 : len(label)-1])
		value, err := strconv.ParseUint(raw, 10, 32)
		if err == nil {
			return uint32(value), nil
		}
	}
	value, err := strconv.ParseUint(label, 10, 32)
	if err == nil {
		return uint32(value), nil
	}

	return 0, fmt.Errorf("unknown position precision selection")
}

func nodeChannelPositionPrecisionLabels() []string {
	labels := make([]string, 0, len(nodeChannelPositionPrecisionOptions))
	for _, option := range nodeChannelPositionPrecisionOptions {
		labels = append(labels, nodeChannelPositionPrecisionLabel(option))
	}

	return labels
}

func nodeChannelPositionPrecisionLabel(value uint32) string {
	switch value {
	case 0:
		return "Disabled (0)"
	case 32:
		return "Precise location (32)"
	default:
		return fmt.Sprintf("Approx. %s (%d)", nodeChannelDistanceLabel(nodeChannelPrecisionBitsToMeters(value)), value)
	}
}

func nodeChannelDisplayTitle(channel app.NodeChannelSettings, index int, presetTitle string) string {
	title := strings.TrimSpace(channel.Name)
	if title != "" {
		return title
	}
	title = strings.TrimSpace(presetTitle)
	if title != "" {
		return title
	}

	if index == 0 {
		return "Primary channel"
	}

	return fmt.Sprintf("Channel %d", index+1)
}

func nodeChannelPrecisionBitsToMeters(bits uint32) float64 {
	return 23905787.925008 * math.Pow(0.5, float64(bits))
}

func nodeChannelDistanceLabel(meters float64) string {
	if meters <= 0 {
		return "0 m"
	}
	if meters >= 10000 {
		return fmt.Sprintf("%.0f km", meters/1000)
	}
	if meters >= 1000 {
		return fmt.Sprintf("%.1f km", meters/1000)
	}

	return fmt.Sprintf("%.0f m", meters)
}

func nodeChannelSettingsListEqual(left []app.NodeChannelSettings, right []app.NodeChannelSettings) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !nodeChannelSettingsEqual(left[index], right[index]) {
			return false
		}
	}

	return true
}

func nodeChannelSettingsEqual(left app.NodeChannelSettings, right app.NodeChannelSettings) bool {
	if strings.TrimSpace(left.Name) != strings.TrimSpace(right.Name) {
		return false
	}
	if left.ID != right.ID ||
		left.UplinkEnabled != right.UplinkEnabled ||
		left.DownlinkEnabled != right.DownlinkEnabled ||
		left.PositionPrecision != right.PositionPrecision ||
		left.Muted != right.Muted {
		return false
	}
	if len(left.PSK) != len(right.PSK) {
		return false
	}
	for index := range left.PSK {
		if left.PSK[index] != right.PSK[index] {
			return false
		}
	}

	return true
}

func cloneNodeChannelSettings(values []app.NodeChannelSettings) []app.NodeChannelSettings {
	if len(values) == 0 {
		return nil
	}
	out := make([]app.NodeChannelSettings, 0, len(values))
	for _, value := range values {
		out = append(out, cloneNodeChannelSettingsEntry(value))
	}

	return out
}

func cloneNodeChannelSettingsEntry(value app.NodeChannelSettings) app.NodeChannelSettings {
	return app.NodeChannelSettings{
		Name:              strings.TrimSpace(value.Name),
		PSK:               cloneNodeSettingsKeyBytes(value.PSK),
		ID:                value.ID,
		UplinkEnabled:     value.UplinkEnabled,
		DownlinkEnabled:   value.DownlinkEnabled,
		PositionPrecision: value.PositionPrecision,
		Muted:             value.Muted,
	}
}

func defaultNodeChannelSettings() app.NodeChannelSettings {
	return app.NodeChannelSettings{
		PSK: cloneNodeSettingsKeyBytes([]byte{1}),
	}
}

func currentThemeVariant() fyne.ThemeVariant {
	currentApp := fyne.CurrentApp()
	if currentApp == nil || currentApp.Settings() == nil {
		return theme.VariantDark
	}

	return currentApp.Settings().ThemeVariant()
}

func containsString(values []string, candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	for _, value := range values {
		if strings.TrimSpace(value) == candidate {
			return true
		}
	}

	return false
}
