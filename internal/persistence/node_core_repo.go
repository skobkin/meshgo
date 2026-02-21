package persistence

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

// NodeCoreRepo persists and queries node core identity/activity snapshots.
type NodeCoreRepo struct {
	db *sql.DB
}

func NewNodeCoreRepo(db *sql.DB) *NodeCoreRepo {
	return &NodeCoreRepo{db: db}
}

func (r *NodeCoreRepo) Upsert(ctx context.Context, update domain.NodeCoreUpdate, identityHistoryLimit int) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("node core repo is not initialized")
	}
	nodeID := strings.TrimSpace(update.Core.NodeID)
	if nodeID == "" {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin node core upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	prev, prevFound, err := fetchNodeIdentitySnapshot(ctx, tx, nodeID)
	if err != nil {
		return err
	}

	writtenAt := time.Now()
	core := update.Core
	if core.UpdatedAt.IsZero() {
		core.UpdatedAt = writtenAt
	}

	var (
		publicKey       any
		channel         any
		isUnmessageable any
		rssi            any
		snr             any
	)
	if len(core.PublicKey) > 0 {
		publicKey = append([]byte(nil), core.PublicKey...)
	}
	if core.Channel != nil {
		channel = int64(*core.Channel)
	}
	if core.IsUnmessageable != nil {
		if *core.IsUnmessageable {
			isUnmessageable = int64(1)
		} else {
			isUnmessageable = int64(0)
		}
	}
	if core.RSSI != nil {
		rssi = *core.RSSI
	}
	if core.SNR != nil {
		snr = *core.SNR
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO nodes(node_id, long_name, short_name, public_key, channel, board_model, firmware_version, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			long_name = CASE
				WHEN excluded.long_name IS NOT NULL AND excluded.long_name <> '' THEN excluded.long_name
				ELSE nodes.long_name
			END,
			short_name = CASE
				WHEN excluded.short_name IS NOT NULL AND excluded.short_name <> '' THEN excluded.short_name
				ELSE nodes.short_name
			END,
			public_key = CASE
				WHEN excluded.public_key IS NOT NULL AND length(excluded.public_key) > 0 THEN excluded.public_key
				ELSE nodes.public_key
			END,
			channel = COALESCE(excluded.channel, nodes.channel),
			board_model = CASE
				WHEN excluded.board_model IS NOT NULL AND excluded.board_model <> '' THEN excluded.board_model
				ELSE nodes.board_model
			END,
			firmware_version = CASE
				WHEN excluded.firmware_version IS NOT NULL AND excluded.firmware_version <> '' THEN excluded.firmware_version
				ELSE nodes.firmware_version
			END,
			device_role = CASE
				WHEN excluded.device_role IS NOT NULL AND excluded.device_role <> '' THEN excluded.device_role
				ELSE nodes.device_role
			END,
			is_unmessageable = COALESCE(excluded.is_unmessageable, nodes.is_unmessageable),
			last_heard_at = CASE
				WHEN excluded.last_heard_at > nodes.last_heard_at THEN excluded.last_heard_at
				ELSE nodes.last_heard_at
			END,
			rssi = COALESCE(excluded.rssi, nodes.rssi),
			snr = COALESCE(excluded.snr, nodes.snr),
			updated_at = CASE
				WHEN excluded.updated_at > nodes.updated_at THEN excluded.updated_at
				ELSE nodes.updated_at
			END
	`,
		nodeID,
		core.LongName,
		core.ShortName,
		publicKey,
		channel,
		nullableString(core.BoardModel),
		nullableString(core.FirmwareVersion),
		nullableString(core.Role),
		isUnmessageable,
		timeToUnixMillis(core.LastHeardAt),
		rssi,
		snr,
		timeToUnixMillis(core.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert node core: %w", err)
	}

	if isReliableIdentityUpdateSource(update.Type) {
		next, nextFound, fetchErr := fetchNodeIdentitySnapshot(ctx, tx, nodeID)
		if fetchErr != nil {
			return fetchErr
		}
		if nextFound && hasNodeIdentityData(next) && (!prevFound || !nodeIdentityEqual(prev, next)) {
			observedAt := core.LastHeardAt
			if observedAt.IsZero() {
				observedAt = core.UpdatedAt
			}
			if observedAt.IsZero() {
				observedAt = writtenAt
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO node_identity_history(node_id, long_name, short_name, public_key, observed_at, written_at, update_type, from_packet)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`,
				nodeID,
				nullableString(next.LongName),
				nullableString(next.ShortName),
				next.PublicKey,
				timeToUnixMillis(observedAt),
				timeToUnixMillis(writtenAt),
				string(update.Type),
				boolToInt64(update.FromPacket),
			)
			if err != nil {
				return fmt.Errorf("insert node identity history: %w", err)
			}
			if err := pruneHistoryRows(ctx, tx, "node_identity_history", nodeID, identityHistoryLimit); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit node core upsert tx: %w", err)
	}

	return nil
}

func (r *NodeCoreRepo) ListSortedByLastHeard(ctx context.Context) ([]domain.NodeCore, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, long_name, short_name, public_key, channel, board_model, firmware_version, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at
		FROM nodes
		ORDER BY last_heard_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list node core: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.NodeCore, 0)
	for rows.Next() {
		item, scanErr := scanNodeCore(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate node core rows: %w", err)
	}

	return out, nil
}

func (r *NodeCoreRepo) GetByNodeID(ctx context.Context, nodeID string) (domain.NodeCore, bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, long_name, short_name, public_key, channel, board_model, firmware_version, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at
		FROM nodes
		WHERE node_id = ?
		LIMIT 1
	`, strings.TrimSpace(nodeID))
	if err != nil {
		return domain.NodeCore{}, false, fmt.Errorf("query node core by id: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	if !rows.Next() {
		return domain.NodeCore{}, false, nil
	}
	item, scanErr := scanNodeCore(rows)
	if scanErr != nil {
		return domain.NodeCore{}, false, scanErr
	}

	return item, true, nil
}

func scanNodeCore(scanner interface{ Scan(dest ...any) error }) (domain.NodeCore, error) {
	var (
		item          domain.NodeCore
		publicKey     []byte
		channel       sql.NullInt64
		board         sql.NullString
		firmware      sql.NullString
		role          sql.NullString
		unmessageable sql.NullInt64
		heardMS       int64
		rssi          sql.NullInt64
		snr           sql.NullFloat64
		updatedMS     int64
	)
	if err := scanner.Scan(&item.NodeID, &item.LongName, &item.ShortName, &publicKey, &channel, &board, &firmware, &role, &unmessageable, &heardMS, &rssi, &snr, &updatedMS); err != nil {
		return domain.NodeCore{}, fmt.Errorf("scan node core row: %w", err)
	}
	if len(publicKey) > 0 {
		item.PublicKey = append(make([]byte, 0, len(publicKey)), publicKey...)
	}
	if channel.Valid {
		if v, ok := int64ToUint32(channel.Int64); ok {
			item.Channel = &v
		}
	}

	if board.Valid {
		item.BoardModel = board.String
	}
	if firmware.Valid {
		item.FirmwareVersion = firmware.String
	}
	if role.Valid {
		item.Role = role.String
	}
	if unmessageable.Valid {
		v := unmessageable.Int64 != 0
		item.IsUnmessageable = &v
	}
	item.LastHeardAt = unixMillisToTime(heardMS)
	if rssi.Valid {
		v := int(rssi.Int64)
		item.RSSI = &v
	}
	if snr.Valid {
		v := snr.Float64
		item.SNR = &v
	}
	item.UpdatedAt = unixMillisToTime(updatedMS)

	return item, nil
}

type nodeIdentitySnapshot struct {
	LongName  string
	ShortName string
	PublicKey []byte
}

func fetchNodeIdentitySnapshot(ctx context.Context, tx *sql.Tx, nodeID string) (nodeIdentitySnapshot, bool, error) {
	var (
		value     nodeIdentitySnapshot
		publicKey []byte
	)
	err := tx.QueryRowContext(ctx, `
		SELECT long_name, short_name, public_key
		FROM nodes
		WHERE node_id = ?
		LIMIT 1
	`, nodeID).Scan(&value.LongName, &value.ShortName, &publicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nodeIdentitySnapshot{}, false, nil
		}

		return nodeIdentitySnapshot{}, false, fmt.Errorf("query node identity snapshot: %w", err)
	}
	if len(publicKey) > 0 {
		value.PublicKey = append([]byte(nil), publicKey...)
	}

	return value, true, nil
}

func nodeIdentityEqual(left, right nodeIdentitySnapshot) bool {
	return left.LongName == right.LongName &&
		left.ShortName == right.ShortName &&
		bytes.Equal(left.PublicKey, right.PublicKey)
}

func hasNodeIdentityData(value nodeIdentitySnapshot) bool {
	return strings.TrimSpace(value.LongName) != "" ||
		strings.TrimSpace(value.ShortName) != "" ||
		len(value.PublicKey) > 0
}

func isReliableIdentityUpdateSource(updateType domain.NodeUpdateType) bool {
	return updateType == domain.NodeUpdateTypeNodeInfoSnapshot ||
		updateType == domain.NodeUpdateTypeNodeInfoPacket
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}

	return 0
}
