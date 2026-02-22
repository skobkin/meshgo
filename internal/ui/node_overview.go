package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/resources"
)

type nodeOverviewOptions struct {
	Title              string
	NodeStore          *domain.NodeStore
	NodeID             func() string
	OnDirectMessage    func(domain.Node)
	OnTraceroute       func(domain.Node)
	OnRequestUserInfo  func(domain.Node)
	OnRequestTelemetry func(domain.Node, radio.TelemetryRequestKind)
	OnTelemetryLog     func(domain.Node)
	ShowCloseButton    bool
	OnClose            func()
	ShowActions        bool
	ModeLocalNode      bool
}

type overviewMetric struct {
	Label     string
	Value     string
	ColorName fyne.ThemeColorName
}

func newNodeOverviewContent(opts nodeOverviewOptions) (fyne.CanvasObject, func()) {
	title := widget.NewLabelWithStyle(orUnknown(opts.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	closeButton := widget.NewButton("X", func() {
		if opts.OnClose != nil {
			opts.OnClose()
		}
	})
	if !opts.ShowCloseButton {
		closeButton.Hide()
	}
	header := container.NewHBox(title, layout.NewSpacer(), closeButton)

	identityTables := container.NewVBox()
	publicKeyEntry := widget.NewEntry()
	publicKeyEntry.Disable()
	publicKeyCopyButton := widget.NewButton("Copy", func() {
		if err := copyTextToClipboard(publicKeyEntry.Text); err != nil {
			nodeSettingsTabLogger.Debug("copy public key from overview failed", "error", err)
		}
	})
	publicKeyCopyButton.Disable()
	identity := container.NewVBox(
		identityTables,
		widget.NewForm(widget.NewFormItem(
			"Public key",
			container.NewBorder(nil, nil, nil, publicKeyCopyButton, publicKeyEntry),
		)),
	)

	powerSection := container.NewVBox()
	environmentSection := container.NewVBox()
	airQualitySection := container.NewVBox()
	otherSection := container.NewVBox()
	positionSection := container.NewVBox()
	firmwareSection := container.NewVBox()

	requestIdentityButton := widget.NewButtonWithIcon("", overviewRefreshIconResource(), nil)
	requestPowerButton := widget.NewButtonWithIcon("", overviewRefreshIconResource(), nil)
	requestEnvironmentButton := widget.NewButtonWithIcon("", overviewRefreshIconResource(), nil)
	requestAirQualityButton := widget.NewButtonWithIcon("", overviewRefreshIconResource(), nil)
	requestOtherButton := widget.NewButtonWithIcon("", overviewRefreshIconResource(), nil)
	requestIdentityButton.Disable()
	requestPowerButton.Disable()
	requestEnvironmentButton.Disable()
	requestAirQualityButton.Disable()
	requestOtherButton.Disable()

	identityCard := overviewCard("Identity", identity, requestIdentityButton)
	powerCard := overviewCard("Telemetry: Power", powerSection, requestPowerButton)
	environmentCard := overviewCard("Telemetry: Environmental", environmentSection, requestEnvironmentButton)
	airQualityCard := overviewCard("Telemetry: Air Quality", airQualitySection, requestAirQualityButton)
	otherCard := overviewCard("Telemetry: Other", otherSection, requestOtherButton)
	positionCard := overviewCard("Position", positionSection)
	firmwareCard := overviewCard("Firmware and Board", firmwareSection)

	adminButton := widget.NewButton("Administration", nil)
	adminButton.Disable()
	adminCard := overviewCard("Remote Administration", container.NewVBox(
		widget.NewLabel("Remote administration is not implemented yet."),
		adminButton,
	))

	chatButton := widget.NewButton("Chat", nil)
	tracerouteButton := widget.NewButton("Traceroute", nil)
	telemetryLogButton := widget.NewButton("Telemetry log", nil)
	tracerouteLogButton := widget.NewButton("Traceroute log", nil)
	telemetryLogButton.Disable()
	tracerouteLogButton.Disable()

	actionsContent := container.NewVBox(
		container.NewGridWithColumns(2, chatButton, tracerouteButton),
		container.NewGridWithColumns(2, telemetryLogButton, tracerouteLogButton),
	)
	actionsCard := overviewCard("Actions", actionsContent)
	if !opts.ShowActions {
		actionsCard.Hide()
	}

	body := container.NewVBox(
		identityCard,
		powerCard,
		environmentCard,
		airQualityCard,
		otherCard,
		positionCard,
		adminCard,
		firmwareCard,
		actionsCard,
	)
	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(700, 520))
	content := container.NewBorder(
		container.NewVBox(
			header,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		scroll,
	)

	stopCh := make(chan struct{})

	resolveNode := func() (domain.Node, bool) {
		if opts.NodeID == nil || opts.NodeStore == nil {
			return domain.Node{}, false
		}
		nodeID := strings.TrimSpace(opts.NodeID())
		if nodeID == "" {
			return domain.Node{}, false
		}

		return opts.NodeStore.Get(nodeID)
	}

	render := func() {
		node, ok := resolveNode()
		if !ok {
			setOverviewSectionMetricRows(identityTables, [][]overviewMetric{
				{
					{Label: "Node", Value: "information is unavailable"},
				},
				{
					{Label: "Last heard", Value: "unknown"},
					{Label: "RSSI", Value: "unknown"},
					{Label: "SNR", Value: "unknown"},
				},
			})
			publicKeyEntry.SetText("")
			publicKeyCopyButton.Disable()
			requestIdentityButton.Disable()
			requestPowerButton.Disable()
			requestEnvironmentButton.Disable()
			requestAirQualityButton.Disable()
			requestOtherButton.Disable()
			powerCard.Hide()
			environmentCard.Hide()
			airQualityCard.Hide()
			otherCard.Hide()
			positionCard.Hide()
			setOverviewSectionMetricRows(firmwareSection, [][]overviewMetric{{
				{Label: "Firmware", Value: "unknown"},
				{Label: "Board", Value: "unknown"},
				{Label: "Image", Value: "unavailable (placeholder)"},
			}})
			firmwareCard.Show()
			chatButton.Disable()
			tracerouteButton.Disable()
			telemetryLogButton.Disable()

			return
		}

		identityMetrics := []overviewMetric{
			{Label: "ID", Value: orUnknown(node.NodeID)},
			{Label: "Short name", Value: orUnknown(node.ShortName)},
			{Label: "Long name", Value: orUnknown(node.LongName)},
		}
		if uptime := overviewUptime(node.UptimeSeconds); uptime != "unknown" {
			identityMetrics = append(identityMetrics, overviewMetric{Label: "Uptime", Value: uptime})
		}
		setOverviewSectionMetricRows(identityTables, [][]overviewMetric{
			identityMetrics,
			{
				{Label: "Last heard", Value: overviewAgo(node.LastHeardAt)},
				overviewRSSIMetric(node),
				overviewSNRMetric(node),
			},
		})
		publicKeyEntry.SetText(encodeNodeSettingsKeyBase64(node.PublicKey))
		if strings.TrimSpace(publicKeyEntry.Text) == "" {
			publicKeyCopyButton.Disable()
		} else {
			publicKeyCopyButton.Enable()
		}

		if opts.OnDirectMessage != nil && !opts.ModeLocalNode {
			chatButton.Enable()
			chatButton.OnTapped = func() { opts.OnDirectMessage(node) }
		} else {
			chatButton.Disable()
		}
		if opts.OnTraceroute != nil && !opts.ModeLocalNode {
			tracerouteButton.Enable()
			tracerouteButton.OnTapped = func() { opts.OnTraceroute(node) }
		} else {
			tracerouteButton.Disable()
		}
		if opts.OnTelemetryLog != nil {
			telemetryLogButton.Enable()
			telemetryLogButton.OnTapped = func() { opts.OnTelemetryLog(node) }
		} else {
			telemetryLogButton.Disable()
		}
		if opts.OnRequestUserInfo != nil && !opts.ModeLocalNode {
			requestIdentityButton.Enable()
			requestIdentityButton.OnTapped = func() { opts.OnRequestUserInfo(node) }
		} else {
			requestIdentityButton.Disable()
		}
		if opts.OnRequestTelemetry != nil && !opts.ModeLocalNode {
			requestPowerButton.Enable()
			requestPowerButton.OnTapped = func() { opts.OnRequestTelemetry(node, radio.TelemetryRequestPower) }
			requestEnvironmentButton.Enable()
			requestEnvironmentButton.OnTapped = func() { opts.OnRequestTelemetry(node, radio.TelemetryRequestEnvironment) }
			requestAirQualityButton.Enable()
			requestAirQualityButton.OnTapped = func() { opts.OnRequestTelemetry(node, radio.TelemetryRequestAirQuality) }
			requestOtherButton.Enable()
			requestOtherButton.OnTapped = func() { opts.OnRequestTelemetry(node, radio.TelemetryRequestDevice) }
		} else {
			requestPowerButton.Disable()
			requestEnvironmentButton.Disable()
			requestAirQualityButton.Disable()
			requestOtherButton.Disable()
		}

		powerMetrics := overviewPowerTelemetryMetrics(node)
		if len(powerMetrics) == 0 {
			powerCard.Hide()
		} else {
			setOverviewSectionMetrics(powerSection, powerMetrics)
			powerCard.Show()
		}

		envMetrics := overviewEnvironmentTelemetryMetrics(node)
		if len(envMetrics) == 0 {
			environmentCard.Hide()
		} else {
			setOverviewSectionMetrics(environmentSection, envMetrics)
			environmentCard.Show()
		}
		airQualityMetrics := overviewAirQualityTelemetryMetrics(node)
		if len(airQualityMetrics) == 0 {
			airQualityCard.Hide()
		} else {
			setOverviewSectionMetrics(airQualitySection, airQualityMetrics)
			airQualityCard.Show()
		}

		otherMetrics := overviewOtherTelemetryMetrics(node)
		if len(otherMetrics) == 0 {
			otherCard.Hide()
		} else {
			setOverviewSectionMetrics(otherSection, otherMetrics)
			otherCard.Show()
		}

		positionMetrics := overviewPositionMetrics(node)
		if len(positionMetrics) == 0 {
			positionCard.Hide()
		} else {
			setOverviewSectionMetricRows(positionSection, [][]overviewMetric{positionMetrics})
			positionCard.Show()
		}

		setOverviewSectionMetricRows(firmwareSection, [][]overviewMetric{{
			{Label: "Firmware", Value: orUnknown(node.FirmwareVersion)},
			{Label: "Board", Value: orUnknown(node.BoardModel)},
			{Label: "Image", Value: "unavailable (placeholder)"},
		}})
	}

	render()

	if opts.NodeStore != nil {
		changes := opts.NodeStore.Changes()
	drain:
		for {
			select {
			case <-changes:
			default:
				break drain
			}
		}
		go func() {
			for {
				select {
				case <-stopCh:
					return
				case <-changes:
					fyne.DoAndWait(render)
				}
			}
		}()
	}

	stop := func() {
		select {
		case <-stopCh:
		default:
			close(stopCh)
		}
	}

	return content, stop
}

func overviewCard(title string, body fyne.CanvasObject, actions ...fyne.CanvasObject) *fyne.Container {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	if len(actions) == 0 {
		return container.NewVBox(titleLabel, body)
	}
	filtered := make([]fyne.CanvasObject, 0, len(actions))
	for _, action := range actions {
		if action != nil {
			filtered = append(filtered, action)
		}
	}
	if len(filtered) == 0 {
		return container.NewVBox(titleLabel, body)
	}

	return container.NewVBox(
		container.NewHBox(titleLabel, layout.NewSpacer(), container.NewHBox(filtered...)),
		body,
	)
}

func overviewRefreshIconResource() fyne.Resource {
	app := fyne.CurrentApp()
	if app != nil {
		if res := resources.UIIconResource(resources.UIIconRefresh, app.Settings().ThemeVariant()); res != nil {
			return res
		}
	}
	if fallback := resources.UIIconResource(resources.UIIconRefresh, theme.VariantDark); fallback != nil {
		return fallback
	}

	return theme.ViewRefreshIcon()
}

func setOverviewSectionMetrics(section *fyne.Container, metrics []overviewMetric) {
	setOverviewSectionMetricRows(section, [][]overviewMetric{metrics})
}

func setOverviewSectionMetricRows(section *fyne.Container, rows [][]overviewMetric) {
	objects := make([]fyne.CanvasObject, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		objects = append(objects, overviewMetricsGrid(row, overviewMetricsRowColumnCount(len(row))))
	}
	section.Objects = objects
	section.Refresh()
}

func overviewMetricsGrid(metrics []overviewMetric, columns int) fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 0, len(metrics))
	for _, metric := range metrics {
		label := widget.NewLabelWithStyle(metric.Label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		cells = append(cells, container.NewVBox(label, overviewMetricValueObject(metric)))
	}

	return container.NewGridWithColumns(columns, cells...)
}

func overviewMetricValueObject(metric overviewMetric) fyne.CanvasObject {
	value := orUnknown(metric.Value)
	if metric.ColorName == "" {
		label := widget.NewLabel(value)
		label.Wrapping = fyne.TextWrapWord

		return label
	}
	style := widget.RichTextStyleInline
	style.ColorName = metric.ColorName
	rich := widget.NewRichText(&widget.TextSegment{Text: value, Style: style})

	return rich
}

func overviewMetricsColumnCount(metricCount int) int {
	switch {
	case metricCount <= 3:
		return 2
	case metricCount <= 8:
		return 3
	case metricCount <= 15:
		return 4
	default:
		return 5
	}
}

func overviewMetricsRowColumnCount(metricCount int) int {
	if metricCount < 2 {
		return 2
	}
	if metricCount > 5 {
		return 5
	}

	return metricCount
}

func overviewAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	return formatSeenAgo(t, time.Now())
}

func overviewRSSIMetric(node domain.Node) overviewMetric {
	metric := overviewMetric{Label: "RSSI", Value: "unknown"}
	if node.RSSI != nil {
		metric.Value = fmt.Sprintf("%d dBm", *node.RSSI)
		metric.ColorName = signalThemeColorForRSSI(*node.RSSI)
	}

	return metric
}

func overviewSNRMetric(node domain.Node) overviewMetric {
	metric := overviewMetric{Label: "SNR", Value: "unknown"}
	if node.SNR != nil {
		metric.Value = fmt.Sprintf("%.2f dB", *node.SNR)
		metric.ColorName = signalThemeColorForSNR(*node.SNR)
	}

	return metric
}

func overviewUptime(uptimeSeconds *uint32) string {
	if uptimeSeconds == nil {
		return "unknown"
	}
	d := time.Duration(*uptimeSeconds) * time.Second
	if d < time.Minute {
		return d.String()
	}
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	minutes := d / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}

	return fmt.Sprintf("%dm", minutes)
}

func overviewPowerTelemetry(node domain.Node) string {
	return overviewMetricLines(overviewPowerTelemetryMetrics(node))
}

func overviewPowerTelemetryMetrics(node domain.Node) []overviewMetric {
	metrics := make([]overviewMetric, 0, 4)
	if node.BatteryLevel != nil {
		metrics = append(metrics, overviewMetric{Label: "Battery", Value: fmt.Sprintf("%d%%", *node.BatteryLevel)})
	}
	if node.Voltage != nil {
		metrics = append(metrics, overviewMetric{Label: "Voltage", Value: fmt.Sprintf("%.2f V", *node.Voltage)})
	}
	if node.PowerVoltage != nil {
		metrics = append(metrics, overviewMetric{Label: "Power voltage", Value: fmt.Sprintf("%.2f V", *node.PowerVoltage)})
	}
	if node.PowerCurrent != nil {
		metrics = append(metrics, overviewMetric{Label: "Power current", Value: fmt.Sprintf("%.3f A", *node.PowerCurrent)})
	}

	return metrics
}

func overviewEnvironmentTelemetry(node domain.Node) string {
	return overviewMetricLines(overviewEnvironmentTelemetryMetrics(node))
}

func overviewEnvironmentTelemetryMetrics(node domain.Node) []overviewMetric {
	metrics := make([]overviewMetric, 0, 10)
	if node.Temperature != nil {
		metrics = append(metrics, overviewMetric{Label: "Temperature", Value: fmt.Sprintf("%.1f C", *node.Temperature)})
	}
	if node.Humidity != nil {
		metrics = append(metrics, overviewMetric{Label: "Humidity", Value: fmt.Sprintf("%.1f%%", *node.Humidity)})
	}
	if node.Pressure != nil {
		metrics = append(metrics, overviewMetric{Label: "Pressure", Value: fmt.Sprintf("%.1f hPa", *node.Pressure)})
	}
	if node.SoilTemperature != nil {
		metrics = append(metrics, overviewMetric{Label: "Soil temperature", Value: fmt.Sprintf("%.1f C", *node.SoilTemperature)})
	}
	if node.SoilMoisture != nil {
		metrics = append(metrics, overviewMetric{Label: "Soil moisture", Value: fmt.Sprintf("%d%%", *node.SoilMoisture)})
	}
	if dewPoint, ok := calculateDewPointCelsius(node.Temperature, node.Humidity); ok {
		metrics = append(metrics, overviewMetric{Label: "Dew point", Value: fmt.Sprintf("%.1f C", dewPoint)})
	}
	if node.GasResistance != nil {
		metrics = append(metrics, overviewMetric{Label: "Gas resistance", Value: fmt.Sprintf("%.2f MOhm", *node.GasResistance)})
	}
	if node.Lux != nil {
		metrics = append(metrics, overviewMetric{Label: "Light", Value: fmt.Sprintf("%.1f lx", *node.Lux)})
	}
	if node.UVLux != nil {
		metrics = append(metrics, overviewMetric{Label: "UV light", Value: fmt.Sprintf("%.1f UVlx", *node.UVLux)})
	}
	if node.Radiation != nil {
		metrics = append(metrics, overviewMetric{Label: "Radiation", Value: fmt.Sprintf("%.2f uR/h", *node.Radiation)})
	}

	return metrics
}

func calculateDewPointCelsius(temperature, humidity *float64) (float64, bool) {
	if temperature == nil || humidity == nil || *humidity <= 0 {
		return 0, false
	}
	const (
		a = 17.27
		b = 237.7
	)
	alpha := (a * *temperature / (b + *temperature)) + math.Log(*humidity/100.0)
	if math.IsNaN(alpha) || math.IsInf(alpha, 0) {
		return 0, false
	}
	dewPoint := (b * alpha) / (a - alpha)
	if math.IsNaN(dewPoint) || math.IsInf(dewPoint, 0) {
		return 0, false
	}

	return dewPoint, true
}

func overviewAirQualityTelemetryMetrics(node domain.Node) []overviewMetric {
	metrics := make([]overviewMetric, 0, 1)
	if node.AirQualityIndex != nil {
		metrics = append(metrics, overviewMetric{Label: "Air quality index", Value: fmt.Sprintf("%.1f", *node.AirQualityIndex)})
	}

	return metrics
}

func overviewOtherTelemetry(node domain.Node) string {
	return overviewMetricLines(overviewOtherTelemetryMetrics(node))
}

func overviewOtherTelemetryMetrics(node domain.Node) []overviewMetric {
	metrics := make([]overviewMetric, 0, 2)
	if node.ChannelUtilization != nil {
		metrics = append(metrics, overviewMetric{Label: "Channel utilization", Value: fmt.Sprintf("%.2f%%", *node.ChannelUtilization)})
	}
	if node.AirUtilTx != nil {
		metrics = append(metrics, overviewMetric{Label: "TX air utilization", Value: fmt.Sprintf("%.2f%%", *node.AirUtilTx)})
	}

	return metrics
}

func overviewPosition(node domain.Node) string {
	return overviewMetricLines(overviewPositionMetrics(node))
}

func overviewPositionMetrics(node domain.Node) []overviewMetric {
	if node.Latitude == nil || node.Longitude == nil {
		return nil
	}
	metrics := []overviewMetric{
		{Label: "Latitude", Value: fmt.Sprintf("%.6f", *node.Latitude)},
		{Label: "Longitude", Value: fmt.Sprintf("%.6f", *node.Longitude)},
	}
	if node.Altitude != nil {
		metrics = append(metrics, overviewMetric{Label: "Altitude", Value: fmt.Sprintf("%d m", *node.Altitude)})
	}
	if node.PositionPrecisionBits != nil {
		metrics = append(metrics, overviewMetric{Label: "Precision", Value: nodeChannelPositionPrecisionLabel(*node.PositionPrecisionBits)})
	}
	relevancy := node.PositionUpdatedAt
	if relevancy.IsZero() {
		relevancy = node.LastHeardAt
	}
	metrics = append(metrics, overviewMetric{Label: "Position age", Value: overviewAgo(relevancy)})

	return metrics
}

func overviewMetricLines(metrics []overviewMetric) string {
	lines := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		lines = append(lines, metric.Label+": "+metric.Value)
	}

	return strings.Join(lines, "\n")
}

func showNodeOverviewModal(
	window fyne.Window,
	dep RuntimeDependencies,
	node domain.Node,
	switchToChats func(),
	openDMChat func(chatKey string),
) {
	if window == nil {
		return
	}
	nodeID := strings.TrimSpace(node.NodeID)
	if nodeID == "" {
		return
	}
	opts := nodeOverviewOptions{
		Title:           nodeDisplayName(node),
		NodeStore:       dep.Data.NodeStore,
		NodeID:          func() string { return nodeID },
		ShowCloseButton: true,
		ShowActions:     true,
		OnDirectMessage: func(target domain.Node) {
			handleNodeDirectMessageAction(dep, switchToChats, openDMChat, target)
		},
		OnTraceroute: func(target domain.Node) {
			handleNodeTracerouteAction(window, dep, target)
		},
		OnRequestUserInfo: func(target domain.Node) {
			handleNodeRequestUserInfoAction(dep, target)
		},
		OnRequestTelemetry: func(target domain.Node, kind radio.TelemetryRequestKind) {
			handleNodeRequestTelemetryAction(dep, target, kind)
		},
		OnTelemetryLog: func(target domain.Node) {
			handleNodeTelemetryLogAction(window, dep, target)
		},
	}
	var modal *widget.PopUp
	var stop func()
	opts.OnClose = func() {
		if stop != nil {
			stop()
		}
		if modal != nil {
			modal.Hide()
		}
	}

	content, stopFn := newNodeOverviewContent(opts)
	stop = stopFn
	modal = widget.NewModalPopUp(content, window.Canvas())
	modal.Resize(fyne.NewSize(760, 560))
	modal.Show()
}

func newNodeOverviewSettingsPage(dep RuntimeDependencies) fyne.CanvasObject {
	content, _ := newNodeOverviewContent(nodeOverviewOptions{
		Title:     "Node overview",
		NodeStore: dep.Data.NodeStore,
		NodeID: func() string {
			return localNodeSnapshot(dep).ID
		},
		ShowActions:   true,
		ModeLocalNode: true,
		OnTelemetryLog: func(target domain.Node) {
			handleNodeTelemetryLogAction(currentRuntimeWindow(dep), dep, target)
		},
	})

	return content
}
