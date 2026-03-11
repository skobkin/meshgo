package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type chatRowItem struct {
	widget.BaseWidget

	content      fyne.CanvasObject
	onSecondary  func(position fyne.Position)
	onPrimaryTap func()
}

var _ fyne.Tappable = (*chatRowItem)(nil)
var _ fyne.SecondaryTappable = (*chatRowItem)(nil)

func newChatRowItem(content fyne.CanvasObject) *chatRowItem {
	item := &chatRowItem{content: content}
	item.ExtendBaseWidget(item)

	return item
}

func (r *chatRowItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.content)
}

func (r *chatRowItem) Tapped(*fyne.PointEvent) {
	if r == nil || r.onPrimaryTap == nil {
		return
	}
	r.onPrimaryTap()
}

func (r *chatRowItem) TappedSecondary(event *fyne.PointEvent) {
	if r == nil || r.onSecondary == nil || event == nil {
		return
	}
	r.onSecondary(event.AbsolutePosition)
}
