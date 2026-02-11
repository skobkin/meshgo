package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*tooltipLabel)(nil)

type tooltipLabel struct {
	widget.Label

	tooltip string
	manager *hoverTooltipManager
	hovered bool
	hide    *time.Timer
}

const tooltipHideDelay = 200 * time.Millisecond

func newTooltipLabel(text, tooltip string, manager *hoverTooltipManager) *tooltipLabel {
	l := &tooltipLabel{tooltip: strings.TrimSpace(tooltip), manager: manager}
	l.Label = *widget.NewLabel(text)
	l.ExtendBaseWidget(l)

	return l
}

func (l *tooltipLabel) SetBadge(text, tooltip string) {
	if text == "" || strings.TrimSpace(tooltip) == "" {
		l.hideTooltip()
	}
	l.hideTooltip()

	l.tooltip = strings.TrimSpace(tooltip)
	l.SetText(text)
}

func (l *tooltipLabel) MouseIn(*desktop.MouseEvent) {
	if strings.TrimSpace(l.Text) == "" || l.tooltip == "" {
		return
	}
	l.hovered = true
	l.cancelHide()
	l.manager.Show(l, widget.NewLabel(l.tooltip))
}

func (l *tooltipLabel) MouseMoved(*desktop.MouseEvent) {
}

func (l *tooltipLabel) MouseOut() {
	l.hovered = false
	l.scheduleHide()
}

func (l *tooltipLabel) hideTooltip() {
	l.cancelHide()
	l.manager.Hide(l)
}

func (l *tooltipLabel) scheduleHide() {
	l.cancelHide()
	l.hide = time.AfterFunc(tooltipHideDelay, func() {
		fyne.Do(func() {
			if l.hovered {
				return
			}
			l.hideTooltip()
		})
	})
}

func (l *tooltipLabel) cancelHide() {
	if l.hide == nil {
		return
	}

	l.hide.Stop()
	l.hide = nil
}
