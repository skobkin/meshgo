package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func handleNodeRequestUserInfoAction(dep RuntimeDependencies, node domain.Node) {
	if dep.Actions.NodeOverview == nil {
		showErrorModal(dep, fmt.Errorf("node overview actions are unavailable"))

		return
	}
	nodeID := strings.TrimSpace(node.NodeID)
	if nodeID == "" {
		return
	}
	requester := localNodeSnapshot(dep)
	if strings.TrimSpace(requester.ID) == "" {
		showErrorModal(dep, fmt.Errorf("local node id is unavailable"))

		return
	}

	go func() {
		err := dep.Actions.NodeOverview.RequestUserInfo(context.Background(), nodeID, requester)
		fyne.Do(func() {
			if err != nil {
				showErrorModal(dep, err)

				return
			}
			showNodeOverviewInfo(dep, fmt.Sprintf("Requested user info from %s.", nodeDisplayName(node)))
		})
	}()
}

func handleNodeRequestTelemetryAction(dep RuntimeDependencies, node domain.Node, kind radio.TelemetryRequestKind) {
	if dep.Actions.NodeOverview == nil {
		showErrorModal(dep, fmt.Errorf("node overview actions are unavailable"))

		return
	}
	nodeID := strings.TrimSpace(node.NodeID)
	if nodeID == "" {
		return
	}

	go func() {
		err := dep.Actions.NodeOverview.RequestTelemetry(context.Background(), nodeID, kind)
		fyne.Do(func() {
			if err != nil {
				showErrorModal(dep, err)

				return
			}
			showNodeOverviewInfo(dep, fmt.Sprintf("Requested %s telemetry from %s.", telemetryKindTitle(kind), nodeDisplayName(node)))
		})
	}()
}

func telemetryKindTitle(kind radio.TelemetryRequestKind) string {
	switch kind {
	case radio.TelemetryRequestDevice:
		return "device"
	case radio.TelemetryRequestEnvironment:
		return "environment"
	case radio.TelemetryRequestAirQuality:
		return "air quality"
	case radio.TelemetryRequestPower:
		return "power"
	default:
		return "unknown"
	}
}

func showNodeOverviewInfo(dep RuntimeDependencies, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	window := currentRuntimeWindow(dep)
	if dep.UIHooks.ShowInfoDialog != nil {
		dep.UIHooks.ShowInfoDialog("Node overview", message, window)

		return
	}
	if window == nil {
		return
	}
	dialog.ShowInformation("Node overview", message, window)
}
