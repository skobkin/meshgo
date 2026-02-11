package ui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*tooltipRichText)(nil)

type tooltipRichText struct {
	widget.RichText

	tooltip []widget.RichTextSegment
	manager *hoverTooltipManager
	hovered bool
	hide    *time.Timer
}

func newTooltipRichText(segments []widget.RichTextSegment, tooltip []widget.RichTextSegment, manager *hoverTooltipManager) *tooltipRichText {
	rt := &tooltipRichText{
		tooltip: append([]widget.RichTextSegment(nil), tooltip...),
		manager: manager,
	}
	rt.RichText = *widget.NewRichText(segments...)
	rt.ExtendBaseWidget(rt)

	return rt
}

func (rt *tooltipRichText) MouseIn(*desktop.MouseEvent) {
	if len(rt.tooltip) == 0 {
		return
	}
	if strings.TrimSpace(richTextSegmentsPlainText(rt.Segments)) == "" {
		return
	}
	rt.hovered = true
	rt.cancelHide()
	tooltip := widget.NewRichText(rt.tooltip...)
	rt.manager.Show(rt, tooltip)
}

func (rt *tooltipRichText) MouseMoved(*desktop.MouseEvent) {
}

func (rt *tooltipRichText) MouseOut() {
	rt.hovered = false
	rt.scheduleHide()
}

func (rt *tooltipRichText) hideTooltip() {
	rt.cancelHide()
	rt.manager.Hide(rt)
}

func (rt *tooltipRichText) scheduleHide() {
	rt.cancelHide()
	rt.hide = time.AfterFunc(tooltipHideDelay, func() {
		fyne.Do(func() {
			if rt.hovered {
				return
			}
			rt.hideTooltip()
		})
	})
}

func (rt *tooltipRichText) cancelHide() {
	if rt.hide == nil {
		return
	}

	rt.hide.Stop()
	rt.hide = nil
}

func richTextSegmentsPlainText(segs []widget.RichTextSegment) string {
	var b strings.Builder
	for _, seg := range segs {
		text, ok := seg.(*widget.TextSegment)
		if !ok {
			continue
		}
		b.WriteString(text.Text)
	}

	return b.String()
}
