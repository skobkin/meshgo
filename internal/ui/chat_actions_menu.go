package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

// ChatAction identifies a message-level action available from context menu.
type ChatAction string

const (
	ChatActionReply ChatAction = "reply"
)

// ChatActionHandler handles selected chat message context action.
type ChatActionHandler func(message domain.ChatMessage, action ChatAction)

func newChatMessageContextMenu(message domain.ChatMessage, onAction ChatActionHandler) *fyne.Menu {
	title := "Message"
	if body := strings.TrimSpace(message.Body); body != "" {
		title = body
	}
	itemReply := fyne.NewMenuItem("Reply", func() {
		if onAction != nil {
			onAction(message, ChatActionReply)
		}
	})
	if !canReplyToMessage(message) {
		itemReply.Disabled = true
	}

	return fyne.NewMenu(title, itemReply)
}

func showChatMessageContextMenu(
	fyneCanvas fyne.Canvas,
	position fyne.Position,
	message domain.ChatMessage,
	onAction ChatActionHandler,
) {
	if fyneCanvas == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(newChatMessageContextMenu(message, onAction), fyneCanvas, position)
}
