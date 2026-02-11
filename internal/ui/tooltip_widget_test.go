package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
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
