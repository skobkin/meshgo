package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func TestDefaultReactionEmojis_ListContract(t *testing.T) {
	// The picker exposes a curated 10-emoji strip; the exact set is
	// part of the UI contract, not just a stylistic choice. Locking the
	// count and the two emoji swaps requested in the issue thread
	// (🫡 replaces ✨, 🤷‍♂️ replaces ❓) prevents accidental drift.
	if got, want := len(defaultReactionEmojis), 10; got != want {
		t.Fatalf("expected %d emojis, got %d (%v)", want, got, defaultReactionEmojis)
	}

	seen := make(map[string]struct{}, len(defaultReactionEmojis))
	for _, emoji := range defaultReactionEmojis {
		if emoji == "" {
			t.Fatalf("defaultReactionEmojis contains an empty entry")
		}
		if _, dup := seen[emoji]; dup {
			t.Fatalf("defaultReactionEmojis contains duplicate entry %q", emoji)
		}
		seen[emoji] = struct{}{}
	}

	for _, must := range []string{"🫡", "🤷‍♂️"} {
		if _, ok := seen[must]; !ok {
			t.Fatalf("expected defaultReactionEmojis to include %q", must)
		}
	}
	for _, banned := range []string{"✨", "❓"} {
		if _, ok := seen[banned]; ok {
			t.Fatalf("defaultReactionEmojis must not include %q (replaced per issue #51)", banned)
		}
	}
}

func TestNewReactionPickerContent_TenButtons(t *testing.T) {
	content := newReactionPickerContent(nil)

	cont, ok := content.(*fyne.Container)
	if !ok {
		t.Fatalf("expected *fyne.Container, got %T", content)
	}
	if got, want := len(cont.Objects), len(defaultReactionEmojis); got != want {
		t.Fatalf("expected %d children, got %d", want, got)
	}
	for i, child := range cont.Objects {
		btn, ok := child.(*widget.Button)
		if !ok {
			t.Fatalf("child %d: expected *widget.Button, got %T", i, child)
		}
		if btn.Text != defaultReactionEmojis[i] {
			t.Fatalf("child %d: expected text %q, got %q", i, defaultReactionEmojis[i], btn.Text)
		}
		if btn.Importance != widget.LowImportance {
			t.Fatalf("child %d: expected LowImportance, got %v", i, btn.Importance)
		}
	}
}

func TestNewReactionPickerContent_InvokesCallbackWithEmoji(t *testing.T) {
	var picked []string
	content := newReactionPickerContent(func(emoji string) {
		picked = append(picked, emoji)
	})

	cont, ok := content.(*fyne.Container)
	if !ok {
		t.Fatalf("expected *fyne.Container, got %T", content)
	}

	for i, child := range cont.Objects {
		btn := child.(*widget.Button)
		// Invoke the registered tap closure directly. This is the
		// same callback Fyne dispatches for a real click.
		btn.OnTapped()
		if got := picked[len(picked)-1]; got != defaultReactionEmojis[i] {
			t.Fatalf("button %d: expected callback to receive %q, got %q", i, defaultReactionEmojis[i], got)
		}
	}
	if got, want := len(picked), len(defaultReactionEmojis); got != want {
		t.Fatalf("expected %d picks, got %d", want, got)
	}
}

func TestNewReactionPickerContent_NilCallback(t *testing.T) {
	// Constructing the picker with a nil callback must not panic, and
	// tapping a button afterwards must be a no-op.
	content := newReactionPickerContent(nil)
	cont := content.(*fyne.Container)
	for _, child := range cont.Objects {
		btn := child.(*widget.Button)
		btn.OnTapped() // should not panic
	}
}

func TestShowReactionPicker_NilCanvasIsNoOp(t *testing.T) {
	// A nil canvas must not panic and must not invoke the callback.
	called := false
	showReactionPicker(nil, fyne.NewPos(0, 0), func(string) {
		called = true
	})
	if called {
		t.Fatalf("expected callback not to fire with nil canvas")
	}
}
