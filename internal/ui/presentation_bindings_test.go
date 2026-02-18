package ui

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

func TestBindPresentationListenersAppliesInitialAndLiveUpdates(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	window := app.NewWindow("bindings")
	statusLabel := widget.NewLabel("")
	initialStatus := busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateDisconnected,
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
			CurrentConnStatus: func() (busmsg.ConnectionStatus, bool) {
				return busmsg.ConnectionStatus{
					State:         busmsg.ConnectionStateConnected,
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

	messageBus.Publish(bus.TopicUpdateSnapshot, meshapp.UpdateSnapshot{
		CurrentVersion:  "0.6.0",
		UpdateAvailable: true,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	})

	waitForCondition(t, func() bool {
		status := presenter.CurrentStatus()
		snapshot, known := indicator.Snapshot()

		return status.State == busmsg.ConnectionStateConnected &&
			status.TransportName == "serial" &&
			status.Target == "/dev/ttyUSB0" &&
			known &&
			snapshot.UpdateAvailable &&
			snapshot.Latest.Version == "0.7.0" &&
			refreshCalls.Load() >= 1
	})

	messageBus.Publish(bus.TopicConnStatus, busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateDisconnected,
		TransportName: "serial",
		Target:        "/dev/ttyUSB0",
		Err:           "link lost",
	})
	messageBus.Publish(bus.TopicUpdateSnapshot, meshapp.UpdateSnapshot{
		CurrentVersion:  "0.7.0",
		UpdateAvailable: false,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	})

	waitForCondition(t, func() bool {
		status := presenter.CurrentStatus()
		snapshot, known := indicator.Snapshot()

		return status.State == busmsg.ConnectionStateDisconnected &&
			status.TransportName == "serial" &&
			status.Target == "/dev/ttyUSB0" &&
			status.Err == "link lost" &&
			known &&
			!snapshot.UpdateAvailable &&
			snapshot.Latest.Version == "0.7.0" &&
			refreshCalls.Load() >= 2
	})
}
