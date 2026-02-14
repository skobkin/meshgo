package ui

import (
	"sync"

	"fyne.io/fyne/v2"
)

type uiRuntime struct {
	fyApp  fyne.App
	window fyne.Window

	stopNotifications   func()
	stopUIListeners     func()
	stopUpdateSnapshots func()
	onQuit              func()

	shutdownOnce sync.Once
}

func newUIRuntime(
	fyApp fyne.App,
	window fyne.Window,
	stopNotifications func(),
	stopUIListeners func(),
	stopUpdateSnapshots func(),
	onQuit func(),
) *uiRuntime {
	return &uiRuntime{
		fyApp:               fyApp,
		window:              window,
		stopNotifications:   stopNotifications,
		stopUIListeners:     stopUIListeners,
		stopUpdateSnapshots: stopUpdateSnapshots,
		onQuit:              onQuit,
	}
}

func (r *uiRuntime) BindCloseIntercept() {
	if r.window == nil {
		return
	}
	r.window.SetCloseIntercept(func() {
		appLogger.Debug("main window close intercepted: hiding to tray")
		r.window.Hide()
	})
}

func (r *uiRuntime) Quit() {
	r.shutdownOnce.Do(func() {
		appLogger.Info("quitting UI runtime")
		r.stop()
		if r.fyApp != nil {
			r.fyApp.Quit()
		}
	})
}

func (r *uiRuntime) Run(startHidden bool) {
	if r.window != nil {
		r.window.Show()
		if startHidden {
			appLogger.Info("launch option start_hidden is enabled: hiding main window")
			r.window.Hide()
		}
	}
	if r.fyApp != nil {
		r.fyApp.Run()
	}
	appLogger.Info("UI runtime stopped")
	r.shutdownOnce.Do(func() {
		r.stop()
	})
}

func (r *uiRuntime) stop() {
	if r.stopNotifications != nil {
		r.stopNotifications()
	}
	if r.stopUIListeners != nil {
		r.stopUIListeners()
	}
	if r.stopUpdateSnapshots != nil {
		r.stopUpdateSnapshots()
	}
	if r.onQuit != nil {
		r.onQuit()
	}
}
