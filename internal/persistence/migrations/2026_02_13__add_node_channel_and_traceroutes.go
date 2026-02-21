package migrations

import (
	"context"
	"database/sql"
)

func migrateV6AddNodeChannelAndTraceroutes(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN channel INTEGER NULL;`,
		`CREATE TABLE IF NOT EXISTS traceroutes (
			request_id TEXT PRIMARY KEY,
			target_node_id TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			completed_at INTEGER NULL,
			status TEXT NOT NULL,
			forward_route_json TEXT NULL,
			forward_snr_json TEXT NULL,
			return_route_json TEXT NULL,
			return_snr_json TEXT NULL,
			error_text TEXT NULL,
			duration_ms INTEGER NULL
		);`,
		`CREATE INDEX IF NOT EXISTS traceroutes_started_at_idx ON traceroutes(started_at DESC);`,
		`CREATE INDEX IF NOT EXISTS traceroutes_target_started_idx ON traceroutes(target_node_id, started_at DESC);`,
	}

	return applyStatements(ctx, tx, "v6 add node channel and traceroutes", statements)
}
