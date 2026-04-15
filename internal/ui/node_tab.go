package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func newNodeTab(dep RuntimeDependencies) fyne.CanvasObject {
	return newNodeTabWithOnShow(dep)
}

func newNodeTabWithOnShow(dep RuntimeDependencies) fyne.CanvasObject {
	nodeSettingsTabLogger.Info("building node settings tab")
	saveGate := &nodeSettingsSaveGate{}
	loraPage, onLoRaTabOpened := newNodeLoRaSettingsPage(dep, saveGate)
	loraTab := container.NewTabItem("LoRa", loraPage)
	channelsPage, onChannelsTabOpened := newNodeChannelsSettingsPage(dep, saveGate)
	channelsTab := container.NewTabItem("Channels", channelsPage)
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
	networkPage, onNetworkTabOpened := newNodeNetworkSettingsPage(dep, saveGate)
	networkTab := container.NewTabItem("Network", networkPage)
	mqttPage, onMQTTTabOpened := newNodeMQTTSettingsPage(dep, saveGate)
	mqttTab := container.NewTabItem("MQTT", mqttPage)
	serialPage, onSerialTabOpened := newNodeSerialSettingsPage(dep, saveGate)
	serialTab := container.NewTabItem("Serial", serialPage)
	externalNotificationPage, onExternalNotificationTabOpened := newNodeExternalNotificationSettingsPage(dep, saveGate)
	externalNotificationTab := container.NewTabItem("External notification", externalNotificationPage)
	storeForwardPage, onStoreForwardTabOpened := newNodeStoreForwardSettingsPage(dep, saveGate)
	storeForwardTab := container.NewTabItem("Store & Forward", storeForwardPage)
	rangeTestPage, onRangeTestTabOpened := newNodeRangeTestSettingsPage(dep, saveGate)
	rangeTestTab := container.NewTabItem("Range test", rangeTestPage)
	telemetryPage, onTelemetryTabOpened := newNodeTelemetrySettingsPage(dep, saveGate)
	telemetryTab := container.NewTabItem("Telemetry", telemetryPage)
	cannedMessagePage, onCannedMessageTabOpened := newNodeCannedMessageSettingsPage(dep, saveGate)
	cannedMessageTab := container.NewTabItem("Canned Message", cannedMessagePage)
	audioPage, onAudioTabOpened := newNodeAudioSettingsPage(dep, saveGate)
	audioTab := container.NewTabItem("Audio", audioPage)
	remoteHardwarePage, onRemoteHardwareTabOpened := newNodeRemoteHardwareSettingsPage(dep, saveGate)
	remoteHardwareTab := container.NewTabItem("Remote Hardware", remoteHardwarePage)
	neighborInfoPage, onNeighborInfoTabOpened := newNodeNeighborInfoSettingsPage(dep, saveGate)
	neighborInfoTab := container.NewTabItem("Neighbor Info", neighborInfoPage)
	ambientLightingPage, onAmbientLightingTabOpened := newNodeAmbientLightingSettingsPage(dep, saveGate)
	ambientLightingTab := container.NewTabItem("Ambient Lighting", ambientLightingPage)
	detectionSensorPage, onDetectionSensorTabOpened := newNodeDetectionSensorSettingsPage(dep, saveGate)
	detectionSensorTab := container.NewTabItem("Detection Sensor", detectionSensorPage)
	paxcounterPage, onPaxcounterTabOpened := newNodePaxcounterSettingsPage(dep, saveGate)
	paxcounterTab := container.NewTabItem("Paxcounter", paxcounterPage)
	statusMessagePage, onStatusMessageTabOpened := newNodeStatusMessageSettingsPage(dep, saveGate)
	statusMessageTab := container.NewTabItem("Status Message", statusMessagePage)

	radioTabs := container.NewAppTabs(
		loraTab,
		channelsTab,
		securityTab,
	)
	radioTabs.SetTabLocation(container.TabLocationTop)
	openSelectedRadioTab := func() {
		switch radioTabs.Selected() {
		case loraTab:
			if onLoRaTabOpened != nil {
				onLoRaTabOpened()
			}
		case channelsTab:
			if onChannelsTabOpened != nil {
				onChannelsTabOpened()
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
		networkTab,
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
		case networkTab:
			if onNetworkTabOpened != nil {
				onNetworkTabOpened()
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
		serialTab,
		externalNotificationTab,
		storeForwardTab,
		rangeTestTab,
		telemetryTab,
		cannedMessageTab,
		audioTab,
		remoteHardwareTab,
		neighborInfoTab,
		ambientLightingTab,
		detectionSensorTab,
		paxcounterTab,
		statusMessageTab,
	)
	moduleTabs.SetTabLocation(container.TabLocationTop)
	openSelectedModuleTab := func() {
		switch moduleTabs.Selected() {
		case mqttTab:
			if onMQTTTabOpened != nil {
				onMQTTTabOpened()
			}
		case serialTab:
			if onSerialTabOpened != nil {
				onSerialTabOpened()
			}
		case externalNotificationTab:
			if onExternalNotificationTabOpened != nil {
				onExternalNotificationTabOpened()
			}
		case storeForwardTab:
			if onStoreForwardTabOpened != nil {
				onStoreForwardTabOpened()
			}
		case rangeTestTab:
			if onRangeTestTabOpened != nil {
				onRangeTestTabOpened()
			}
		case telemetryTab:
			if onTelemetryTabOpened != nil {
				onTelemetryTabOpened()
			}
		case cannedMessageTab:
			if onCannedMessageTabOpened != nil {
				onCannedMessageTabOpened()
			}
		case audioTab:
			if onAudioTabOpened != nil {
				onAudioTabOpened()
			}
		case remoteHardwareTab:
			if onRemoteHardwareTabOpened != nil {
				onRemoteHardwareTabOpened()
			}
		case neighborInfoTab:
			if onNeighborInfoTabOpened != nil {
				onNeighborInfoTabOpened()
			}
		case ambientLightingTab:
			if onAmbientLightingTabOpened != nil {
				onAmbientLightingTabOpened()
			}
		case detectionSensorTab:
			if onDetectionSensorTabOpened != nil {
				onDetectionSensorTabOpened()
			}
		case paxcounterTab:
			if onPaxcounterTabOpened != nil {
				onPaxcounterTabOpened()
			}
		case statusMessageTab:
			if onStatusMessageTabOpened != nil {
				onStatusMessageTabOpened()
			}
		}
	}
	moduleTabs.OnSelected = func(_ *container.TabItem) { openSelectedModuleTab() }

	importExportTab := newNodeImportExportPage(dep)
	maintenanceTab := newNodeMaintenancePage(dep)

	overviewTopTab := container.NewTabItem("Node overview", newNodeOverviewSettingsPage(dep))
	radioConfigTab := container.NewTabItem("Radio configuration", radioTabs)
	deviceConfigTab := container.NewTabItem("Device configuration", deviceTabs)
	moduleConfigTab := container.NewTabItem("Module configuration", moduleTabs)
	importExportTopTab := container.NewTabItem("Import/Export", importExportTab)
	maintenanceTopTab := container.NewTabItem("Maintenance", maintenanceTab)
	topTabs := container.NewAppTabs(
		overviewTopTab,
		radioConfigTab,
		deviceConfigTab,
		moduleConfigTab,
		importExportTopTab,
		maintenanceTopTab,
	)
	topTabs.SetTabLocation(container.TabLocationTop)
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

	return topTabs
}
