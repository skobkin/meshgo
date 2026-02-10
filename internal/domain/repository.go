package domain

import "context"

type NodeRepository interface {
	Upsert(ctx context.Context, n Node) error
	ListSortedByLastHeard(ctx context.Context) ([]Node, error)
}

type ChatRepository interface {
	Upsert(ctx context.Context, c Chat) error
	ListSortedByLastSentByMe(ctx context.Context) ([]Chat, error)
}

type MessageRepository interface {
	Insert(ctx context.Context, m ChatMessage) (int64, error)
	LoadRecentPerChat(ctx context.Context, limit int) (map[string][]ChatMessage, error)
}
