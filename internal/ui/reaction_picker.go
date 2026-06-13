package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// defaultReactionEmojis is the curated list of reactions available from
// the chat message context menu. The order is roughly positive → joy →
// sad → negative → love → agree → disagree → hype → acknowledge → unsure.
//
// The list is hardcoded for v1; a user-configurable picker will be
// tracked in a separate issue. See https://github.com/skobkin/meshgo/issues/51
// for the original discussion.
var defaultReactionEmojis = []string{
	"😀",    // smile / acknowledge
	"😂",    // joy / funny
	"😢",    // sad / sympathy
	"😡",    // angry / disagree-strong
	"❤️",   // love / strong agree
	"👍",    // agree
	"👎",    // disagree
	"🔥",    // hot / important
	"🫡",    // salute / formal acknowledge
	"🤷‍♂️", // shrug / idk
}

// newReactionPickerContent builds the chip-strip content shown inside the
// reaction picker pop-up. It is exposed so the picker layout and
// callback wiring can be unit-tested without standing up a Fyne canvas
// or a real pop-up.
//
// The returned container holds one button per defaultReactionEmojis entry.
// Tapping a button invokes onPick with the matching emoji string.
func newReactionPickerContent(onPick func(emoji string)) fyne.CanvasObject {
	buttons := make([]fyne.CanvasObject, 0, len(defaultReactionEmojis))
	for _, emoji := range defaultReactionEmojis {
		emoji := emoji
		btn := widget.NewButton(emoji, func() {
			if onPick != nil {
				onPick(emoji)
			}
		})
		btn.Importance = widget.LowImportance
		buttons = append(buttons, btn)
	}

	return container.NewHBox(buttons...)
}

// showReactionPicker shows a small pop-up anchored to the given canvas
// position. The onPick callback is invoked with the chosen emoji when
// the user clicks one of the buttons; the pop-up is hidden and the Esc
// shortcut removed at that point. Esc and click-outside (Fyne's default
// behaviour for widget.PopUp) dismiss the pop-up without invoking
// onPick.
//
// The picker re-registers its Esc shortcut on each invocation and
// removes it on dismiss, so a new picker cleanly replaces the previous
// one and there is no handler leak.
func showReactionPicker(fyneCanvas fyne.Canvas, anchor fyne.Position, onPick func(emoji string)) {
	if fyneCanvas == nil {
		return
	}

	var popup *widget.PopUp
	escShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyEscape}

	dismiss := func() {
		if popup != nil {
			popup.Hide()
		}
		fyneCanvas.RemoveShortcut(escShortcut)
	}

	popup = widget.NewPopUp(newReactionPickerContent(func(emoji string) {
		dismiss()
		if onPick != nil {
			onPick(emoji)
		}
	}), fyneCanvas)

	fyneCanvas.AddShortcut(escShortcut, func(fyne.Shortcut) {
		dismiss()
	})

	popup.ShowAtPosition(anchor)
}
