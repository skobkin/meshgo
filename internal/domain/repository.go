package domain

import "context"

// NodeCoreRepository persists node core snapshots.
type NodeCoreRepository interface {
	Upsert(ctx context.Context, update NodeCoreUpdate, identityHistoryLimit int) error
	ListSortedByLastHeard(ctx context.Context) ([]NodeCore, error)
	GetByNodeID(ctx context.Context, nodeID string) (NodeCore, bool, error)
}

// NodePositionRepository persists node position latest snapshot and history.
type NodePositionRepository interface {
	Upsert(ctx context.Context, update NodePositionUpdate, historyLimit int) error
	ListLatest(ctx context.Context) ([]NodePosition, error)
	GetLatestByNodeID(ctx context.Context, nodeID string) (NodePosition, bool, error)
	ListHistoryByNodeID(ctx context.Context, query NodeHistoryQuery) ([]NodePositionHistoryEntry, error)
}

// NodeTelemetryRepository persists node telemetry latest snapshot and history.
type NodeTelemetryRepository interface {
	Upsert(ctx context.Context, update NodeTelemetryUpdate, historyLimit int) error
	ListLatest(ctx context.Context) ([]NodeTelemetry, error)
	GetLatestByNodeID(ctx context.Context, nodeID string) (NodeTelemetry, bool, error)
	ListHistoryByNodeID(ctx context.Context, query NodeHistoryQuery) ([]NodeTelemetryHistoryEntry, error)
}

// NodeIdentityHistoryRepository persists identity change history for nodes.
type NodeIdentityHistoryRepository interface {
	ListHistoryByNodeID(ctx context.Context, query NodeHistoryQuery) ([]NodeIdentityHistoryEntry, error)
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

// TracerouteRepository persists traceroute request/response snapshots.
type TracerouteRepository interface {
	Upsert(ctx context.Context, rec TracerouteRecord) error
}
