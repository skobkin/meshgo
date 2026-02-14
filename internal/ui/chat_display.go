package ui

import (
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

func chatTypeLabel(chat domain.Chat) string {
	if domain.IsDMChat(chat) {
		return "DM"
	}

	return "Channel"
}

func chatDisplayTitle(chat domain.Chat, nodeNameByID func(string) string) string {
	defaultTitle := domain.ChatDisplayTitle(chat)
	if !domain.IsDMChat(chat) {
		return defaultTitle
	}

	nodeID := domain.NormalizeNodeID(domain.NodeIDFromDMChatKey(chat.Key))
	if nodeID == "" {
		return defaultTitle
	}

	if nodeNameByID != nil {
		if display := strings.TrimSpace(nodeNameByID(nodeID)); display != "" && display != nodeID {
			return display
		}
	}

	if custom := strings.TrimSpace(chat.Title); custom != "" && custom != strings.TrimSpace(chat.Key) {
		return custom
	}

	return nodeID
}

func chatTitleByKey(chats []domain.Chat, key string, nodeNameByID func(string) string) string {
	if key == "" {
		return "No chat selected"
	}
	for _, chat := range chats {
		if chat.Key == key {
			return chatDisplayTitle(chat, nodeNameByID)
		}
	}

	return key
}
