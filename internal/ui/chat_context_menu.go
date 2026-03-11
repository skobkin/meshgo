package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

type chatListAction string

const (
	chatListActionDelete chatListAction = "delete"
)

type chatListActionHandler func(chat domain.Chat, action chatListAction)

func newChatListContextMenu(chat domain.Chat, onAction chatListActionHandler) *fyne.Menu {
	title := strings.TrimSpace(chatDisplayTitle(chat, nil))
	if title == "" {
		title = "Chat"
	}

	deleteItem := fyne.NewMenuItem("Delete chat", func() {
		if onAction != nil {
			onAction(chat, chatListActionDelete)
		}
	})
	if !domain.IsDMChat(chat) {
		deleteItem.Disabled = true
	}

	return fyne.NewMenu(title, deleteItem)
}

func showChatListContextMenu(
	fyneCanvas fyne.Canvas,
	position fyne.Position,
	chat domain.Chat,
	onAction chatListActionHandler,
) {
	if fyneCanvas == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(newChatListContextMenu(chat, onAction), fyneCanvas, position)
}
