package resources

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func TestUIIconResource_AllIconsMappedForBothVariants(t *testing.T) {
	icons := []UIIcon{
		UIIconChats,
		UIIconNodes,
		UIIconMap,
		UIIconNodeSettings,
		UIIconAppSettings,
		UIIconConnected,
		UIIconDisconnected,
		UIIconMapNodeMarker,
		UIIconUpdateAvailable,
	}

	variants := []struct {
		name    string
		variant fyne.ThemeVariant
	}{
		{name: "light", variant: theme.VariantLight},
		{name: "dark", variant: theme.VariantDark},
	}

	for _, icon := range icons {
		for _, variant := range variants {
			if got := UIIconResource(icon, variant.variant); got == nil {
				t.Fatalf("expected %s icon resource for %q variant", icon, variant.name)
			}
		}
	}
}
