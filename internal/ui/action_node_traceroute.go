package ui

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
)

var tracerouteShowErrorDialog = dialog.ShowError
var tracerouteShowModal = showTracerouteModal

func handleNodeTracerouteAction(window fyne.Window, dep RuntimeDependencies, node domain.Node) {
	if window == nil {
		return
	}
	if dep.Actions.Traceroute == nil {
		tracerouteShowErrorDialog(fmt.Errorf("traceroute is unavailable: radio service is not configured"), window)

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
			tracerouteShowErrorDialog(fmt.Errorf("traceroute is locked, try again in %s", remaining), window)

			return
		}
		tracerouteShowErrorDialog(err, window)

		return
	}

	tracerouteShowModal(window, dep.Data.Bus, dep.Data.NodeStore, node, initial)
}
