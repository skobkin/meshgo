package migrations

import (
	"context"
	"database/sql"
)

func migrateV10AddNodeRuntimeAndFirmwareFields(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN uptime_seconds INTEGER NULL;`,
		`ALTER TABLE nodes ADD COLUMN channel_utilization REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN air_util_tx REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN firmware_version TEXT NULL;`,
		`ALTER TABLE nodes ADD COLUMN position_updated_at INTEGER NULL;`,
	}

	return applyStatements(ctx, tx, "v10 add node runtime and firmware fields", statements)
}
