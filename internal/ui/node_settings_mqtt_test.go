package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestNodeMQTTSettingsLegacyJSONControlIsReadOnly(t *testing.T) {
	page, _ := newNodeMQTTSettingsPage(RuntimeDependencies{}, nil)

	var jsonCheck *widget.Check
	walkCanvasObjects(page, func(object fyne.CanvasObject) bool {
		form, ok := object.(*widget.Form)
		if !ok {
			return false
		}
		for _, item := range form.Items {
			if item.Text != "JSON output enabled (legacy, read-only)" {
				continue
			}
			jsonCheck, ok = item.Widget.(*widget.Check)
			if !ok {
				t.Fatalf("expected legacy JSON form item to contain a check, got %T", item.Widget)
			}

			return true
		}

		return false
	})

	if jsonCheck == nil {
		t.Fatal("legacy JSON control not found")
	}
	if !jsonCheck.Disabled() {
		t.Fatal("legacy JSON control must be read-only")
	}
}
