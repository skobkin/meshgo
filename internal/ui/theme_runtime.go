package ui

import (
	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/resources"
)

type themeRuntime struct {
	fyApp               fyne.App
	sidebar             sidebarLayout
	updateIndicator     *updateIndicator
	mapTab              fyne.CanvasObject
	connStatusPresenter *connectionStatusPresenter
	setTrayIcon         func(fyne.ThemeVariant)
}

func newThemeRuntime(
	fyApp fyne.App,
	sidebar sidebarLayout,
	updateIndicator *updateIndicator,
	mapTab fyne.CanvasObject,
	connStatusPresenter *connectionStatusPresenter,
) *themeRuntime {
	return &themeRuntime{
		fyApp:               fyApp,
		sidebar:             sidebar,
		updateIndicator:     updateIndicator,
		mapTab:              mapTab,
		connStatusPresenter: connStatusPresenter,
		setTrayIcon:         func(fyne.ThemeVariant) {},
	}
}

func (r *themeRuntime) SetTrayIconSetter(setter func(fyne.ThemeVariant)) {
	if setter == nil {
		r.setTrayIcon = func(fyne.ThemeVariant) {}

		return
	}
	r.setTrayIcon = setter
}

func (r *themeRuntime) BindSettings() {
	r.fyApp.Settings().AddListener(func(_ fyne.Settings) {
		appLogger.Debug("theme settings changed")
		r.Apply(r.fyApp.Settings().ThemeVariant())
	})
}

func (r *themeRuntime) Apply(variant fyne.ThemeVariant) {
	appLogger.Debug("applying theme resources", "theme", variant)
	r.fyApp.SetIcon(resources.AppIconResource(variant))
	r.setTrayIcon(variant)
	r.sidebar.applyTheme(variant)
	if r.updateIndicator != nil {
		r.updateIndicator.ApplyTheme(variant)
	}
	if mapWidget, ok := r.mapTab.(*mapTabWidget); ok {
		mapWidget.applyThemeVariant(variant)
	}
	if r.connStatusPresenter != nil {
		r.connStatusPresenter.ApplyTheme(variant)
	}
}
