package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*tooltipRichText)(nil)

type tooltipRichText struct {
	widget.RichText

	tooltip []widget.RichTextSegment
	popup   *widget.PopUp
}

func newTooltipRichText(segments []widget.RichTextSegment, tooltip []widget.RichTextSegment) *tooltipRichText {
	rt := &tooltipRichText{
		tooltip: append([]widget.RichTextSegment(nil), tooltip...),
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

	canvas := fyne.CurrentApp().Driver().CanvasForObject(rt)
	if canvas == nil {
		return
	}

	rt.hideTooltip()
	tooltip := widget.NewRichText(rt.tooltip...)
	rt.popup = widget.NewPopUp(tooltip, canvas)
	rt.popup.ShowAtRelativePosition(fyne.NewPos(0, rt.Size().Height+theme.Padding()), rt)
}

func (rt *tooltipRichText) MouseMoved(*desktop.MouseEvent) {
}

func (rt *tooltipRichText) MouseOut() {
	rt.hideTooltip()
}

func (rt *tooltipRichText) hideTooltip() {
	if rt.popup == nil {
		return
	}

	rt.popup.Hide()
	rt.popup = nil
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
