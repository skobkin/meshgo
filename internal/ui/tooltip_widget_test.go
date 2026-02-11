package ui

import (
	"net/url"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func TestTooltipPopupPosition(t *testing.T) {
	gap := theme.Padding()
	tests := []struct {
		name      string
		anchorPos fyne.Position
		anchor    fyne.Size
		popup     fyne.Size
		canvas    fyne.Size
		want      fyne.Position
	}{
		{
			name:      "fits below",
			anchorPos: fyne.NewPos(10, 20),
			anchor:    fyne.NewSize(30, 10),
			popup:     fyne.NewSize(40, 20),
			canvas:    fyne.NewSize(200, 200),
			want:      fyne.NewPos(10, 20+10+gap),
		},
		{
			name:      "falls back above when below overflows",
			anchorPos: fyne.NewPos(30, 90),
			anchor:    fyne.NewSize(20, 10),
			popup:     fyne.NewSize(50, 30),
			canvas:    fyne.NewSize(200, 120),
			want:      fyne.NewPos(30, 90-30-gap),
		},
		{
			name:      "clamps right edge",
			anchorPos: fyne.NewPos(170, 20),
			anchor:    fyne.NewSize(20, 10),
			popup:     fyne.NewSize(60, 20),
			canvas:    fyne.NewSize(200, 200),
			want:      fyne.NewPos(200-60-gap, 20+10+gap),
		},
		{
			name:      "clamps top when popup too tall",
			anchorPos: fyne.NewPos(30, 10),
			anchor:    fyne.NewSize(20, 10),
			popup:     fyne.NewSize(100, 300),
			canvas:    fyne.NewSize(200, 200),
			want:      fyne.NewPos(30, 0),
		},
		{
			name:      "clamps left edge with padding",
			anchorPos: fyne.NewPos(0, 20),
			anchor:    fyne.NewSize(20, 10),
			popup:     fyne.NewSize(60, 20),
			canvas:    fyne.NewSize(200, 200),
			want:      fyne.NewPos(gap, 20+10+gap),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tooltipPopupPosition(tc.anchorPos, tc.anchor, tc.popup, tc.canvas)
			if got != tc.want {
				t.Fatalf("unexpected position: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestTooltipWidgetBuildTooltip_Label(t *testing.T) {
	w := newTooltipLabel("✓", "Sent", nil)

	got := w.buildTooltip()
	label, ok := got.(*widget.Label)
	if !ok {
		t.Fatalf("expected label tooltip, got %T", got)
	}
	if label.Text != "Sent" {
		t.Fatalf("unexpected tooltip text: %q", label.Text)
	}

	w.SetBadge("", "Sent")
	if got := w.buildTooltip(); got != nil {
		t.Fatalf("expected nil tooltip for empty badge text, got %T", got)
	}
}

func TestTooltipWidgetBuildTooltip_RichText(t *testing.T) {
	w := newTooltipRichText(
		[]widget.RichTextSegment{
			&widget.TextSegment{Text: "▆", Style: widget.RichTextStyleInline},
		},
		[]widget.RichTextSegment{
			&widget.TextSegment{Text: "RSSI: ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: "-98", Style: widget.RichTextStyleInline},
		},
		nil,
	)

	got := w.buildTooltip()
	rich, ok := got.(*widget.RichText)
	if !ok {
		t.Fatalf("expected richtext tooltip, got %T", got)
	}
	if len(rich.Segments) != 2 {
		t.Fatalf("unexpected tooltip segment count: %d", len(rich.Segments))
	}

	w.rich.Segments = []widget.RichTextSegment{
		&widget.TextSegment{Text: " ", Style: widget.RichTextStyleInline},
	}
	if got := w.buildTooltip(); got != nil {
		t.Fatalf("expected nil tooltip for empty rich badge content, got %T", got)
	}
}

func TestHoverTooltipManagerHideOwnerSemantics(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	layer := container.NewWithoutLayout()
	manager := newHoverTooltipManager(layer)
	owner := newTooltipLabel("✓", "Sent", manager)
	other := newTooltipLabel("☁", "via MQTT", manager)
	root := container.New(layout.NewStackLayout(), container.NewHBox(owner, other), layer)

	win := app.NewWindow("tooltip")
	win.SetContent(root)
	win.Resize(fyne.NewSize(320, 200))
	win.Show()

	manager.Show(owner, widget.NewLabel("tip"))
	if len(layer.Objects) != 1 {
		t.Fatalf("expected tooltip to be visible, got %d objects", len(layer.Objects))
	}

	manager.Hide(other)
	if len(layer.Objects) != 1 {
		t.Fatalf("tooltip should remain visible on mismatched owner hide, got %d objects", len(layer.Objects))
	}

	manager.Hide(owner)
	if len(layer.Objects) != 0 {
		t.Fatalf("expected tooltip to be hidden, got %d objects", len(layer.Objects))
	}
}

func TestHideTooltipWidgets_RecursivelyHidesOwnedTooltip(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	layer := container.NewWithoutLayout()
	manager := newHoverTooltipManager(layer)
	owner := newTooltipLabel("✓", "Sent", manager)
	root := container.New(layout.NewStackLayout(), container.NewVBox(owner), layer)

	win := app.NewWindow("tooltip")
	win.SetContent(root)
	win.Resize(fyne.NewSize(320, 200))
	win.Show()

	manager.Show(owner, widget.NewLabel("tip"))
	if len(layer.Objects) != 1 {
		t.Fatalf("expected tooltip to be visible, got %d objects", len(layer.Objects))
	}

	hideTooltipWidgets([]fyne.CanvasObject{container.NewHBox(owner)})
	if len(layer.Objects) != 0 {
		t.Fatalf("expected tooltip to be hidden by recursive cleanup, got %d objects", len(layer.Objects))
	}
}

func TestCloneRichTextSegments_DeepCopy(t *testing.T) {
	linkURL, err := url.Parse("https://example.org")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	top := &widget.TextSegment{Text: "top", Style: widget.RichTextStyleInline}
	paraInner := &widget.TextSegment{Text: "inner", Style: widget.RichTextStyleInline}
	para := &widget.ParagraphSegment{Texts: []widget.RichTextSegment{paraInner}}
	link := &widget.HyperlinkSegment{Text: "link", URL: linkURL}
	listInner := &widget.TextSegment{Text: "item", Style: widget.RichTextStyleInline}
	list := &widget.ListSegment{Items: []widget.RichTextSegment{listInner}, Ordered: true}
	list.SetStartNumber(3)

	clone := cloneRichTextSegments([]widget.RichTextSegment{top, para, link, list})

	top.Text = "mut-top"
	paraInner.Text = "mut-inner"
	link.URL.Host = "mutated.invalid"
	listInner.Text = "mut-item"
	list.SetStartNumber(9)

	clonedTop, ok := clone[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("unexpected clone type for top: %T", clone[0])
	}
	if clonedTop.Text != "top" {
		t.Fatalf("top segment should be independent copy, got %q", clonedTop.Text)
	}

	clonedPara, ok := clone[1].(*widget.ParagraphSegment)
	if !ok {
		t.Fatalf("unexpected clone type for paragraph: %T", clone[1])
	}
	clonedParaInner, ok := clonedPara.Texts[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("unexpected paragraph inner type: %T", clonedPara.Texts[0])
	}
	if clonedParaInner.Text != "inner" {
		t.Fatalf("paragraph inner segment should be independent copy, got %q", clonedParaInner.Text)
	}

	clonedLink, ok := clone[2].(*widget.HyperlinkSegment)
	if !ok {
		t.Fatalf("unexpected clone type for link: %T", clone[2])
	}
	if clonedLink.URL == nil || clonedLink.URL.Host != "example.org" {
		t.Fatalf("hyperlink URL should be independent copy, got %v", clonedLink.URL)
	}

	clonedList, ok := clone[3].(*widget.ListSegment)
	if !ok {
		t.Fatalf("unexpected clone type for list: %T", clone[3])
	}
	clonedListInner, ok := clonedList.Items[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("unexpected list inner type: %T", clonedList.Items[0])
	}
	if clonedListInner.Text != "item" {
		t.Fatalf("list inner segment should be independent copy, got %q", clonedListInner.Text)
	}
	if clonedList.StartNumber() != 3 {
		t.Fatalf("list start number should be independent copy, got %d", clonedList.StartNumber())
	}
}
