package migrations

import (
	"context"
	"database/sql"
)

func migrateV5AddNodeCoordinates(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN latitude REAL NULL;`,
		`ALTER TABLE nodes ADD COLUMN longitude REAL NULL;`,
	}

	return applyStatements(ctx, tx, "v5 add node coordinates", statements)
}
