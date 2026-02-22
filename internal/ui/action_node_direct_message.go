package ui

import "github.com/skobkin/meshgo/internal/domain"

func handleNodeDirectMessageAction(
	dep RuntimeDependencies,
	switchToChats func(),
	requestOpenChat func(chatKey string),
	node domain.Node,
) {
	if dep.Data.ChatStore == nil {
		appLogger.Warn("direct message action ignored: chat store is unavailable")

		return
	}

	nodeID := domain.NormalizeNodeID(node.NodeID)
	if nodeID == "" {
		appLogger.Warn("direct message action ignored: invalid node id", "node_id", node.NodeID)

		return
	}

	chatKey := domain.ChatKeyForDM(nodeID)
	dep.Data.ChatStore.UpsertChat(domain.Chat{
		Key:   chatKey,
		Title: chatKey,
		Type:  domain.ChatTypeDM,
	})
	if switchToChats != nil {
		switchToChats()
	}
	if requestOpenChat != nil {
		requestOpenChat(chatKey)
	}
}
