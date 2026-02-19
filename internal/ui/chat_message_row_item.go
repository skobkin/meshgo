package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type chatMessageRowItem struct {
	widget.BaseWidget

	content       fyne.CanvasObject
	onSecondary   func(position fyne.Position)
	onHoverChange func(hovered bool)
}

var _ fyne.Tappable = (*chatMessageRowItem)(nil)
var _ fyne.SecondaryTappable = (*chatMessageRowItem)(nil)
var _ desktop.Hoverable = (*chatMessageRowItem)(nil)

func newChatMessageRowItem(content fyne.CanvasObject) *chatMessageRowItem {
	item := &chatMessageRowItem{content: content}
	item.ExtendBaseWidget(item)

	return item
}

func (r *chatMessageRowItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.content)
}

func (r *chatMessageRowItem) Tapped(*fyne.PointEvent) {}

func (r *chatMessageRowItem) TappedSecondary(event *fyne.PointEvent) {
	if r == nil || r.onSecondary == nil || event == nil {
		return
	}
	r.onSecondary(event.AbsolutePosition)
}

func (r *chatMessageRowItem) MouseIn(*desktop.MouseEvent) {
	if r == nil || r.onHoverChange == nil {
		return
	}
	r.onHoverChange(true)
}

func (r *chatMessageRowItem) MouseMoved(*desktop.MouseEvent) {}

func (r *chatMessageRowItem) MouseOut() {
	if r == nil || r.onHoverChange == nil {
		return
	}
	r.onHoverChange(false)
}
