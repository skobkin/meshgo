package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

type hoverTooltipManager struct {
	layer *fyne.Container
	owner fyne.CanvasObject
}

func newHoverTooltipManager(layer *fyne.Container) *hoverTooltipManager {
	if layer == nil {
		return nil
	}

	return &hoverTooltipManager{layer: layer}
}

func (m *hoverTooltipManager) Show(owner fyne.CanvasObject, content fyne.CanvasObject) {
	if m == nil || m.layer == nil || owner == nil || content == nil {
		return
	}

	app := fyne.CurrentApp()
	if app == nil {
		return
	}

	driver := app.Driver()
	if driver == nil {
		return
	}

	cnv := driver.CanvasForObject(owner)
	if cnv == nil {
		return
	}
	layerSize := m.layer.Size()
	layerPos := driver.AbsolutePositionForObject(m.layer)
	ownerPos := driver.AbsolutePositionForObject(owner).Subtract(layerPos)
	if layerSize.Width <= 0 || layerSize.Height <= 0 {
		layerSize = cnv.Size()
		ownerPos = driver.AbsolutePositionForObject(owner)
	}

	bubble := newTooltipBubble(content)
	bubble.Resize(bubble.MinSize())
	bubble.Move(tooltipPopupPosition(
		ownerPos,
		owner.Size(),
		bubble.Size(),
		layerSize,
	))

	m.layer.Objects = []fyne.CanvasObject{bubble}
	m.owner = owner
	m.layer.Refresh()
}

func (m *hoverTooltipManager) Hide(owner fyne.CanvasObject) {
	if m == nil || m.layer == nil || m.owner == nil {
		return
	}
	if owner != nil && owner != m.owner {
		return
	}

	m.layer.Objects = nil
	m.owner = nil
	m.layer.Refresh()
}

func newTooltipBubble(content fyne.CanvasObject) *fyne.Container {
	bgColor := theme.DefaultTheme().Color(theme.ColorNameOverlayBackground, theme.VariantDark)
	if app := fyne.CurrentApp(); app != nil {
		bgColor = app.Settings().Theme().Color(theme.ColorNameOverlayBackground, app.Settings().ThemeVariant())
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = theme.Padding()

	return container.NewStack(bg, container.NewPadded(content))
}

func tooltipPopupPosition(anchorPos fyne.Position, anchorSize, popupSize, canvasSize fyne.Size) fyne.Position {
	gap := theme.Padding()
	edge := theme.Padding()
	x := anchorPos.X
	yBelow := anchorPos.Y + anchorSize.Height + gap
	yAbove := anchorPos.Y - popupSize.Height - gap

	minX := edge
	maxX := canvasSize.Width - popupSize.Width - edge
	if maxX < minX {
		minX = 0
		maxX = canvasSize.Width - popupSize.Width
	}
	if maxX < 0 {
		maxX = 0
	}

	minY := edge
	maxY := canvasSize.Height - popupSize.Height - edge
	if maxY < minY {
		minY = 0
		maxY = canvasSize.Height - popupSize.Height
	}
	if maxY < 0 {
		maxY = 0
	}

	availableBelow := maxY - yBelow
	availableAbove := yAbove - minY

	y := yBelow
	switch {
	case yBelow <= maxY:
		// Keep default below-anchor position.
	case yAbove >= minY:
		y = yAbove
	default:
		// Choose side with more visible space when popup cannot fully fit either side.
		if availableAbove > availableBelow {
			y = minY
		} else {
			y = maxY
		}
	}

	if x > maxX {
		x = maxX
	}
	if x < minX {
		x = minX
	}
	if y > maxY {
		y = maxY
	}
	if y < minY {
		y = minY
	}

	return fyne.NewPos(x, y)
}
