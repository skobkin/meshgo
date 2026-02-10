package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type iconNavButton struct {
	widget.DisableableWidget

	icon     fyne.Resource
	iconSize float32
	onTap    func()
	selected bool
	hovered  bool
}

func newIconNavButton(icon fyne.Resource, iconSize float32, onTap func()) *iconNavButton {
	b := &iconNavButton{
		icon:     icon,
		iconSize: iconSize,
		onTap:    onTap,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *iconNavButton) SetIcon(icon fyne.Resource) {
	b.icon = icon
	b.Refresh()
}

func (b *iconNavButton) SetSelected(selected bool) {
	if b.selected == selected {
		return
	}
	b.selected = selected
	b.Refresh()
}

func (b *iconNavButton) MinSize() fyne.Size {
	th := b.Theme()
	pad := th.Size(theme.SizeNamePadding) * 2
	side := b.iconSize + pad
	return fyne.NewSize(side, side)
}

func (b *iconNavButton) Tapped(_ *fyne.PointEvent) {
	if b.Disabled() {
		return
	}
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *iconNavButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *iconNavButton) MouseIn(_ *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *iconNavButton) MouseMoved(_ *desktop.MouseEvent) {}

func (b *iconNavButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *iconNavButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = b.Theme().Size(theme.SizeNameInputRadius)

	img := canvas.NewImageFromResource(b.icon)
	img.FillMode = canvas.ImageFillContain

	return &iconNavButtonRenderer{
		button:     b,
		background: bg,
		icon:       img,
		objects:    []fyne.CanvasObject{bg, img},
	}
}

type iconNavButtonRenderer struct {
	button     *iconNavButton
	background *canvas.Rectangle
	icon       *canvas.Image
	objects    []fyne.CanvasObject
}

func (r *iconNavButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)

	pad := r.button.Theme().Size(theme.SizeNamePadding)
	maxW := size.Width - pad*2
	maxH := size.Height - pad*2

	iconSide := r.button.iconSize
	if maxW < iconSide {
		iconSide = maxW
	}
	if maxH < iconSide {
		iconSide = maxH
	}
	if iconSide < 0 {
		iconSide = 0
	}

	iconSize := fyne.NewSquareSize(iconSide)
	r.icon.Resize(iconSize)
	r.icon.Move(fyne.NewPos(
		(size.Width-iconSize.Width)/2,
		(size.Height-iconSize.Height)/2,
	))
}

func (r *iconNavButtonRenderer) MinSize() fyne.Size {
	return r.button.MinSize()
}

func (r *iconNavButtonRenderer) Refresh() {
	th := r.button.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	switch {
	case r.button.Disabled():
		r.background.FillColor = th.Color(theme.ColorNameDisabledButton, v)
	case r.button.selected:
		r.background.FillColor = th.Color(theme.ColorNameSelection, v)
	case r.button.hovered:
		r.background.FillColor = th.Color(theme.ColorNameHover, v)
	default:
		r.background.FillColor = color.Transparent
	}
	r.background.CornerRadius = th.Size(theme.SizeNameInputRadius)
	r.background.Refresh()

	icon := r.button.icon
	if r.button.Disabled() && icon != nil {
		icon = theme.NewDisabledResource(icon)
	}
	r.icon.Resource = icon
	r.icon.Refresh()

	r.Layout(r.button.Size())
}

func (r *iconNavButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *iconNavButtonRenderer) Destroy() {}
