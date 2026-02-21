package migrations

import (
	"context"
	"database/sql"
)

func migrateV11AddNodePublicKey(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE nodes ADD COLUMN public_key BLOB NULL;`,
	}

	return applyStatements(ctx, tx, "v11 add node public key", statements)
}
