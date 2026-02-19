package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// IconNavButton is a navigation button with an icon and optional text label.
type IconNavButton struct {
	widget.DisableableWidget

	icon     fyne.Resource
	iconSize float32
	text     string
	onTap    func()
	selected bool
	hovered  bool
}

const defaultIconNavButtonSize float32 = 48

func NewIconNavButton(icon fyne.Resource, onTap func()) *IconNavButton {
	b := &IconNavButton{
		icon:     icon,
		iconSize: defaultIconNavButtonSize,
		onTap:    onTap,
	}
	b.ExtendBaseWidget(b)

	return b
}

func (b *IconNavButton) SetIcon(icon fyne.Resource) {
	b.icon = icon
	b.Refresh()
}

func (b *IconNavButton) Text() string {
	return b.text
}

func (b *IconNavButton) SetText(text string) {
	if b.text == text {
		return
	}
	b.text = text
	b.Refresh()
}

func (b *IconNavButton) Icon() fyne.Resource {
	return b.icon
}

func (b *IconNavButton) SetSelected(selected bool) {
	if b.selected == selected {
		return
	}
	b.selected = selected
	b.Refresh()
}

func (b *IconNavButton) MinSize() fyne.Size {
	th := b.Theme()
	pad := th.Size(theme.SizeNamePadding) * 2
	side := b.iconSize + pad
	if b.text == "" {
		return fyne.NewSize(side, side)
	}

	label := canvas.NewText(b.text, color.Transparent)
	label.TextSize = th.Size(theme.SizeNameText) * 0.9
	labelSize := label.MinSize()

	width := side
	if labelSize.Width+pad > width {
		width = labelSize.Width + pad
	}

	height := side + labelSize.Height + th.Size(theme.SizeNamePadding)/2

	return fyne.NewSize(width, height)
}

func (b *IconNavButton) Tapped(_ *fyne.PointEvent) {
	if b.Disabled() {
		return
	}
	if b.onTap != nil {
		b.onTap()
	}
}

func (b *IconNavButton) TappedSecondary(_ *fyne.PointEvent) {}

func (b *IconNavButton) MouseIn(_ *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *IconNavButton) MouseMoved(_ *desktop.MouseEvent) {}

func (b *IconNavButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *IconNavButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = b.Theme().Size(theme.SizeNameInputRadius)

	img := canvas.NewImageFromResource(b.icon)
	img.FillMode = canvas.ImageFillContain
	label := canvas.NewText("", color.Transparent)
	label.Alignment = fyne.TextAlignCenter

	return &iconNavButtonRenderer{
		button:     b,
		background: bg,
		icon:       img,
		label:      label,
		objects:    []fyne.CanvasObject{bg, img, label},
	}
}

type iconNavButtonRenderer struct {
	button     *IconNavButton
	background *canvas.Rectangle
	icon       *canvas.Image
	label      *canvas.Text
	objects    []fyne.CanvasObject
}

func (r *iconNavButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)

	pad := r.button.Theme().Size(theme.SizeNamePadding)
	maxW := size.Width - pad*2
	maxH := size.Height - pad*2
	if maxW < 0 {
		maxW = 0
	}
	if maxH < 0 {
		maxH = 0
	}

	if r.button.text == "" {
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
		r.label.Move(fyne.NewPos(0, size.Height))
		r.label.Resize(fyne.NewSize(0, 0))

		return
	}

	gap := pad / 2
	labelHeight := r.label.MinSize().Height
	iconMaxH := maxH - labelHeight - gap
	if iconMaxH < 0 {
		iconMaxH = 0
	}

	iconSide := r.button.iconSize
	if maxW < iconSide {
		iconSide = maxW
	}
	if iconMaxH < iconSide {
		iconSide = iconMaxH
	}
	if iconSide < 0 {
		iconSide = 0
	}

	contentHeight := iconSide + gap + labelHeight
	top := (size.Height - contentHeight) / 2
	if top < 0 {
		top = 0
	}

	iconSize := fyne.NewSquareSize(iconSide)
	r.icon.Resize(iconSize)
	r.icon.Move(fyne.NewPos(
		(size.Width-iconSize.Width)/2,
		top,
	))
	r.label.Resize(fyne.NewSize(maxW, labelHeight))
	r.label.Move(fyne.NewPos(pad, top+iconSide+gap))
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
	r.label.Text = r.button.text
	r.label.TextSize = th.Size(theme.SizeNameText) * 0.9
	if r.button.Disabled() {
		r.label.Color = th.Color(theme.ColorNameDisabled, v)
	} else {
		r.label.Color = th.Color(theme.ColorNameForeground, v)
	}
	if r.button.text == "" {
		r.label.Hide()
	} else {
		r.label.Show()
	}
	r.label.Refresh()

	r.Layout(r.button.Size())
}

func (r *iconNavButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *iconNavButtonRenderer) Destroy() {}
