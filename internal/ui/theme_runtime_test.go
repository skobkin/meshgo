package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func TestThemeRuntimeApplyInvokesThemeTargets(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	var sidebarCalls int
	var mapCalls int
	var trayCalls int
	indicator := &updateIndicator{button: newIconNavButton(nil, nil)}
	presenter := &connectionStatusPresenter{sidebarIcon: widget.NewIcon(nil)}

	runtime := newThemeRuntime(
		app,
		sidebarLayout{
			applyTheme: func(fyne.ThemeVariant) {
				sidebarCalls++
			},
		},
		indicator,
		func(fyne.ThemeVariant) {
			mapCalls++
		},
		presenter,
	)
	runtime.SetTrayIconSetter(func(fyne.ThemeVariant) {
		trayCalls++
	})
	runtime.BindSettings()
	runtime.Apply(theme.VariantDark)

	if sidebarCalls != 1 {
		t.Fatalf("expected sidebar theme callback once, got %d", sidebarCalls)
	}
	if trayCalls != 1 {
		t.Fatalf("expected tray theme callback once, got %d", trayCalls)
	}
	if mapCalls != 1 {
		t.Fatalf("expected map theme callback once, got %d", mapCalls)
	}
	if indicator.button.icon == nil {
		t.Fatalf("expected update indicator icon to be applied")
	}
	if presenter.sidebarIcon.Resource == nil {
		t.Fatalf("expected connection status icon to be applied")
	}
}

func TestThemeRuntimeSetTrayIconSetterNilUsesNoop(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	runtime := newThemeRuntime(app, sidebarLayout{
		applyTheme: func(fyne.ThemeVariant) {},
	}, nil, nil, nil)
	runtime.SetTrayIconSetter(nil)
	runtime.Apply(theme.VariantLight)
}
