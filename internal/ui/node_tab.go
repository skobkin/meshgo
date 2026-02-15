package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func newNodeTab(dep RuntimeDependencies) fyne.CanvasObject {
	tab, _ := newNodeTabWithOnShow(dep)

	return tab
}

func newNodeTabWithOnShow(dep RuntimeDependencies) (fyne.CanvasObject, func()) {
	nodeSettingsTabLogger.Info("building node settings tab")
	saveGate := &nodeSettingsSaveGate{}
	loraPage, onLoRaTabOpened := newNodeLoRaSettingsPage(dep, saveGate)
	loraTab := container.NewTabItem("LoRa", loraPage)
	securityPage, onSecurityTabOpened := newNodeSecuritySettingsPage(dep, saveGate)
	securityTab := container.NewTabItem("Security", securityPage)
	devicePage, onDeviceTabOpened := newNodeDeviceSettingsPage(dep, saveGate)
	deviceTab := container.NewTabItem("Device", devicePage)
	positionPage, onPositionTabOpened := newNodePositionSettingsPage(dep, saveGate)
	positionTab := container.NewTabItem("Position", positionPage)
	powerPage, onPowerTabOpened := newNodePowerSettingsPage(dep, saveGate)
	powerTab := container.NewTabItem("Power", powerPage)
	displayPage, onDisplayTabOpened := newNodeDisplaySettingsPage(dep, saveGate)
	displayTab := container.NewTabItem("Display", displayPage)
	bluetoothPage, onBluetoothTabOpened := newNodeBluetoothSettingsPage(dep, saveGate)
	bluetoothTab := container.NewTabItem("Bluetooth", bluetoothPage)
	mqttPage, onMQTTTabOpened := newNodeMQTTSettingsPage(dep, saveGate)
	mqttTab := container.NewTabItem("MQTT", mqttPage)

	radioTabs := container.NewAppTabs(
		loraTab,
		container.NewTabItem("Channels", newSettingsPlaceholderPage("Channels editor is planned.")),
		securityTab,
	)
	radioTabs.SetTabLocation(container.TabLocationTop)
	openSelectedRadioTab := func() {
		switch radioTabs.Selected() {
		case loraTab:
			if onLoRaTabOpened != nil {
				onLoRaTabOpened()
			}
		case securityTab:
			if onSecurityTabOpened != nil {
				onSecurityTabOpened()
			}
		}
	}
	radioTabs.OnSelected = func(_ *container.TabItem) { openSelectedRadioTab() }

	deviceTabs := container.NewAppTabs(
		container.NewTabItem("User", newNodeUserSettingsPage(dep, saveGate)),
		deviceTab,
		positionTab,
		powerTab,
		displayTab,
		bluetoothTab,
	)
	deviceTabs.SetTabLocation(container.TabLocationTop)
	openSelectedDeviceTab := func() {
		switch deviceTabs.Selected() {
		case deviceTab:
			if onDeviceTabOpened != nil {
				onDeviceTabOpened()
			}
		case positionTab:
			if onPositionTabOpened != nil {
				onPositionTabOpened()
			}
		case powerTab:
			if onPowerTabOpened != nil {
				onPowerTabOpened()
			}
		case displayTab:
			if onDisplayTabOpened != nil {
				onDisplayTabOpened()
			}
		case bluetoothTab:
			if onBluetoothTabOpened != nil {
				onBluetoothTabOpened()
			}
		}
	}
	deviceTabs.OnSelected = func(_ *container.TabItem) { openSelectedDeviceTab() }

	moduleTabs := container.NewAppTabs(
		mqttTab,
		container.NewTabItem("Serial", newSettingsPlaceholderPage("Serial module settings editing is planned.")),
		container.NewTabItem("External notification", newSettingsPlaceholderPage("External notification module settings editing is planned.")),
		container.NewTabItem("Store & Forward", newSettingsPlaceholderPage("Store & Forward module settings editing is planned.")),
		container.NewTabItem("Range test", newSettingsPlaceholderPage("Range test module settings editing is planned.")),
		container.NewTabItem("Telemetry", newSettingsPlaceholderPage("Telemetry module settings editing is planned.")),
		container.NewTabItem("Neighbor Info", newSettingsPlaceholderPage("Neighbor Info module settings editing is planned.")),
		container.NewTabItem("Status Message", newSettingsPlaceholderPage("Status Message module settings editing is planned.")),
	)
	moduleTabs.SetTabLocation(container.TabLocationTop)
	openSelectedModuleTab := func() {
		if moduleTabs.Selected() == mqttTab && onMQTTTabOpened != nil {
			onMQTTTabOpened()
		}
	}
	moduleTabs.OnSelected = func(_ *container.TabItem) { openSelectedModuleTab() }

	importExportTab := newDisabledTopLevelPage("Import/Export is planned and currently disabled.")
	maintenanceTab := newDisabledTopLevelPage("Maintenance is planned and currently disabled.")

	radioConfigTab := container.NewTabItem("Radio configuration", radioTabs)
	deviceConfigTab := container.NewTabItem("Device configuration", deviceTabs)
	moduleConfigTab := container.NewTabItem("Module configuration", moduleTabs)
	importExportTopTab := container.NewTabItem("Import/Export", importExportTab)
	maintenanceTopTab := container.NewTabItem("Maintenance", maintenanceTab)
	topTabs := container.NewAppTabs(
		radioConfigTab,
		deviceConfigTab,
		moduleConfigTab,
		importExportTopTab,
		maintenanceTopTab,
	)
	topTabs.SetTabLocation(container.TabLocationTop)
	topTabs.DisableIndex(3)
	topTabs.DisableIndex(4)
	topTabs.OnSelected = func(_ *container.TabItem) {
		switch topTabs.Selected() {
		case radioConfigTab:
			openSelectedRadioTab()
		case deviceConfigTab:
			openSelectedDeviceTab()
		case moduleConfigTab:
			openSelectedModuleTab()
		}
	}

	onShow := func() {
		// AppTabs keeps LoRa selected by default, but `OnSelected` does not fire for that
		// initial selection. Trigger the currently selected nested tab when Node Settings
		// becomes visible so LoRa performs its first lazy load on first open (not at startup).
		switch topTabs.Selected() {
		case radioConfigTab:
			openSelectedRadioTab()
		case deviceConfigTab:
			openSelectedDeviceTab()
		case moduleConfigTab:
			openSelectedModuleTab()
		}
	}

	return topTabs, onShow
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
