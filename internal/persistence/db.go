package persistence

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
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

func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version;`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if version >= 2 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback()

	if version < 1 {
		stmts := []string{
			`CREATE TABLE IF NOT EXISTS nodes (
				node_id TEXT PRIMARY KEY,
				long_name TEXT,
				short_name TEXT,
				battery_level INTEGER NULL,
				voltage REAL NULL,
				board_model TEXT NULL,
				device_role TEXT NULL,
				last_heard_at INTEGER,
				rssi INTEGER NULL,
				snr REAL NULL,
				updated_at INTEGER NOT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS nodes_last_heard_at_idx ON nodes(last_heard_at DESC);`,
			`CREATE TABLE IF NOT EXISTS chats (
				chat_key TEXT PRIMARY KEY,
				type INTEGER NOT NULL,
				title TEXT NOT NULL,
				last_sent_by_me_at INTEGER NULL,
				updated_at INTEGER NOT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS chats_last_sent_by_me_idx ON chats(last_sent_by_me_at DESC);`,
			`CREATE TABLE IF NOT EXISTS messages (
				local_id INTEGER PRIMARY KEY AUTOINCREMENT,
				chat_key TEXT NOT NULL,
				device_message_id TEXT NULL,
				direction INTEGER NOT NULL,
				body TEXT NOT NULL,
				status INTEGER NOT NULL,
				at INTEGER NOT NULL,
				meta_json TEXT NULL
			);`,
			`CREATE INDEX IF NOT EXISTS messages_chat_at_idx ON messages(chat_key, at ASC);`,
			`CREATE UNIQUE INDEX IF NOT EXISTS messages_chat_device_unique_idx ON messages(chat_key, device_message_id) WHERE device_message_id IS NOT NULL;`,
			`PRAGMA user_version = 2;`,
		}

		for _, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("apply migration statement: %w", err)
			}
		}
	} else {
		stmts := []string{
			`ALTER TABLE nodes ADD COLUMN battery_level INTEGER NULL;`,
			`ALTER TABLE nodes ADD COLUMN voltage REAL NULL;`,
			`ALTER TABLE nodes ADD COLUMN board_model TEXT NULL;`,
			`ALTER TABLE nodes ADD COLUMN device_role TEXT NULL;`,
			`PRAGMA user_version = 2;`,
		}
		for _, stmt := range stmts {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("apply migration statement: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
