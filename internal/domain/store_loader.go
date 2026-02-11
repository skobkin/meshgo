package domain

import (
	"context"
	"fmt"
)

const defaultRecentMessagesLoad = 200

func LoadStoresFromRepositories(ctx context.Context, nodes *NodeStore, chats *ChatStore, nodeRepo NodeRepository, chatRepo ChatRepository, msgRepo MessageRepository) error {
	nodeItems, err := nodeRepo.ListSortedByLastHeard(ctx)
	if err != nil {
		return fmt.Errorf("load nodes from db: %w", err)
	}
	chatItems, err := chatRepo.ListSortedByLastSentByMe(ctx)
	if err != nil {
		return fmt.Errorf("load chats from db: %w", err)
	}
	messageItems, err := msgRepo.LoadRecentPerChat(ctx, defaultRecentMessagesLoad)
	if err != nil {
		return fmt.Errorf("load messages from db: %w", err)
	}

	nodes.Load(nodeItems)
	chats.Load(chatItems, messageItems)

	return nil
}
