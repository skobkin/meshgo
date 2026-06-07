package ui

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

func TestPresentationCallbackGateDropsQueuedCallbackAfterStop(t *testing.T) {
	var queued func()
	gate := newPresentationCallbackGate(func(callback func()) {
		queued = callback
	})

	var calls atomic.Int64
	gate.Do(func() {
		calls.Add(1)
	})
	if queued == nil {
		t.Fatal("expected callback to be queued")
	}

	gate.Stop()
	queued()

	if calls.Load() != 0 {
		t.Fatalf("expected stopped gate to drop queued callback, got %d calls", calls.Load())
	}
}

func TestPresentationCallbackGateDoesNotQueueAfterStop(t *testing.T) {
	var queuedCalls atomic.Int64
	gate := newPresentationCallbackGate(func(func()) {
		queuedCalls.Add(1)
	})

	gate.Stop()
	gate.Do(func() {})

	if queuedCalls.Load() != 0 {
		t.Fatalf("expected stopped gate not to queue callbacks, got %d", queuedCalls.Load())
	}
}

func TestPresentationCallbackGateStopWaitsForExecutingCallback(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	gate := newPresentationCallbackGate(func(callback func()) {
		go callback()
	})

	gate.Do(func() {
		close(started)
		<-release
	})
	<-started

	stopped := make(chan struct{})
	go func() {
		gate.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("expected stop to wait for the executing callback")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stop after callback completed")
	}
}

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
