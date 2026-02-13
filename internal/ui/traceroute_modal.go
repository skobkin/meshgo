package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

const tracerouteModalTimeout = 60 * time.Second

func handleNodeTracerouteAction(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	if window == nil {
		return
	}
	if dep.Actions.Traceroute == nil {
		dialog.ShowError(fmt.Errorf("traceroute is unavailable: radio service is not configured"), window)

		return
	}

	initial, err := dep.Actions.Traceroute.StartTraceroute(context.Background(), app.TracerouteTarget{NodeID: node.NodeID})
	if err != nil {
		var cooldownErr *app.TracerouteCooldownError
		if errors.As(err, &cooldownErr) {
			remaining := cooldownErr.Remaining.Round(time.Second)
			if remaining < time.Second {
				remaining = time.Second
			}
			dialog.ShowError(fmt.Errorf("traceroute is locked, try again in %s", remaining), window)

			return
		}
		dialog.ShowError(err, window)

		return
	}

	showTracerouteModal(window, dep.Data.Bus, dep.Data.NodeStore, node, initial)
}

func showTracerouteModal(
	window fyne.Window,
	messageBus bus.MessageBus,
	nodeStore *domain.NodeStore,
	targetNode domain.Node,
	initial connectors.TracerouteUpdate,
) {
	if window == nil {
		return
	}

	title := widget.NewLabelWithStyle(
		nodeDisplayName(targetNode),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	elapsed := widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{})
	status := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	status.Truncation = fyne.TextTruncateEllipsis
	progress := widget.NewProgressBar()
	progress.SetValue(0)
	errorLabel := widget.NewLabel("")
	errorLabel.Wrapping = fyne.TextWrapWord
	errorLabel.Hide()

	forwardHeader := widget.NewLabelWithStyle("Route traced toward destination:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	forwardPath := widget.NewLabel("")
	forwardPath.TextStyle = fyne.TextStyle{Monospace: true}
	forwardPath.Wrapping = fyne.TextWrapWord

	returnHeader := widget.NewLabelWithStyle("Route traced back to us:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	returnPath := widget.NewLabel("")
	returnPath.TextStyle = fyne.TextStyle{Monospace: true}
	returnPath.Wrapping = fyne.TextWrapWord

	var copyText string
	copyButton := widget.NewButton("Copy", func() {
		app := fyne.CurrentApp()
		if app == nil || app.Clipboard() == nil {
			return
		}
		app.Clipboard().SetContent(copyText)
	})
	copyButton.Disable()

	titleRow := container.NewBorder(nil, nil, title, elapsed, status)
	forwardRow := container.NewHBox(forwardHeader, layout.NewSpacer(), copyButton)

	content := container.NewVBox(
		titleRow,
		progress,
		errorLabel,
		forwardRow,
		forwardPath,
		layout.NewSpacer(),
		returnHeader,
		returnPath,
	)
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(680, 420))

	var modal *widget.PopUp
	stopCh := make(chan struct{})
	var stopOnce sync.Once
	var sub bus.Subscription
	current := initial

	stop := func() {
		stopOnce.Do(func() {
			close(stopCh)
			if messageBus != nil && sub != nil {
				messageBus.Unsubscribe(sub, connectors.TopicTracerouteUpdate)
			}
			if modal != nil {
				modal.Hide()
			}
		})
	}

	closeButton := widget.NewButton("Close", stop)
	content.Add(closeButton)

	refresh := func(now time.Time) {
		status.SetText(tracerouteStatusText(current))
		elapsedDuration := tracerouteElapsedDuration(current, now)
		elapsed.SetText(fmt.Sprintf("Elapsed: %.1f s", float64(elapsedDuration.Milliseconds())/1000))

		progress.SetValue(tracerouteProgressValue(current, elapsedDuration))

		if strings.TrimSpace(current.Error) != "" {
			errorLabel.SetText(current.Error)
			errorLabel.Show()
		} else {
			errorLabel.SetText("")
			errorLabel.Hide()
		}

		forwardText := formatTraceroutePath(current.ForwardRoute, current.ForwardSNR, nodeStore)
		returnText := formatTraceroutePath(current.ReturnRoute, current.ReturnSNR, nodeStore)
		forwardPath.SetText(forwardText)
		returnPath.SetText(returnText)

		copyText = formatTracerouteResults(forwardText, returnText)
		if isTracerouteCopyAvailable(current.Status) {
			copyButton.Enable()
		} else {
			copyButton.Disable()
		}
	}

	refresh(time.Now())

	if messageBus != nil {
		sub = messageBus.Subscribe(connectors.TopicTracerouteUpdate)
		go func() {
			for {
				select {
				case <-stopCh:
					return
				case raw, ok := <-sub:
					if !ok {
						return
					}
					update, ok := raw.(connectors.TracerouteUpdate)
					if !ok || update.RequestID != initial.RequestID {
						continue
					}
					fyne.Do(func() {
						current = update
						refresh(time.Now())
					})
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				fyne.Do(func() {
					if !isTracerouteRunning(current.Status) {
						return
					}
					refresh(time.Now())
				})
			}
		}
	}()

	modal = widget.NewModalPopUp(scroll, window.Canvas())
	modal.Resize(fyne.NewSize(740, 520))
	modal.Show()
}

func tracerouteStatusText(update connectors.TracerouteUpdate) string {
	switch update.Status {
	case connectors.TracerouteStatusStarted:
		return "Started"
	case connectors.TracerouteStatusProgress:
		return "In progress"
	case connectors.TracerouteStatusCompleted:
		return "Complete"
	case connectors.TracerouteStatusFailed:
		return "Failed"
	case connectors.TracerouteStatusTimedOut:
		return "Timed out"
	default:
		return "Update"
	}
}

func isTracerouteRunning(status connectors.TracerouteStatus) bool {
	return status == connectors.TracerouteStatusStarted || status == connectors.TracerouteStatusProgress
}

func tracerouteElapsedDuration(update connectors.TracerouteUpdate, now time.Time) time.Duration {
	elapsedDuration := update.UpdatedAt.Sub(update.StartedAt)
	if isTracerouteRunning(update.Status) {
		elapsedDuration = now.Sub(update.StartedAt)
	}
	if elapsedDuration < 0 {
		return 0
	}

	return elapsedDuration
}

func tracerouteProgressValue(update connectors.TracerouteUpdate, elapsedDuration time.Duration) float64 {
	if update.Status == connectors.TracerouteStatusCompleted {
		return 1
	}
	progressRatio := elapsedDuration.Seconds() / tracerouteModalTimeout.Seconds()
	if progressRatio < 0 {
		return 0
	}
	if progressRatio > 1 {
		return 1
	}

	return progressRatio
}

func isTracerouteCopyAvailable(status connectors.TracerouteStatus) bool {
	return status == connectors.TracerouteStatusCompleted ||
		status == connectors.TracerouteStatusFailed ||
		status == connectors.TracerouteStatusTimedOut
}

func formatTracerouteResults(forwardPath, returnPath string) string {
	lines := []string{
		"Route traced toward destination:",
		forwardPath,
		"",
		"Route traced back to us:",
		returnPath,
	}

	return strings.Join(lines, "\n")
}

func formatTraceroutePath(nodeIDs []string, signals []int32, nodeStore *domain.NodeStore) string {
	if len(nodeIDs) == 0 {
		return "Waiting for route data..."
	}
	effectiveSignals := signals
	if len(signals) != len(nodeIDs)-1 {
		effectiveSignals = make([]int32, len(nodeIDs)-1)
		for i := range effectiveSignals {
			effectiveSignals[i] = -128
		}
	}

	lines := make([]string, 0, len(nodeIDs)*2)
	lines = append(lines, "■ "+tracerouteNodeDisplay(nodeIDs[0], nodeStore))
	for i := 1; i < len(nodeIDs); i++ {
		lines = append(lines, fmt.Sprintf("⇊ %s", tracerouteHopSignalLabel(effectiveSignals[i-1])))
		lines = append(lines, "■ "+tracerouteNodeDisplay(nodeIDs[i], nodeStore))
	}

	return strings.Join(lines, "\n")
}

func tracerouteHopSignalLabel(raw int32) string {
	if raw == -128 {
		return "SNR: ?"
	}
	if tracerouteHopSignalIsRSSI(raw) {
		return fmt.Sprintf("RSSI: %d dBm", raw)
	}

	return fmt.Sprintf("SNR: %.2f dB", float64(raw)/4)
}

func tracerouteHopSignalIsRSSI(raw int32) bool {
	return raw < -80
}

func tracerouteNodeDisplay(nodeID string, nodeStore *domain.NodeStore) string {
	id := strings.TrimSpace(nodeID)
	if id == "" {
		return "unknown"
	}
	if nodeStore == nil {
		return id
	}
	node, ok := nodeStore.Get(id)
	if !ok {
		return id
	}
	display := strings.TrimSpace(nodeDisplayName(node))
	if display == "" || display == id {
		return id
	}

	return fmt.Sprintf("%s (%s)", display, id)
}
