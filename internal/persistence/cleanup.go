package persistence

import (
	"context"
	"database/sql"
	"fmt"
)

//goland:noinspection SqlWithoutWhere
var clearDatabaseStatements = []string{
	`DELETE FROM messages;`,
	`DELETE FROM chats;`,
	`DELETE FROM nodes;`,
	`DELETE FROM traceroutes;`,
}

func ClearDatabase(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clear database tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, stmt := range clearDatabaseStatements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("clear database tables: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clear database tx: %w", err)
	}

	return nil
}
