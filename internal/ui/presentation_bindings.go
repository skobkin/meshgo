package ui

import (
	"sync"

	"fyne.io/fyne/v2"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

type presentationCallbackGate struct {
	mu               sync.Mutex
	pendingSchedules sync.WaitGroup
	runningCallbacks sync.WaitGroup
	active           bool
	schedule         func(func())
}

func newPresentationCallbackGate(schedule func(func())) *presentationCallbackGate {
	return &presentationCallbackGate{
		active:   true,
		schedule: schedule,
	}
}

func (g *presentationCallbackGate) Do(callback func()) {
	if g == nil || callback == nil || g.schedule == nil {
		return
	}
	g.mu.Lock()
	if !g.active {
		g.mu.Unlock()

		return
	}
	g.pendingSchedules.Add(1)
	g.mu.Unlock()

	g.schedule(func() {
		g.mu.Lock()
		if !g.active {
			g.mu.Unlock()

			return
		}
		g.runningCallbacks.Add(1)
		g.mu.Unlock()

		defer g.runningCallbacks.Done()
		callback()
	})
	g.pendingSchedules.Done()
}

func (g *presentationCallbackGate) Stop() {
	if g == nil {
		return
	}
	g.mu.Lock()
	g.active = false
	g.mu.Unlock()
	g.pendingSchedules.Wait()
	g.runningCallbacks.Wait()
}

func bindPresentationListeners(
	dep RuntimeDependencies,
	fyApp fyne.App,
	connStatusPresenter *connectionStatusPresenter,
	updateIndicator *updateIndicator,
	refreshSidebar func(),
) (func(), func()) {
	callbackGate := newPresentationCallbackGate(fyne.Do)

	appLogger.Debug("starting UI event listeners")
	stopUIListeners := startUIEventListeners(
		dep.Data.Bus,
		func(status busmsg.ConnectionStatus) {
			callbackGate.Do(func() {
				if connStatusPresenter != nil {
					connStatusPresenter.Set(status, fyApp.Settings().ThemeVariant())
				}
			})
		},
		func() {
			callbackGate.Do(func() {
				if connStatusPresenter != nil {
					connStatusPresenter.Refresh(fyApp.Settings().ThemeVariant())
				}
			})
		},
	)
	if status, ok := currentConnStatus(dep); ok && connStatusPresenter != nil {
		connStatusPresenter.Set(status, fyApp.Settings().ThemeVariant())
	}

	appLogger.Debug("starting update snapshot listener")
	stopUpdateSnapshots := startUpdateSnapshotListener(dep.Data.Bus, func(snapshot meshapp.UpdateSnapshot) {
		callbackGate.Do(func() {
			if updateIndicator != nil {
				updateIndicator.ApplySnapshot(snapshot)
			}
			if refreshSidebar != nil {
				refreshSidebar()
			}
		})
	})

	return func() {
			callbackGate.Stop()
			stopUIListeners()
		}, func() {
			callbackGate.Stop()
			stopUpdateSnapshots()
		}
}
