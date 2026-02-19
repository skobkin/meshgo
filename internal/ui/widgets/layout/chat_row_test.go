package layout

import (
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestChatRowLayoutMinSize(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	layout := NewChatRowLayout(false)
	label := widget.NewLabel("test")
	min := layout.MinSize([]fyne.CanvasObject{label})

	if min != label.MinSize() {
		t.Fatalf("expected min size %v, got %v", label.MinSize(), min)
	}
}

func TestChatRowLayoutAlignRight(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	layout := NewChatRowLayout(true)
	label := widget.NewLabel("test")
	size := fyne.NewSize(100, 50)
	layout.Layout([]fyne.CanvasObject{label}, size)

	if label.Position().X == 0 {
		t.Fatalf("expected right-aligned widget to not be at x=0")
	}
}
