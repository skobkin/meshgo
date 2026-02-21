package migrations

import (
	"context"
	"database/sql"
)

func migrateV3AddNodeUnmessageableFlag(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN is_unmessageable INTEGER NULL;`,
	}

	return applyStatements(ctx, tx, "v3 add node unmessageable flag", statements)
}
