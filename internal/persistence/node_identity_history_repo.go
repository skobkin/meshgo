package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeIdentityHistoryRepo reads historical node identity snapshots.
type NodeIdentityHistoryRepo struct {
	db *sql.DB
}

func NewNodeIdentityHistoryRepo(db *sql.DB) *NodeIdentityHistoryRepo {
	return &NodeIdentityHistoryRepo{db: db}
}

func (r *NodeIdentityHistoryRepo) ListHistoryByNodeID(ctx context.Context, query domain.NodeHistoryQuery) ([]domain.NodeIdentityHistoryEntry, error) {
	nodeID := strings.TrimSpace(query.NodeID)
	if nodeID == "" {
		return nil, nil
	}
	order := historyOrderSQL(query.Order)
	where := "WHERE node_id = ?"
	args := []any{nodeID}
	where, args = applyHistoryCursor(where, query, args)
	limit := historyLimitValue(query.Limit)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, node_id, long_name, short_name, public_key, observed_at, written_at, update_type, from_packet
		FROM node_identity_history
		%s
		ORDER BY observed_at %s, id %s
		LIMIT ?
	`, where, order, order), append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("list node identity history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.NodeIdentityHistoryEntry, 0)
	for rows.Next() {
		item, scanErr := scanNodeIdentityHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node identity history rows: %w", err)
	}

	return out, nil
}

func scanNodeIdentityHistory(scanner interface{ Scan(dest ...any) error }) (domain.NodeIdentityHistoryEntry, error) {
	var (
		item       domain.NodeIdentityHistoryEntry
		longName   sql.NullString
		shortName  sql.NullString
		publicKey  []byte
		observedMS int64
		writtenMS  int64
		updateType string
		fromPacket int64
	)
	if err := scanner.Scan(&item.RowID, &item.NodeID, &longName, &shortName, &publicKey, &observedMS, &writtenMS, &updateType, &fromPacket); err != nil {
		return domain.NodeIdentityHistoryEntry{}, fmt.Errorf("scan node identity history row: %w", err)
	}
	if longName.Valid {
		item.LongName = longName.String
	}
	if shortName.Valid {
		item.ShortName = shortName.String
	}
	if len(publicKey) > 0 {
		item.PublicKey = append([]byte(nil), publicKey...)
	}
	item.ObservedAt = unixMillisToTime(observedMS)
	item.WrittenAt = unixMillisToTime(writtenMS)
	item.UpdateType = domain.NodeUpdateType(strings.TrimSpace(updateType))
	item.FromPacket = fromPacket != 0

	return item, nil
}
