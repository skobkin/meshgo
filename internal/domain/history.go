package domain

import "time"

// NodeHistoryQuery defines paginated per-node history reads.
type NodeHistoryQuery struct {
	NodeID           string
	Limit            int
	BeforeObservedAt time.Time
	BeforeRowID      int64
	Order            SortOrder
}
