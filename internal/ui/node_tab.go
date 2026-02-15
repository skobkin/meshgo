package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func newNodeTab(dep RuntimeDependencies) fyne.CanvasObject {
	nodeSettingsTabLogger.Info("building node settings tab")
	saveGate := &nodeSettingsSaveGate{}
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
		deviceTab,
		positionTab,
		powerTab,
		displayTab,
		bluetoothTab,
	)
	deviceTabs.SetTabLocation(container.TabLocationTop)
	deviceTabs.OnSelected = func(item *container.TabItem) {
		if onDeviceTabOpened != nil && item == deviceTab {
			onDeviceTabOpened()
		}
		if onPositionTabOpened != nil && item == positionTab {
			onPositionTabOpened()
		}
		if onPowerTabOpened != nil && item == powerTab {
			onPowerTabOpened()
		}
		if onDisplayTabOpened != nil && item == displayTab {
			onDisplayTabOpened()
		}
		if onBluetoothTabOpened != nil && item == bluetoothTab {
			onBluetoothTabOpened()
		}
	}

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
