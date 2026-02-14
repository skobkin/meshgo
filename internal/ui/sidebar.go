package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/resources"
)

const sidebarConnIconSize float32 = 32

type sidebarLayout struct {
	left       *fyne.Container
	rightStack *fyne.Container
	applyTheme func(fyne.ThemeVariant)
}

func buildSidebarLayout(
	initialVariant fyne.ThemeVariant,
	tabContent map[string]fyne.CanvasObject,
	order []string,
	tabIcons map[string]resources.UIIcon,
	updateButton *iconNavButton,
	sidebarConnIcon *widget.Icon,
) sidebarLayout {
	rightStack := container.NewStack()
	for _, key := range order {
		tab := tabContent[key]
		if tab == nil {
			continue
		}
		rightStack.Add(tab)
		tab.Hide()
	}

	active := ""
	for _, name := range order {
		if tabContent[name] == nil {
			continue
		}
		active = name
		tabContent[name].Show()

		break
	}

	navButtons := make(map[string]*iconNavButton, len(order))
	updateNavSelection := func() {
		for name, button := range navButtons {
			button.SetSelected(name == active && !button.Disabled())
		}
	}

	switchTab := func(name string) {
		if name == active {
			return
		}

		current := tabContent[active]
		next := tabContent[name]
		if current == nil || next == nil {
			return
		}

		appLogger.Debug("switching sidebar tab", "from", active, "to", name)
		current.Hide()
		active = name
		next.Show()
		if onShow, ok := next.(interface{ OnShow() }); ok {
			onShow.OnShow()
		}
		updateNavSelection()
		rightStack.Refresh()
	}

	left := container.NewVBox()
	for _, name := range order {
		nameCopy := name
		button := newIconNavButton(resources.UIIconResource(tabIcons[name], initialVariant), 48, func() {
			switchTab(nameCopy)
		})
		navButtons[name] = button
		left.Add(button)
	}

	updateNavSelection()
	left.Add(layout.NewSpacer())
	left.Add(updateButton)
	left.Add(container.NewCenter(container.NewGridWrap(
		fyne.NewSquareSize(sidebarConnIconSize),
		sidebarConnIcon,
	)))

	applyTheme := func(variant fyne.ThemeVariant) {
		for tabName, button := range navButtons {
			button.SetIcon(resources.UIIconResource(tabIcons[tabName], variant))
		}
	}

	return sidebarLayout{
		left:       left,
		rightStack: rightStack,
		applyTheme: applyTheme,
	}
}
