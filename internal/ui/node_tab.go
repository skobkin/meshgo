package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

type nodeOverviewView struct {
	statusLabel   *widget.Label
	nodeIDLabel   *widget.Label
	displayName   *widget.Label
	longName      *widget.Label
	shortName     *widget.Label
	hardwareModel *widget.Label
	role          *widget.Label
	unmessageable *widget.Label
	charge        *widget.Label
	voltage       *widget.Label
	temperature   *widget.Label
	humidity      *widget.Label
	pressure      *widget.Label
	airQuality    *widget.Label
	powerVoltage  *widget.Label
	powerCurrent  *widget.Label
	rssi          *widget.Label
	snr           *widget.Label
	lastHeard     *widget.Label
	lastUpdated   *widget.Label
}

type nodeUserView struct {
	nodeIDLabel      *widget.Label
	longNameEntry    *widget.Entry
	shortNameEntry   *widget.Entry
	hardwareModel    *widget.Label
	unmessageableBox *widget.Check
	hamLicensedBox   *widget.Check
}

func newNodeTab(store *domain.NodeStore, localNodeID func() string) fyne.CanvasObject {
	overviewContent, overview := newNodeOverviewView()
	userContent, user := newNodeUserView()

	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", overviewContent),
		container.NewTabItem("User", userContent),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	refresh := func() {
		node, known := localNodeSnapshot(store, localNodeID)
		overview.Update(node, known)
		user.Update(node)
	}
	refresh()

	if store != nil {
		go func() {
			for range store.Changes() {
				fyne.Do(refresh)
			}
		}()
	}

	return tabs
}

func newNodeOverviewView() (fyne.CanvasObject, *nodeOverviewView) {
	view := &nodeOverviewView{
		statusLabel:   widget.NewLabel("Local node is not known yet."),
		nodeIDLabel:   widget.NewLabel("unknown"),
		displayName:   widget.NewLabel("unknown"),
		longName:      widget.NewLabel("unknown"),
		shortName:     widget.NewLabel("unknown"),
		hardwareModel: widget.NewLabel("unknown"),
		role:          widget.NewLabel("unknown"),
		unmessageable: widget.NewLabel("unknown"),
		charge:        widget.NewLabel("unknown"),
		voltage:       widget.NewLabel("unknown"),
		temperature:   widget.NewLabel("unknown"),
		humidity:      widget.NewLabel("unknown"),
		pressure:      widget.NewLabel("unknown"),
		airQuality:    widget.NewLabel("unknown"),
		powerVoltage:  widget.NewLabel("unknown"),
		powerCurrent:  widget.NewLabel("unknown"),
		rssi:          widget.NewLabel("unknown"),
		snr:           widget.NewLabel("unknown"),
		lastHeard:     widget.NewLabel("unknown"),
		lastUpdated:   widget.NewLabel("unknown"),
	}
	view.nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	content := container.NewVBox(
		view.statusLabel,
		widget.NewCard("Identity", "", widget.NewForm(
			widget.NewFormItem("Node ID", view.nodeIDLabel),
			widget.NewFormItem("Display Name", view.displayName),
			widget.NewFormItem("Long Name", view.longName),
			widget.NewFormItem("Short Name", view.shortName),
			widget.NewFormItem("Hardware Model", view.hardwareModel),
			widget.NewFormItem("Role", view.role),
			widget.NewFormItem("Unmessageable", view.unmessageable),
		)),
		widget.NewCard("Telemetry", "", widget.NewForm(
			widget.NewFormItem("Charge", view.charge),
			widget.NewFormItem("Voltage", view.voltage),
			widget.NewFormItem("Temperature", view.temperature),
			widget.NewFormItem("Humidity", view.humidity),
			widget.NewFormItem("Pressure", view.pressure),
			widget.NewFormItem("Air Quality Index", view.airQuality),
			widget.NewFormItem("Power Voltage", view.powerVoltage),
			widget.NewFormItem("Power Current", view.powerCurrent),
			widget.NewFormItem("RSSI", view.rssi),
			widget.NewFormItem("SNR", view.snr),
			widget.NewFormItem("Last Heard", view.lastHeard),
			widget.NewFormItem("Updated", view.lastUpdated),
		)),
	)

	return container.NewVScroll(content), view
}

func newNodeUserView() (fyne.CanvasObject, *nodeUserView) {
	longNameEntry := widget.NewEntry()
	longNameEntry.Disable()
	shortNameEntry := widget.NewEntry()
	shortNameEntry.Disable()

	unmessageableBox := widget.NewCheck("", nil)
	unmessageableBox.Disable()

	hamLicensedBox := widget.NewCheck("", nil)
	hamLicensedBox.Disable()

	view := &nodeUserView{
		nodeIDLabel:      widget.NewLabel("unknown"),
		longNameEntry:    longNameEntry,
		shortNameEntry:   shortNameEntry,
		hardwareModel:    widget.NewLabel("unknown"),
		unmessageableBox: unmessageableBox,
		hamLicensedBox:   hamLicensedBox,
	}
	view.nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	form := widget.NewForm(
		widget.NewFormItem("Node ID", view.nodeIDLabel),
		widget.NewFormItem("Long Name", view.longNameEntry),
		widget.NewFormItem("Short Name", view.shortNameEntry),
		widget.NewFormItem("Hardware Model", view.hardwareModel),
		widget.NewFormItem("Unmessageable", view.unmessageableBox),
		widget.NewFormItem("Licensed amateur radio (HAM)", view.hamLicensedBox),
	)

	saveButton := widget.NewButton("Save", nil)
	saveButton.Disable()

	return container.NewVBox(
		widget.NewLabel("Draft settings. Editing and saving are not implemented yet."),
		form,
		saveButton,
	), view
}

func (v *nodeOverviewView) Update(node domain.Node, known bool) {
	if known {
		v.statusLabel.SetText("Local node data loaded.")
	} else {
		v.statusLabel.SetText("Local node data is incomplete or not loaded yet.")
	}

	v.nodeIDLabel.SetText(nodeIDOrUnknown(node.NodeID))
	v.displayName.SetText(nodeDisplayNameOrUnknown(node))
	v.longName.SetText(orUnknown(node.LongName))
	v.shortName.SetText(orUnknown(node.ShortName))
	v.hardwareModel.SetText(orUnknown(node.BoardModel))
	v.role.SetText(orUnknown(node.Role))
	v.unmessageable.SetText(boolPtrText(node.IsUnmessageable))
	v.charge.SetText(chargeOrUnknown(node.BatteryLevel))
	v.voltage.SetText(voltageOrUnknown(node.Voltage))
	v.temperature.SetText(temperatureOrUnknown(node.Temperature))
	v.humidity.SetText(humidityOrUnknown(node.Humidity))
	v.pressure.SetText(pressureOrUnknown(node.Pressure))
	v.airQuality.SetText(aqiOrUnknown(node.AirQualityIndex))
	v.powerVoltage.SetText(voltageOrUnknown(node.PowerVoltage))
	v.powerCurrent.SetText(currentOrUnknown(node.PowerCurrent))
	v.rssi.SetText(intOrUnknown(node.RSSI))
	v.snr.SetText(floatOrUnknown(node.SNR))
	v.lastHeard.SetText(timeOrUnknown(node.LastHeardAt))
	v.lastUpdated.SetText(timeOrUnknown(node.UpdatedAt))
}

func (v *nodeUserView) Update(node domain.Node) {
	v.nodeIDLabel.SetText(nodeIDOrUnknown(node.NodeID))
	v.longNameEntry.SetText(strings.TrimSpace(node.LongName))
	v.shortNameEntry.SetText(strings.TrimSpace(node.ShortName))
	v.hardwareModel.SetText(orUnknown(node.BoardModel))
	v.unmessageableBox.SetChecked(node.IsUnmessageable != nil && *node.IsUnmessageable)
	// HAM flag is not available in the current domain model yet.
	v.hamLicensedBox.SetChecked(false)
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

func nodeIDOrUnknown(nodeID string) string {
	if v := strings.TrimSpace(nodeID); v != "" {
		return v
	}
	return "unknown"
}

func nodeDisplayNameOrUnknown(node domain.Node) string {
	if v := strings.TrimSpace(nodeDisplayName(node)); v != "" {
		return v
	}
	return "unknown"
}

func orUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}
	return v
}

func boolPtrText(v *bool) string {
	if v == nil {
		return "unknown"
	}
	if *v {
		return "yes"
	}
	return "no"
}

func chargeOrUnknown(level *uint32) string {
	if level == nil {
		return "unknown"
	}
	if *level > 100 {
		return "external"
	}
	return fmt.Sprintf("%d%%", *level)
}

func voltageOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.2fV", *value)
}

func temperatureOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f C", *value)
}

func humidityOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f%%", *value)
}

func pressureOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f hPa", *value)
}

func aqiOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.1f", *value)
}

func currentOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.3fA", *value)
}

func intOrUnknown(value *int) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%d", *value)
}

func floatOrUnknown(value *float64) string {
	if value == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.2f", *value)
}

func timeOrUnknown(v time.Time) string {
	if v.IsZero() {
		return "unknown"
	}
	return v.Local().Format("2006-01-02 15:04:05")
}
