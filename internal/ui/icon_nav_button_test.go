package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
)

func TestIconNavButtonMinSizeWithText(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	button := newIconNavButton(nil, nil)
	base := button.MinSize()

	button.SetText("0.7.0")
	withText := button.MinSize()

	if withText.Height <= base.Height {
		t.Fatalf("expected text button height to grow: base=%v withText=%v", base, withText)
	}
	if withText.Width < base.Width {
		t.Fatalf("expected text button width not to shrink: base=%v withText=%v", base, withText)
	}
}
