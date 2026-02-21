package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodePositionRepo persists and queries node position snapshots and history.
type NodePositionRepo struct {
	db *sql.DB
}

func NewNodePositionRepo(db *sql.DB) *NodePositionRepo {
	return &NodePositionRepo{db: db}
}

func (r *NodePositionRepo) Upsert(ctx context.Context, update domain.NodePositionUpdate, historyLimit int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("node position repo is not initialized")
	}
	nodeID := strings.TrimSpace(update.Position.NodeID)
	if nodeID == "" {
		return nil
	}
	incoming := update.Position

	writtenAt := time.Now()
	if incoming.ObservedAt.IsZero() {
		incoming.ObservedAt = incoming.UpdatedAt
	}
	if incoming.ObservedAt.IsZero() {
		incoming.ObservedAt = writtenAt
	}
	if incoming.UpdatedAt.IsZero() {
		incoming.UpdatedAt = writtenAt
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin node position upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO nodes(node_id, last_heard_at, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(node_id) DO NOTHING
	`, nodeID, timeToUnixMillis(incoming.ObservedAt), timeToUnixMillis(incoming.UpdatedAt))
	if err != nil {
		return fmt.Errorf("ensure node core row for position: %w", err)
	}

	existing, found, err := fetchNodePositionLatest(ctx, tx, nodeID)
	if err != nil {
		return err
	}
	next := mergeNodePosition(existing, incoming)
	if next.ObservedAt.IsZero() {
		next.ObservedAt = incoming.ObservedAt
	}
	if next.UpdatedAt.IsZero() {
		next.UpdatedAt = incoming.UpdatedAt
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO node_position_latest(node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at, update_type, from_packet)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			channel = COALESCE(excluded.channel, node_position_latest.channel),
			latitude = COALESCE(excluded.latitude, node_position_latest.latitude),
			longitude = COALESCE(excluded.longitude, node_position_latest.longitude),
			altitude = COALESCE(excluded.altitude, node_position_latest.altitude),
			precision_bits = COALESCE(excluded.precision_bits, node_position_latest.precision_bits),
			position_updated_at = CASE
				WHEN excluded.position_updated_at IS NOT NULL AND (
					node_position_latest.position_updated_at IS NULL OR excluded.position_updated_at > node_position_latest.position_updated_at
				) THEN excluded.position_updated_at
				ELSE node_position_latest.position_updated_at
			END,
			observed_at = excluded.observed_at,
			written_at = excluded.written_at,
			update_type = excluded.update_type,
			from_packet = excluded.from_packet
	`,
		nodeID,
		nullableUint32(next.Channel),
		nullableFloat64(next.Latitude),
		nullableFloat64(next.Longitude),
		nullableInt32(next.Altitude),
		nullableUint32(next.PositionPrecisionBits),
		nullableTime(next.PositionUpdatedAt),
		timeToUnixMillis(next.ObservedAt),
		timeToUnixMillis(writtenAt),
		string(update.Type),
		boolToInt64(update.FromPacket),
	)
	if err != nil {
		return fmt.Errorf("upsert node position latest: %w", err)
	}

	if hasPositionCoordinates(next) && (!found || !nodePositionEqual(existing, next)) {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO node_position_history(node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at, update_type, from_packet)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			nodeID,
			nullableUint32(next.Channel),
			nullableFloat64(next.Latitude),
			nullableFloat64(next.Longitude),
			nullableInt32(next.Altitude),
			nullableUint32(next.PositionPrecisionBits),
			nullableTime(next.PositionUpdatedAt),
			timeToUnixMillis(next.ObservedAt),
			timeToUnixMillis(writtenAt),
			string(update.Type),
			boolToInt64(update.FromPacket),
		)
		if err != nil {
			return fmt.Errorf("insert node position history: %w", err)
		}
		if err := pruneHistoryRows(ctx, tx, "node_position_history", nodeID, historyLimit); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit node position upsert tx: %w", err)
	}

	return nil
}

func (r *NodePositionRepo) ListLatest(ctx context.Context) ([]domain.NodePosition, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at
		FROM node_position_latest
	`)
	if err != nil {
		return nil, fmt.Errorf("list node position latest: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	out := make([]domain.NodePosition, 0)
	for rows.Next() {
		item, scanErr := scanNodePositionLatest(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node position latest rows: %w", err)
	}

	return out, nil
}

func (r *NodePositionRepo) GetLatestByNodeID(ctx context.Context, nodeID string) (domain.NodePosition, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at
		FROM node_position_latest
		WHERE node_id = ?
		LIMIT 1
	`, strings.TrimSpace(nodeID))
	if err != nil {
		return domain.NodePosition{}, false, fmt.Errorf("query node position latest by id: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if !rows.Next() {
		return domain.NodePosition{}, false, nil
	}
	item, scanErr := scanNodePositionLatest(rows)
	if scanErr != nil {
		return domain.NodePosition{}, false, scanErr
	}

	return item, true, nil
}

func (r *NodePositionRepo) ListHistoryByNodeID(ctx context.Context, query domain.NodeHistoryQuery) ([]domain.NodePositionHistoryEntry, error) {
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
		SELECT id, node_id, channel, latitude, longitude, altitude, precision_bits, observed_at, written_at, update_type, from_packet
		FROM node_position_history
		%s
		ORDER BY observed_at %s, id %s
		LIMIT ?
	`, where, order, order), append(args, limit)...)
	if err != nil {
		return nil, fmt.Errorf("list node position history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.NodePositionHistoryEntry, 0)
	for rows.Next() {
		item, scanErr := scanNodePositionHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node position history rows: %w", err)
	}

	return out, nil
}

func fetchNodePositionLatest(ctx context.Context, tx *sql.Tx, nodeID string) (domain.NodePosition, bool, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT node_id, channel, latitude, longitude, altitude, precision_bits, position_updated_at, observed_at, written_at
		FROM node_position_latest
		WHERE node_id = ?
		LIMIT 1
	`, nodeID)
	if err != nil {
		return domain.NodePosition{}, false, fmt.Errorf("query existing node position latest: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if !rows.Next() {
		return domain.NodePosition{}, false, nil
	}
	item, scanErr := scanNodePositionLatest(rows)
	if scanErr != nil {
		return domain.NodePosition{}, false, scanErr
	}

	return item, true, nil
}

func scanNodePositionLatest(scanner interface{ Scan(dest ...any) error }) (domain.NodePosition, error) {
	var (
		item          domain.NodePosition
		channel       sql.NullInt64
		latitude      sql.NullFloat64
		longitude     sql.NullFloat64
		altitude      sql.NullInt64
		precisionBits sql.NullInt64
		positionMS    sql.NullInt64
		observedMS    int64
		writtenMS     int64
	)
	if err := scanner.Scan(&item.NodeID, &channel, &latitude, &longitude, &altitude, &precisionBits, &positionMS, &observedMS, &writtenMS); err != nil {
		return domain.NodePosition{}, fmt.Errorf("scan node position latest row: %w", err)
	}
	if channel.Valid {
		if v, ok := int64ToUint32(channel.Int64); ok {
			item.Channel = &v
		}
	}
	if latitude.Valid {
		v := latitude.Float64
		item.Latitude = &v
	}
	if longitude.Valid {
		v := longitude.Float64
		item.Longitude = &v
	}
	if altitude.Valid {
		if v, ok := int64ToInt32(altitude.Int64); ok {
			item.Altitude = &v
		}
	}
	if precisionBits.Valid {
		if v, ok := int64ToUint32(precisionBits.Int64); ok {
			item.PositionPrecisionBits = &v
		}
	}
	if positionMS.Valid {
		item.PositionUpdatedAt = unixMillisToTime(positionMS.Int64)
	}
	item.ObservedAt = unixMillisToTime(observedMS)
	item.UpdatedAt = unixMillisToTime(writtenMS)

	return item, nil
}

func scanNodePositionHistory(scanner interface{ Scan(dest ...any) error }) (domain.NodePositionHistoryEntry, error) {
	var (
		item          domain.NodePositionHistoryEntry
		channel       sql.NullInt64
		latitude      sql.NullFloat64
		longitude     sql.NullFloat64
		altitude      sql.NullInt64
		precisionBits sql.NullInt64
		observedMS    int64
		writtenMS     int64
		updateType    string
		fromPacket    int64
	)
	if err := scanner.Scan(&item.RowID, &item.NodeID, &channel, &latitude, &longitude, &altitude, &precisionBits, &observedMS, &writtenMS, &updateType, &fromPacket); err != nil {
		return domain.NodePositionHistoryEntry{}, fmt.Errorf("scan node position history row: %w", err)
	}
	if channel.Valid {
		if v, ok := int64ToUint32(channel.Int64); ok {
			item.Channel = &v
		}
	}
	if latitude.Valid {
		v := latitude.Float64
		item.Latitude = &v
	}
	if longitude.Valid {
		v := longitude.Float64
		item.Longitude = &v
	}
	if altitude.Valid {
		if v, ok := int64ToInt32(altitude.Int64); ok {
			item.Altitude = &v
		}
	}
	if precisionBits.Valid {
		if v, ok := int64ToUint32(precisionBits.Int64); ok {
			item.Precision = &v
		}
	}
	item.ObservedAt = unixMillisToTime(observedMS)
	item.WrittenAt = unixMillisToTime(writtenMS)
	item.UpdateType = domain.NodeUpdateType(strings.TrimSpace(updateType))
	item.FromPacket = fromPacket != 0

	return item, nil
}

func mergeNodePosition(existing, incoming domain.NodePosition) domain.NodePosition {
	next := existing
	if strings.TrimSpace(next.NodeID) == "" {
		next.NodeID = incoming.NodeID
	}
	if incoming.Channel != nil {
		next.Channel = incoming.Channel
	}
	if incoming.Latitude != nil {
		next.Latitude = incoming.Latitude
	}
	if incoming.Longitude != nil {
		next.Longitude = incoming.Longitude
	}
	if incoming.Altitude != nil {
		next.Altitude = incoming.Altitude
	}
	if incoming.PositionPrecisionBits != nil {
		next.PositionPrecisionBits = incoming.PositionPrecisionBits
	}
	if !incoming.PositionUpdatedAt.IsZero() && (next.PositionUpdatedAt.IsZero() || incoming.PositionUpdatedAt.After(next.PositionUpdatedAt)) {
		next.PositionUpdatedAt = incoming.PositionUpdatedAt
	}
	if !incoming.ObservedAt.IsZero() {
		next.ObservedAt = incoming.ObservedAt
	}
	if !incoming.UpdatedAt.IsZero() {
		next.UpdatedAt = incoming.UpdatedAt
	}

	return next
}

func hasPositionCoordinates(value domain.NodePosition) bool {
	return value.Latitude != nil && value.Longitude != nil
}

func nodePositionEqual(left, right domain.NodePosition) bool {
	return nullableUint32Equal(left.Channel, right.Channel) &&
		nullableFloat64Equal(left.Latitude, right.Latitude) &&
		nullableFloat64Equal(left.Longitude, right.Longitude) &&
		nullableInt32Equal(left.Altitude, right.Altitude) &&
		nullableUint32Equal(left.PositionPrecisionBits, right.PositionPrecisionBits) &&
		left.PositionUpdatedAt.Equal(right.PositionUpdatedAt)
}

func nullableUint32(v *uint32) any {
	if v == nil {
		return nil
	}

	return int64(*v)
}

func nullableInt32(v *int32) any {
	if v == nil {
		return nil
	}

	return int64(*v)
}

func nullableFloat64(v *float64) any {
	if v == nil {
		return nil
	}

	return *v
}

func nullableTime(v time.Time) any {
	if v.IsZero() {
		return nil
	}

	return timeToUnixMillis(v)
}

func nullableUint32Equal(left, right *uint32) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func nullableInt32Equal(left, right *int32) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

func nullableFloat64Equal(left, right *float64) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}
