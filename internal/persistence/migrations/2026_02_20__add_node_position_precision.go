package migrations

import (
	"context"
	"database/sql"
)

func migrateV9AddNodePrecisionBits(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN precision_bits INTEGER NULL;`,
	}

	return applyStatements(ctx, tx, "v9 add node precision bits", statements)
}
