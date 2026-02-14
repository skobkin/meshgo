package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
)

func TestUIRuntimeQuitStopsOnceAndQuitsApp(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)
	app := &appRunQuitSpy{App: base}

	var stopNotificationsCalls int
	var stopUIListenersCalls int
	var stopUpdateCalls int
	var onQuitCalls int

	runtime := newUIRuntime(
		app,
		nil,
		func() { stopNotificationsCalls++ },
		func() { stopUIListenersCalls++ },
		func() { stopUpdateCalls++ },
		func() { onQuitCalls++ },
	)

	runtime.Quit()
	runtime.Quit()

	if app.quitCalls != 1 {
		t.Fatalf("expected app quit once, got %d", app.quitCalls)
	}
	if stopNotificationsCalls != 1 || stopUIListenersCalls != 1 || stopUpdateCalls != 1 || onQuitCalls != 1 {
		t.Fatalf(
			"expected stop callbacks once: notifications=%d listeners=%d updates=%d onQuit=%d",
			stopNotificationsCalls,
			stopUIListenersCalls,
			stopUpdateCalls,
			onQuitCalls,
		)
	}
}

func TestUIRuntimeBindCloseInterceptHidesWindow(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)

	window := &windowSpy{Window: base.NewWindow("runtime")}
	runtime := newUIRuntime(base, window, nil, nil, nil, nil)

	runtime.BindCloseIntercept()
	if window.closeIntercept == nil {
		t.Fatalf("expected close intercept to be set")
	}

	window.closeIntercept()
	if window.hideCalls != 1 {
		t.Fatalf("expected intercept to hide window once, got %d", window.hideCalls)
	}
}

func TestUIRuntimeRunShowsWindowAndStopsAfterRunReturns(t *testing.T) {
	base := fynetest.NewApp()
	t.Cleanup(base.Quit)
	app := &appRunQuitSpy{App: base}
	window := &windowSpy{Window: base.NewWindow("runtime")}

	var stopCalls int
	runtime := newUIRuntime(
		app,
		window,
		func() { stopCalls++ },
		nil,
		nil,
		nil,
	)

	runtime.Run(true)

	if app.runCalls != 1 {
		t.Fatalf("expected app run once, got %d", app.runCalls)
	}
	if window.showCalls != 1 {
		t.Fatalf("expected window show once, got %d", window.showCalls)
	}
	if window.hideCalls != 1 {
		t.Fatalf("expected window hide once for start hidden, got %d", window.hideCalls)
	}
	if stopCalls != 1 {
		t.Fatalf("expected stop callback once, got %d", stopCalls)
	}
}
