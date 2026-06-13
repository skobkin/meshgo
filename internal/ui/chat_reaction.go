package ui

import (
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

// openReactionPicker computes an anchor above the message row and shows
// the reaction picker pop-up. The user's selection is dispatched as a
// reaction packet through the supplied sender.
//
// The anchor is the row's top-center (absolute), so the pop-up hovers
// above the message bubble regardless of where on the bubble the user
// right-clicked. Fyne's widget.PopUp clamps to the canvas if the
// computed anchor would push the pop-up off-screen.
//
// The function is a no-op (besides logging) if the message does not
// satisfy canReactToMessage, the sender is nil, or the message has no
// chat key / target id.
func openReactionPicker(
	fyneCanvas fyne.Canvas,
	rowItem fyne.CanvasObject,
	message domain.ChatMessage,
	sender MessageSender,
	statusLabel *widget.Label,
	logger *slog.Logger,
) {
	if fyneCanvas == nil || sender == nil || !canReactToMessage(message) {
		return
	}
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	driver := app.Driver()
	if driver == nil {
		return
	}
	rowPos := driver.AbsolutePositionForObject(rowItem)
	rowSize := rowItem.Size()
	anchor := fyne.NewPos(rowPos.X+rowSize.Width/2, rowPos.Y)

	chatKey := strings.TrimSpace(message.ChatKey)
	targetID := strings.TrimSpace(message.DeviceMessageID)
	if chatKey == "" || targetID == "" {
		return
	}

	showReactionPicker(fyneCanvas, anchor, func(emoji string) {
		opts := radio.TextSendOptions{
			Emoji:                  1,
			ReplyToDeviceMessageID: targetID,
		}
		logger.Info("sending reaction", "chat_key", chatKey, "emoji", emoji, "target_message_id", targetID)
		go func() {
			res := <-sender.SendText(chatKey, emoji, opts)
			if res.Err != nil {
				fyne.Do(func() {
					logger.Warn("reaction send failed", "chat_key", chatKey, "emoji", emoji, "target_message_id", targetID, "error", res.Err)
					if statusLabel != nil {
						statusLabel.SetText("Reaction failed: " + res.Err.Error())
					}
				})

				return
			}
			fyne.Do(func() {
				logger.Info("reaction sent", "chat_key", chatKey, "emoji", emoji, "target_message_id", targetID)
			})
		}()
	})
}
