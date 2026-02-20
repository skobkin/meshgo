package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/resources"
	"github.com/skobkin/meshgo/internal/ui/widgets"
)

type onShowCanvasObject struct {
	fyne.CanvasObject
	showCalls int
}

func (o *onShowCanvasObject) OnShow() {
	o.showCalls++
}

func TestBuildSidebarLayoutSwitchesTabsAndCallsOnShow(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	chatsTab := widget.NewLabel("Chats")
	nodesTab := &onShowCanvasObject{CanvasObject: widget.NewLabel("Nodes")}
	updateButton := widgets.NewIconNavButton(nil, nil)
	connIcon := widget.NewIcon(nil)

	tabContent := map[string]fyne.CanvasObject{
		"Chats": chatsTab,
		"Nodes": nodesTab,
	}
	order := []string{"Chats", "Nodes"}
	tabIcons := map[string]resources.UIIcon{
		"Chats": resources.UIIconChats,
		"Nodes": resources.UIIconNodes,
	}

	layout := buildSidebarLayout(
		theme.VariantLight,
		tabContent,
		nil,
		order,
		tabIcons,
		updateButton,
		connIcon,
	)

	if layout.left == nil || layout.rightStack == nil {
		t.Fatalf("expected sidebar containers to be initialized")
	}
	if !chatsTab.Visible() {
		t.Fatalf("expected first tab to be visible")
	}
	if nodesTab.Visible() {
		t.Fatalf("expected inactive tab to be hidden")
	}

	nodesButton, ok := layout.left.Objects[1].(*widgets.IconNavButton)
	if !ok {
		t.Fatalf("expected second sidebar object to be icon button")
	}
	nodesButton.Tapped(nil)

	if chatsTab.Visible() {
		t.Fatalf("expected previous tab to be hidden after switch")
	}
	if !nodesTab.Visible() {
		t.Fatalf("expected selected tab to be visible")
	}
	if nodesTab.showCalls != 1 {
		t.Fatalf("expected OnShow to be called once, got %d", nodesTab.showCalls)
	}

	layout.applyTheme(theme.VariantDark)
}

func TestSidebarLayoutSwitchTab_SwitchesProgrammatically(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	chatsTab := widget.NewLabel("Chats")
	nodesTab := &onShowCanvasObject{CanvasObject: widget.NewLabel("Nodes")}
	updateButton := widgets.NewIconNavButton(nil, nil)
	connIcon := widget.NewIcon(nil)

	layout := buildSidebarLayout(
		theme.VariantLight,
		map[string]fyne.CanvasObject{
			"Chats": chatsTab,
			"Nodes": nodesTab,
		},
		nil,
		[]string{"Chats", "Nodes"},
		map[string]resources.UIIcon{
			"Chats": resources.UIIconChats,
			"Nodes": resources.UIIconNodes,
		},
		updateButton,
		connIcon,
	)

	layout.SwitchTab("Nodes")

	if chatsTab.Visible() {
		t.Fatalf("expected chats tab to be hidden after programmatic switch")
	}
	if !nodesTab.Visible() {
		t.Fatalf("expected nodes tab to be visible after programmatic switch")
	}
	if nodesTab.showCalls != 1 {
		t.Fatalf("expected OnShow to be called once, got %d", nodesTab.showCalls)
	}
}
