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
	ChatActionReact ChatAction = "react"
)

// ChatActionHandler handles selected chat message context action.
type ChatActionHandler func(message domain.ChatMessage, action ChatAction)

const chatMenuTitleMaxLen = 32

func newChatMessageContextMenu(message domain.ChatMessage, onAction ChatActionHandler) *fyne.Menu {
	title := "Message"
	if body := strings.TrimSpace(message.Body); body != "" {
		title = body
		if len(title) > chatMenuTitleMaxLen {
			title = title[:chatMenuTitleMaxLen-1] + "…"
		}
	}

	itemReply := fyne.NewMenuItem("Reply", func() {
		if onAction != nil {
			onAction(message, ChatActionReply)
		}
	})
	if !canReplyToMessage(message) {
		itemReply.Disabled = true
	}

	itemReact := fyne.NewMenuItem("Add reaction", func() {
		if onAction != nil {
			onAction(message, ChatActionReact)
		}
	})
	if !canReactToMessage(message) {
		itemReact.Disabled = true
	}

	return fyne.NewMenu(title, itemReply, itemReact)
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
