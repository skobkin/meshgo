package ui

import (
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

func TestBindPresentationListenersAppliesInitialAndLiveUpdates(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	window := app.NewWindow("bindings")
	statusLabel := widget.NewLabel("")
	initialStatus := connectors.ConnectionStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: "ip",
	}
	presenter := newConnectionStatusPresenter(window, statusLabel, initialStatus, app.Settings().ThemeVariant(), nil)
	indicator := newUpdateIndicator(app.Settings().ThemeVariant(), false, nil)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	var refreshCalls atomic.Int64

	dep := RuntimeDependencies{
		Data: DataDependencies{
			Bus: messageBus,
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{
					State:         connectors.ConnectionStateConnected,
					TransportName: "serial",
					Target:        "/dev/ttyUSB0",
				}, true
			},
		},
	}

	stopUI, stopUpdates := bindPresentationListeners(dep, app, presenter, indicator, func() {
		refreshCalls.Add(1)
	})
	defer func() {
		stopUpdates()
		stopUI()
		messageBus.Close()
	}()

	messageBus.Publish(connectors.TopicUpdateSnapshot, meshapp.UpdateSnapshot{
		CurrentVersion:  "0.6.0",
		UpdateAvailable: true,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	})

	waitForCondition(t, func() bool {
		return strings.Contains(statusLabel.Text, "connected") &&
			strings.Contains(statusLabel.Text, "/dev/ttyUSB0") &&
			indicator.Button().Visible() &&
			indicator.Button().text == "0.7.0" &&
			refreshCalls.Load() >= 1
	})

	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: "serial",
		Target:        "/dev/ttyUSB0",
		Err:           "link lost",
	})
	messageBus.Publish(connectors.TopicUpdateSnapshot, meshapp.UpdateSnapshot{
		CurrentVersion:  "0.7.0",
		UpdateAvailable: false,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	})

	waitForCondition(t, func() bool {
		return strings.Contains(statusLabel.Text, "disconnected") &&
			strings.Contains(statusLabel.Text, "link lost") &&
			!indicator.Button().Visible() &&
			indicator.Button().text == "" &&
			refreshCalls.Load() >= 2
	})
}
