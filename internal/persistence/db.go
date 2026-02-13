package persistence

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // register sqlite driver
)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("set wal mode: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()

		return nil, err
	}

	return db, nil
}
