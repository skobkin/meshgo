package layout

import "fyne.io/fyne/v2"

const chatRowWidthRatio float32 = 0.8

type ChatRowLayout struct {
	alignRight bool
}

func NewChatRowLayout(alignRight bool) *ChatRowLayout {
	return &ChatRowLayout{alignRight: alignRight}
}

func (l *ChatRowLayout) SetAlignRight(v bool) {
	l.alignRight = v
}

func (l *ChatRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	row := objects[0]
	rowSize := fyne.NewSize(chatRowWidth(size.Width, row.MinSize().Width), size.Height)
	x := float32(0)
	if l.alignRight {
		x = size.Width - rowSize.Width
	}

	row.Move(fyne.NewPos(x, 0))
	row.Resize(rowSize)
}

func (l *ChatRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}

	return objects[0].MinSize()
}

func chatRowWidth(totalWidth, minWidth float32) float32 {
	targetWidth := totalWidth * chatRowWidthRatio
	if targetWidth < minWidth {
		targetWidth = minWidth
	}
	if targetWidth > totalWidth {
		targetWidth = totalWidth
	}

	return targetWidth
}
