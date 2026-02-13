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

func ChatTypeForKey(key string) ChatType {
	if strings.HasPrefix(strings.TrimSpace(key), "dm:") {
		return ChatTypeDM
	}

	return ChatTypeChannel
}

func NodeIDFromDMChatKey(key string) string {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "dm:") {
		return ""
	}

	return strings.TrimPrefix(key, "dm:")
}
