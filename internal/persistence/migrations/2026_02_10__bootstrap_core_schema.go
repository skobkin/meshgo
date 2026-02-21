package migrations

import (
	"context"
	"database/sql"
)

func migrateV1BootstrapCoreSchema(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			node_id TEXT PRIMARY KEY,
			long_name TEXT,
			short_name TEXT,
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
	}

	return applyStatements(ctx, tx, "v1 bootstrap core schema", statements)
}
