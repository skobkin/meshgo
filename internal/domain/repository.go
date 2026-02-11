package domain

import "context"

// NodeRepository persists node snapshots.
type NodeRepository interface {
	Upsert(ctx context.Context, n Node) error
	ListSortedByLastHeard(ctx context.Context) ([]Node, error)
}

// ChatRepository persists chat metadata.
type ChatRepository interface {
	Upsert(ctx context.Context, c Chat) error
	ListSortedByLastSentByMe(ctx context.Context) ([]Chat, error)
}

// MessageRepository persists chat messages and delivery statuses.
type MessageRepository interface {
	Insert(ctx context.Context, m ChatMessage) (int64, error)
	LoadRecentPerChat(ctx context.Context, limit int) (map[string][]ChatMessage, error)
	UpdateStatusByDeviceMessageID(ctx context.Context, deviceMessageID string, status MessageStatus) error
}
