package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

func historyOrderSQL(order domain.HistorySortOrder) string {
	if order == domain.HistorySortAscending {
		return "ASC"
	}

	return "DESC"
}

func historyLimitValue(limit int) int {
	if limit <= 0 {
		return 100
	}

	return limit
}

func applyHistoryCursor(base string, query domain.NodeHistoryQuery, args []any) (string, []any) {
	if query.BeforeObservedAt.IsZero() {
		return base, args
	}

	observedMS := timeToUnixMillis(query.BeforeObservedAt)
	if query.Order == domain.HistorySortAscending {
		base += " AND (observed_at > ? OR (observed_at = ? AND id > ?))"
		args = append(args, observedMS, observedMS, query.BeforeRowID)

		return base, args
	}

	base += " AND (observed_at < ? OR (observed_at = ? AND id < ?))"
	args = append(args, observedMS, observedMS, query.BeforeRowID)

	return base, args
}

func pruneHistoryRows(ctx context.Context, tx *sql.Tx, table, nodeID string, limit int) error {
	if limit <= 0 {
		return nil
	}
	safeTable := strings.TrimSpace(table)
	switch safeTable {
	case "node_position_history", "node_telemetry_history", "node_identity_history":
	default:
		return fmt.Errorf("unsafe history table name: %q", safeTable)
	}
	_, err := tx.ExecContext(
		ctx,
		fmt.Sprintf(`
			DELETE FROM %s
			WHERE id IN (
				SELECT id FROM %s
				WHERE node_id = ?
				ORDER BY observed_at DESC, id DESC
				LIMIT -1 OFFSET ?
			)
		`, safeTable, safeTable),
		nodeID,
		limit,
	)
	if err != nil {
		return fmt.Errorf("prune %s rows for node %s: %w", safeTable, nodeID, err)
	}

	return nil
}
