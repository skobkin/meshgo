package resources

import (
	"testing"

	"fyne.io/fyne/v2/theme"
)

func TestUIIconResourceUpdateAvailable(t *testing.T) {
	if got := UIIconResource(UIIconUpdateAvailable, theme.VariantLight); got == nil {
		t.Fatalf("expected light update icon resource")
	}
	if got := UIIconResource(UIIconUpdateAvailable, theme.VariantDark); got == nil {
		t.Fatalf("expected dark update icon resource")
	}
}
