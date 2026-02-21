package migrations

import (
	"context"
	"database/sql"
)

func migrateV8AddMessageRepliesAndReactions(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`ALTER TABLE messages ADD COLUMN reply_to_device_message_id TEXT NULL;`,
		`ALTER TABLE messages ADD COLUMN emoji INTEGER NOT NULL DEFAULT 0;`,
	}

	return applyStatements(ctx, tx, "v8 add message replies and reactions", statements)
}
