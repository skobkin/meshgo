package ui

import (
	"fyne.io/fyne/v2"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

func bindPresentationListeners(
	dep RuntimeDependencies,
	fyApp fyne.App,
	connStatusPresenter *connectionStatusPresenter,
	updateIndicator *updateIndicator,
	refreshSidebar func(),
) (func(), func()) {
	appLogger.Debug("starting UI event listeners")
	stopUIListeners := startUIEventListeners(
		dep.Data.Bus,
		func(status connectors.ConnectionStatus) {
			fyne.Do(func() {
				if connStatusPresenter != nil {
					connStatusPresenter.Set(status, fyApp.Settings().ThemeVariant())
				}
			})
		},
		func() {
			fyne.Do(func() {
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
	stopUpdateSnapshots := startUpdateSnapshotListener(dep.Data.UpdateSnapshots, func(snapshot meshapp.UpdateSnapshot) {
		fyne.Do(func() {
			if updateIndicator != nil {
				updateIndicator.ApplySnapshot(snapshot)
			}
			if refreshSidebar != nil {
				refreshSidebar()
			}
		})
	})
	if snapshot, ok := currentUpdateSnapshot(dep); ok && updateIndicator != nil {
		updateIndicator.ApplySnapshot(snapshot)
		if refreshSidebar != nil {
			refreshSidebar()
		}
	}

	return stopUIListeners, stopUpdateSnapshots
}
