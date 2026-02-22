package migrations

import (
	"context"
	"database/sql"
)

func migrateV14AddNodeFavoriteFlag(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN is_favorite INTEGER NULL;`,
	}

	return applyStatements(ctx, tx, "v14 add node favorite flag", statements)
}
