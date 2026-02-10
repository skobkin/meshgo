package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/skobkin/meshgo/internal/domain"
)

type NodeRepo struct {
	db *sql.DB
}

func NewNodeRepo(db *sql.DB) *NodeRepo {
	return &NodeRepo{db: db}
}

func (r *NodeRepo) Upsert(ctx context.Context, n domain.Node) error {
	var (
		batteryLevel    any
		voltage         any
		isUnmessageable any
	)
	if n.BatteryLevel != nil {
		batteryLevel = int64(*n.BatteryLevel)
	}
	if n.Voltage != nil {
		voltage = *n.Voltage
	}
	if n.IsUnmessageable != nil {
		if *n.IsUnmessageable {
			isUnmessageable = int64(1)
		} else {
			isUnmessageable = int64(0)
		}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO nodes(node_id, long_name, short_name, battery_level, voltage, board_model, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			long_name = excluded.long_name,
			short_name = excluded.short_name,
			battery_level = excluded.battery_level,
			voltage = excluded.voltage,
			board_model = excluded.board_model,
			device_role = excluded.device_role,
			is_unmessageable = excluded.is_unmessageable,
			last_heard_at = excluded.last_heard_at,
			rssi = excluded.rssi,
			snr = excluded.snr,
			updated_at = excluded.updated_at
	`, n.NodeID, n.LongName, n.ShortName, batteryLevel, voltage, nullableString(n.BoardModel), nullableString(n.Role), isUnmessageable, toUnixMillis(n.LastHeardAt), n.RSSI, n.SNR, toUnixMillis(n.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert node: %w", err)
	}
	return nil
}

func (r *NodeRepo) ListSortedByLastHeard(ctx context.Context) ([]domain.Node, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT node_id, long_name, short_name, battery_level, voltage, board_model, device_role, is_unmessageable, last_heard_at, rssi, snr, updated_at
		FROM nodes
		ORDER BY last_heard_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var out []domain.Node
	for rows.Next() {
		var (
			n             domain.Node
			heardMs       int64
			updMs         int64
			battery       sql.NullInt64
			voltage       sql.NullFloat64
			board         sql.NullString
			role          sql.NullString
			unmessageable sql.NullInt64
			rssi          sql.NullInt64
			snr           sql.NullFloat64
		)
		if err := rows.Scan(&n.NodeID, &n.LongName, &n.ShortName, &battery, &voltage, &board, &role, &unmessageable, &heardMs, &rssi, &snr, &updMs); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		n.LastHeardAt = fromUnixMillis(heardMs)
		n.UpdatedAt = fromUnixMillis(updMs)
		if battery.Valid {
			v := uint32(battery.Int64)
			n.BatteryLevel = &v
		}
		if voltage.Valid {
			v := voltage.Float64
			n.Voltage = &v
		}
		if board.Valid {
			n.BoardModel = board.String
		}
		if role.Valid {
			n.Role = role.String
		}
		if unmessageable.Valid {
			v := unmessageable.Int64 != 0
			n.IsUnmessageable = &v
		}
		if rssi.Valid {
			v := int(rssi.Int64)
			n.RSSI = &v
		}
		if snr.Valid {
			v := snr.Float64
			n.SNR = &v
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}
	return out, nil
}
