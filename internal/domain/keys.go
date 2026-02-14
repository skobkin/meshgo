package domain

import (
	"fmt"
	"strings"
)

func ChatKeyForChannel(index int) string {
	return fmt.Sprintf("channel:%d", index)
}

func ChatKeyForDM(nodeID string) string {
	return "dm:" + nodeID
}

func IsDMKey(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), "dm:")
}

func ChatTypeForKey(key string) ChatType {
	if IsDMKey(key) {
		return ChatTypeDM
	}

	return ChatTypeChannel
}

func IsDMChat(chat Chat) bool {
	if chat.Type == ChatTypeDM {
		return true
	}

	return IsDMKey(chat.Key)
}

func NodeIDFromDMChatKey(key string) string {
	key = strings.TrimSpace(key)
	if !IsDMKey(key) {
		return ""
	}

	return strings.TrimPrefix(key, "dm:")
}
