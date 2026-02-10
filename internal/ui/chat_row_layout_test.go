package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestChatRowLayout_LeftAlignAndRatio(t *testing.T) {
	row := &testCanvasObject{minSize: fyne.NewSize(20, 10)}
	layout := newChatRowLayout(false)

	layout.Layout([]fyne.CanvasObject{row}, fyne.NewSize(200, 40))

	if row.Position().X != 0 {
		t.Fatalf("expected left-aligned row, got X=%v", row.Position().X)
	}
	if row.Size().Width != 160 {
		t.Fatalf("expected width 160, got %v", row.Size().Width)
	}
	if row.Size().Height != 40 {
		t.Fatalf("expected height 40, got %v", row.Size().Height)
	}
}

func TestChatRowLayout_RightAlignAndRatio(t *testing.T) {
	row := &testCanvasObject{minSize: fyne.NewSize(20, 10)}
	layout := newChatRowLayout(true)

	layout.Layout([]fyne.CanvasObject{row}, fyne.NewSize(200, 40))

	if row.Position().X != 40 {
		t.Fatalf("expected right-aligned row at X=40, got %v", row.Position().X)
	}
	if row.Size().Width != 160 {
		t.Fatalf("expected width 160, got %v", row.Size().Width)
	}
}

func TestChatRowWidth_RespectsBounds(t *testing.T) {
	if w := chatRowWidth(100, 20); w != 80 {
		t.Fatalf("expected ratio width 80, got %v", w)
	}
	if w := chatRowWidth(100, 95); w != 95 {
		t.Fatalf("expected min width 95, got %v", w)
	}
	if w := chatRowWidth(100, 120); w != 100 {
		t.Fatalf("expected bounded width 100, got %v", w)
	}
}

type testCanvasObject struct {
	position fyne.Position
	size     fyne.Size
	minSize  fyne.Size
	hidden   bool
}

func (o *testCanvasObject) MinSize() fyne.Size {
	return o.minSize
}

func (o *testCanvasObject) Move(pos fyne.Position) {
	o.position = pos
}

func (o *testCanvasObject) Position() fyne.Position {
	return o.position
}

func (o *testCanvasObject) Resize(size fyne.Size) {
	o.size = size
}

func (o *testCanvasObject) Size() fyne.Size {
	return o.size
}

func (o *testCanvasObject) Hide() {
	o.hidden = true
}

func (o *testCanvasObject) Visible() bool {
	return !o.hidden
}

func (o *testCanvasObject) Show() {
	o.hidden = false
}

func (o *testCanvasObject) Refresh() {
}
