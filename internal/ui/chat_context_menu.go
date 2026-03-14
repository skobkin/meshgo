package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

type chatListAction string

const (
	chatListActionShare  chatListAction = "share"
	chatListActionDelete chatListAction = "delete"
)

type chatListActionHandler func(chat domain.Chat, action chatListAction)

func newChatListContextMenu(chat domain.Chat, onAction chatListActionHandler) *fyne.Menu {
	title := strings.TrimSpace(chatDisplayTitle(chat, nil))
	if title == "" {
		title = "Chat"
	}

	items := make([]*fyne.MenuItem, 0, 2)
	if !domain.IsDMChat(chat) {
		items = append(items, fyne.NewMenuItem("Share", func() {
			if onAction != nil {
				onAction(chat, chatListActionShare)
			}
		}))
	}

	deleteItem := fyne.NewMenuItem("Delete chat", func() {
		if onAction != nil {
			onAction(chat, chatListActionDelete)
		}
	})
	if !domain.IsDMChat(chat) {
		deleteItem.Disabled = true
	}
	items = append(items, deleteItem)

	return fyne.NewMenu(title, items...)
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
