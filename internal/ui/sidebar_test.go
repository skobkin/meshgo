package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/resources"
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
	updateButton := newIconNavButton(nil, nil)
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

	nodesButton, ok := layout.left.Objects[1].(*iconNavButton)
	if !ok {
		t.Fatalf("expected second sidebar object to be icon button")
	}
	nodesButton.onTap()

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
