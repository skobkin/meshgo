package ui

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func TestManagedNodeSettingsReloadPreservesDirtyStateUntilSuccess(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	type loadResult struct {
		value string
		err   error
	}

	results := make(chan loadResult)
	loadStarted := make(chan struct{}, 3)
	dep := newNodeSettingsRuntimeDeps(&nodeSettingsActionSpy{})
	dep.UIHooks.ShowErrorDialog = func(error, fyne.Window) {}

	var entry *widget.Entry
	page, onOpened := newManagedNodeSettingsPage(
		dep,
		nil,
		"test.managed",
		"Loading test settings...",
		"Test settings loaded.",
		func(context.Context, app.NodeSettingsTarget) (string, error) {
			loadStarted <- struct{}{}
			result := <-results

			return result.value, result.err
		},
		func(context.Context, app.NodeSettingsTarget, string) error { return nil },
		func(value string) string { return value },
		func(value string) string { return value },
		func(onChanged func()) nodeManagedSettingsForm[string] {
			entry = widget.NewEntry()
			entry.OnChanged = func(string) { onChanged() }

			return nodeManagedSettingsForm[string]{
				content: entry,
				set:     entry.SetText,
				read: func(string, app.NodeSettingsTarget) (string, error) {
					return entry.Text, nil
				},
			}
		},
	)
	_ = fynetest.NewTempWindow(t, page)

	saveButton := mustFindButtonByText(t, page, "Save")
	cancelButton := mustFindButtonByText(t, page, "Cancel")
	reloadButton := mustFindButtonByText(t, page, "Reload")

	onOpened()
	<-loadStarted
	assertManagedSettingsButtons(t, saveButton, cancelButton, reloadButton, true, true, true)

	results <- loadResult{value: "baseline"}
	waitForCondition(t, func() bool {
		return entry.Text == "baseline" && saveButton.Disabled() && cancelButton.Disabled() && !reloadButton.Disabled()
	})

	fyne.DoAndWait(func() {
		entry.SetText("edited")
	})
	waitForCondition(t, func() bool {
		return !saveButton.Disabled() && !cancelButton.Disabled()
	})

	fynetest.Tap(reloadButton)
	<-loadStarted
	assertManagedSettingsButtons(t, saveButton, cancelButton, reloadButton, true, true, true)
	if entry.Text != "edited" {
		t.Fatalf("expected pending reload to retain edited value, got %q", entry.Text)
	}

	results <- loadResult{err: errors.New("device unavailable")}
	waitForCondition(t, func() bool {
		return entry.Text == "edited" && !saveButton.Disabled() && !cancelButton.Disabled() && !reloadButton.Disabled()
	})

	fynetest.Tap(reloadButton)
	<-loadStarted
	results <- loadResult{value: "fresh"}
	waitForCondition(t, func() bool {
		return entry.Text == "fresh" && saveButton.Disabled() && cancelButton.Disabled() && !reloadButton.Disabled()
	})
}

func assertManagedSettingsButtons(
	t *testing.T,
	saveButton *widget.Button,
	cancelButton *widget.Button,
	reloadButton *widget.Button,
	saveDisabled bool,
	cancelDisabled bool,
	reloadDisabled bool,
) {
	t.Helper()
	fyne.DoAndWait(func() {
		if saveButton.Disabled() != saveDisabled {
			t.Errorf("unexpected Save disabled state: got %t want %t", saveButton.Disabled(), saveDisabled)
		}
		if cancelButton.Disabled() != cancelDisabled {
			t.Errorf("unexpected Cancel disabled state: got %t want %t", cancelButton.Disabled(), cancelDisabled)
		}
		if reloadButton.Disabled() != reloadDisabled {
			t.Errorf("unexpected Reload disabled state: got %t want %t", reloadButton.Disabled(), reloadDisabled)
		}
	})
}
