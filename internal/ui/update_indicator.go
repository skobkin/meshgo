package ui

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/resources"
)

type updateIndicator struct {
	button     *iconNavButton
	mu         sync.RWMutex
	snapshot   meshapp.UpdateSnapshot
	known      bool
	onOpenInfo func(meshapp.UpdateSnapshot)
}

func newUpdateIndicator(
	initialVariant fyne.ThemeVariant,
	initialKnown bool,
	onOpenInfo func(meshapp.UpdateSnapshot),
) *updateIndicator {
	indicator := &updateIndicator{
		snapshot:   meshapp.UpdateSnapshot{},
		known:      initialKnown,
		onOpenInfo: onOpenInfo,
	}
	indicator.button = newIconNavButton(
		resources.UIIconResource(resources.UIIconUpdateAvailable, initialVariant),
		indicator.onTap,
	)
	indicator.applySnapshotUI(indicator.snapshot, indicator.known)

	return indicator
}

func (u *updateIndicator) Button() *iconNavButton {
	return u.button
}

func (u *updateIndicator) ApplyTheme(variant fyne.ThemeVariant) {
	u.button.SetIcon(resources.UIIconResource(resources.UIIconUpdateAvailable, variant))
}

func (u *updateIndicator) ApplySnapshot(snapshot meshapp.UpdateSnapshot) {
	u.mu.Lock()
	prevSnapshot := u.snapshot
	prevKnown := u.known
	u.snapshot = snapshot
	u.known = true
	u.mu.Unlock()

	if !prevKnown || prevSnapshot.UpdateAvailable != snapshot.UpdateAvailable || prevSnapshot.Latest.Version != snapshot.Latest.Version {
		appLogger.Info(
			"applied update snapshot",
			"current_version", strings.TrimSpace(snapshot.CurrentVersion),
			"latest_version", strings.TrimSpace(snapshot.Latest.Version),
			"update_available", snapshot.UpdateAvailable,
			"release_count", len(snapshot.Releases),
		)
	} else {
		appLogger.Debug(
			"refreshed unchanged update snapshot",
			"latest_version", strings.TrimSpace(snapshot.Latest.Version),
			"update_available", snapshot.UpdateAvailable,
		)
	}
	u.applySnapshotUI(snapshot, true)
}

func (u *updateIndicator) Snapshot() (meshapp.UpdateSnapshot, bool) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.snapshot, u.known
}

func (u *updateIndicator) applySnapshotUI(snapshot meshapp.UpdateSnapshot, known bool) {
	if known && snapshot.UpdateAvailable {
		u.button.SetText(snapshot.Latest.Version)
		u.button.Show()

		return
	}
	u.button.SetText("")
	u.button.Hide()
}

func (u *updateIndicator) onTap() {
	u.mu.RLock()
	known := u.known
	snapshot := u.snapshot
	u.mu.RUnlock()
	if !known || !snapshot.UpdateAvailable {
		appLogger.Debug("update button tap ignored: no available update")

		return
	}
	appLogger.Info(
		"opening update dialog",
		"current_version", strings.TrimSpace(snapshot.CurrentVersion),
		"latest_version", strings.TrimSpace(snapshot.Latest.Version),
		"release_count", len(snapshot.Releases),
	)
	if u.onOpenInfo != nil {
		u.onOpenInfo(snapshot)
	}
}
