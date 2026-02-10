package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*tooltipLabel)(nil)

type tooltipLabel struct {
	widget.Label

	tooltip string
	popup   *widget.PopUp
}

func newTooltipLabel(text, tooltip string) *tooltipLabel {
	l := &tooltipLabel{tooltip: strings.TrimSpace(tooltip)}
	l.Label = *widget.NewLabel(text)
	l.ExtendBaseWidget(l)

	return l
}

func (l *tooltipLabel) SetBadge(text, tooltip string) {
	if text == "" || strings.TrimSpace(tooltip) == "" {
		l.hideTooltip()
	}

	l.tooltip = strings.TrimSpace(tooltip)
	l.SetText(text)
}

func (l *tooltipLabel) MouseIn(*desktop.MouseEvent) {
	if strings.TrimSpace(l.Text) == "" || l.tooltip == "" {
		return
	}

	canvas := fyne.CurrentApp().Driver().CanvasForObject(l)
	if canvas == nil {
		return
	}

	l.hideTooltip()
	l.popup = widget.NewPopUp(widget.NewLabel(l.tooltip), canvas)
	l.popup.ShowAtRelativePosition(fyne.NewPos(0, l.Size().Height+theme.Padding()), l)
}

func (l *tooltipLabel) MouseMoved(*desktop.MouseEvent) {
}

func (l *tooltipLabel) MouseOut() {
	l.hideTooltip()
}

func (l *tooltipLabel) hideTooltip() {
	if l.popup == nil {
		return
	}

	l.popup.Hide()
	l.popup = nil
}
