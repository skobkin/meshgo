package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

type nodeOverviewOptions struct {
	Title           string
	NodeStore       *domain.NodeStore
	NodeID          func() string
	OnDirectMessage func(domain.Node)
	OnTraceroute    func(domain.Node)
	ShowCloseButton bool
	OnClose         func()
	ShowActions     bool
	ModeLocalNode   bool
}

func newNodeOverviewContent(opts nodeOverviewOptions) (fyne.CanvasObject, func()) {
	title := widget.NewLabelWithStyle(orUnknown(opts.Title), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	identity := widget.NewLabel("")
	identity.Wrapping = fyne.TextWrapWord
	lastHeard := widget.NewLabel("")
	uptime := widget.NewLabel("")

	powerSection := widget.NewLabel("")
	powerSection.Wrapping = fyne.TextWrapWord
	environmentSection := widget.NewLabel("")
	environmentSection.Wrapping = fyne.TextWrapWord
	otherSection := widget.NewLabel("")
	otherSection.Wrapping = fyne.TextWrapWord
	positionSection := widget.NewLabel("")
	positionSection.Wrapping = fyne.TextWrapWord
	firmwareSection := widget.NewLabel("")
	firmwareSection.Wrapping = fyne.TextWrapWord

	powerCard := overviewCard("Telemetry: Power", powerSection)
	environmentCard := overviewCard("Telemetry: Environmental and Air", environmentSection)
	otherCard := overviewCard("Telemetry: Other", otherSection)
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
	requestUserInfoButton := widget.NewButton("Request user info", nil)
	requestTelemetryButton := widget.NewButton("Request telemetry", nil)
	telemetryLogButton := widget.NewButton("Telemetry log", nil)
	tracerouteLogButton := widget.NewButton("Traceroute log", nil)
	requestUserInfoButton.Disable()
	requestTelemetryButton.Disable()
	telemetryLogButton.Disable()
	tracerouteLogButton.Disable()

	actionsContent := container.NewVBox(
		container.NewGridWithColumns(2, chatButton, tracerouteButton),
		container.NewGridWithColumns(2, requestUserInfoButton, requestTelemetryButton),
		container.NewGridWithColumns(2, telemetryLogButton, tracerouteLogButton),
	)
	actionsCard := overviewCard("Actions", actionsContent)
	if !opts.ShowActions {
		actionsCard.Hide()
	}

	closeButton := widget.NewButton("Close", func() {
		if opts.OnClose != nil {
			opts.OnClose()
		}
	})
	if !opts.ShowCloseButton {
		closeButton.Hide()
	}

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		overviewCard("Identity", identity),
		overviewCard("Last Heard", container.NewVBox(lastHeard, uptime)),
		powerCard,
		environmentCard,
		otherCard,
		positionCard,
		adminCard,
		firmwareCard,
		actionsCard,
		layout.NewSpacer(),
		closeButton,
	)
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(700, 520))

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
			identity.SetText("Node information is unavailable.")
			lastHeard.SetText("Last heard: unknown\nRSSI/SNR: unknown")
			uptime.SetText("Uptime: unknown")
			powerCard.Hide()
			environmentCard.Hide()
			otherCard.Hide()
			positionCard.Hide()
			firmwareSection.SetText("Firmware: unknown\nBoard: unknown\nBoard image: unavailable (placeholder)")
			firmwareCard.Show()
			chatButton.Disable()
			tracerouteButton.Disable()

			return
		}

		identity.SetText(fmt.Sprintf("ID: %s\nShort name: %s\nLong name: %s", orUnknown(node.NodeID), orUnknown(node.ShortName), orUnknown(node.LongName)))
		lastHeard.SetText(fmt.Sprintf(
			"Last heard: %s\nRSSI/SNR: %s",
			overviewAgo(node.LastHeardAt),
			overviewLastHeardSignal(node),
		))
		uptime.SetText("Uptime: " + overviewUptime(node.UptimeSeconds))

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

		powerText := overviewPowerTelemetry(node)
		if powerText == "" {
			powerCard.Hide()
		} else {
			powerSection.SetText(powerText)
			powerCard.Show()
		}

		envText := overviewEnvironmentTelemetry(node)
		if envText == "" {
			environmentCard.Hide()
		} else {
			environmentSection.SetText(envText)
			environmentCard.Show()
		}

		otherText := overviewOtherTelemetry(node)
		if otherText == "" {
			otherCard.Hide()
		} else {
			otherSection.SetText(otherText)
			otherCard.Show()
		}

		posText := overviewPosition(node)
		if posText == "" {
			positionCard.Hide()
		} else {
			positionSection.SetText(posText)
			positionCard.Show()
		}

		firmwareSection.SetText(fmt.Sprintf(
			"Firmware: %s\nBoard: %s\nBoard image: unavailable (placeholder)",
			orUnknown(node.FirmwareVersion),
			orUnknown(node.BoardModel),
		))
	}

	fyne.DoAndWait(render)

	if opts.NodeStore != nil {
		changes := opts.NodeStore.Changes()
		go func() {
			for {
				select {
				case <-stopCh:
					return
				case <-changes:
					fyne.Do(render)
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

	return scroll, stop
}

func overviewCard(title string, body fyne.CanvasObject) *fyne.Container {
	return container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		body,
	)
}

func overviewAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	return formatSeenAgo(t, time.Now())
}

func overviewLastHeardSignal(node domain.Node) string {
	rssi := "?"
	if node.RSSI != nil {
		rssi = fmt.Sprintf("%d dBm", *node.RSSI)
	}
	snr := "?"
	if node.SNR != nil {
		snr = fmt.Sprintf("%.2f dB", *node.SNR)
	}

	return fmt.Sprintf("%s / %s", rssi, snr)
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
	lines := make([]string, 0, 4)
	if node.BatteryLevel != nil {
		lines = append(lines, fmt.Sprintf("Battery: %d%%", *node.BatteryLevel))
	}
	if node.Voltage != nil {
		lines = append(lines, fmt.Sprintf("Voltage: %.2f V", *node.Voltage))
	}
	if node.PowerVoltage != nil {
		lines = append(lines, fmt.Sprintf("Power voltage: %.2f V", *node.PowerVoltage))
	}
	if node.PowerCurrent != nil {
		lines = append(lines, fmt.Sprintf("Power current: %.3f A", *node.PowerCurrent))
	}

	return strings.Join(lines, "\n")
}

func overviewEnvironmentTelemetry(node domain.Node) string {
	lines := make([]string, 0, 5)
	if node.Temperature != nil {
		lines = append(lines, fmt.Sprintf("Temperature: %.1f C", *node.Temperature))
	}
	if node.Humidity != nil {
		lines = append(lines, fmt.Sprintf("Humidity: %.1f%%", *node.Humidity))
	}
	if node.Pressure != nil {
		lines = append(lines, fmt.Sprintf("Pressure: %.1f hPa", *node.Pressure))
	}
	if node.AirQualityIndex != nil {
		lines = append(lines, fmt.Sprintf("Air quality index: %.1f", *node.AirQualityIndex))
	}

	return strings.Join(lines, "\n")
}

func overviewOtherTelemetry(node domain.Node) string {
	lines := make([]string, 0, 2)
	if node.ChannelUtilization != nil {
		lines = append(lines, fmt.Sprintf("Channel utilization: %.2f%%", *node.ChannelUtilization))
	}
	if node.AirUtilTx != nil {
		lines = append(lines, fmt.Sprintf("TX air utilization: %.2f%%", *node.AirUtilTx))
	}

	return strings.Join(lines, "\n")
}

func overviewPosition(node domain.Node) string {
	if node.Latitude == nil || node.Longitude == nil {
		return ""
	}
	lines := []string{
		fmt.Sprintf("Latitude: %.6f", *node.Latitude),
		fmt.Sprintf("Longitude: %.6f", *node.Longitude),
	}
	if node.Altitude != nil {
		lines = append(lines, fmt.Sprintf("Altitude: %d m", *node.Altitude))
	}
	if node.PositionPrecisionBits != nil {
		lines = append(lines, "Precision: "+nodeChannelPositionPrecisionLabel(*node.PositionPrecisionBits))
	}
	relevancy := node.PositionUpdatedAt
	if relevancy.IsZero() {
		relevancy = node.LastHeardAt
	}
	lines = append(lines, "Position age: "+overviewAgo(relevancy))

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
			if dep.Data.LocalNodeID == nil {
				return ""
			}

			return strings.TrimSpace(dep.Data.LocalNodeID())
		},
		ShowActions:   true,
		ModeLocalNode: true,
	})

	return content
}
