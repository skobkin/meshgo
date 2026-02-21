package migrations

import (
	"context"
	"database/sql"
)

func migrateV2AddNodePowerAndDeviceMetadata(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN battery_level INTEGER NULL;`,
		`ALTER TABLE nodes ADD COLUMN voltage REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN board_model TEXT NULL;`,
		`ALTER TABLE nodes ADD COLUMN device_role TEXT NULL;`,
	}

	return applyStatements(ctx, tx, "v2 add node power and device metadata", statements)
}
