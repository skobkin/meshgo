package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type linkImage struct {
	widget.BaseWidget
	image *canvas.Image
	onTap func()
}

func newLinkImage(resource fyne.Resource, minSize fyne.Size, onTap func()) *linkImage {
	image := canvas.NewImageFromResource(resource)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(minSize)

	l := &linkImage{
		image: image,
		onTap: onTap,
	}
	l.ExtendBaseWidget(l)

	return l
}

func (l *linkImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(l.image)
}

func (l *linkImage) Tapped(_ *fyne.PointEvent) {
	if l.onTap != nil {
		l.onTap()
	}
}

func (l *linkImage) TappedSecondary(_ *fyne.PointEvent) {}
