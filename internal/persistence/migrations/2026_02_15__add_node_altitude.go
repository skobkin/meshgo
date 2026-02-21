package migrations

import (
	"context"
	"database/sql"
)

func migrateV7AddNodeAltitude(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN altitude INTEGER NULL;`,
	}

	return applyStatements(ctx, tx, "v7 add node altitude", statements)
}
