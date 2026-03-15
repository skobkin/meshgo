package ui

import (
	"context"
	"log/slog"
	"sync/atomic"

	"fyne.io/fyne/v2"

	meshapp "github.com/skobkin/meshgo/internal/app"
)

func startNotificationService(dep RuntimeDependencies, fyApp fyne.App, startHidden bool) func() {
	var appForeground atomic.Bool
	appForeground.Store(!startHidden)
	lifecycle := fyApp.Lifecycle()
	lifecycle.SetOnEnteredForeground(func() {
		appForeground.Store(true)
	})
	lifecycle.SetOnExitedForeground(func() {
		appForeground.Store(false)
	})

	notificationsCtx, stopNotifications := context.WithCancel(context.Background())
	lifecycle.SetOnStopped(stopNotifications)
	notificationService := meshapp.NewNotificationService(
		dep.Data.Bus,
		dep.Data.ChatStore,
		dep.Data.NodeStore,
		dep.Data.CurrentConfig,
		appForeground.Load,
		NewFyneNotificationSender(fyApp),
		slog.With("component", "ui.notifications"),
	)
	notificationService.Start(notificationsCtx)

	return stopNotifications
}
