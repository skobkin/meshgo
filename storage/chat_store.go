package storage

import (
	"context"
	"database/sql"

	"meshgo/domain"
	_ "modernc.org/sqlite"
)

// ChatStore persists chat metadata to SQLite.
type ChatStore struct {
	db *sql.DB
}

// OpenChatStore opens the SQLite database at the provided path.
func OpenChatStore(path string) (*ChatStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &ChatStore{db: db}, nil
}

// Init ensures the schema exists.
func (s *ChatStore) Init(ctx context.Context) error {
	schema := `CREATE TABLE IF NOT EXISTS chats (
        id TEXT PRIMARY KEY,
        title TEXT,
        encryption INTEGER,
        last_message_ts INTEGER
    );
    CREATE INDEX IF NOT EXISTS idx_chats_last_ts ON chats(last_message_ts DESC);`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// UpsertChat inserts or updates a chat record.
func (s *ChatStore) UpsertChat(ctx context.Context, c *domain.Chat) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO chats (
        id, title, encryption, last_message_ts
    ) VALUES (?,?,?,?)
    ON CONFLICT(id) DO UPDATE SET
        title=excluded.title,
        encryption=excluded.encryption,
        last_message_ts=excluded.last_message_ts`,
		c.ID, c.Title, c.Encryption, c.LastMessageTS)
	return err
}

// ListChats returns all chats ordered by most recent activity.
func (s *ChatStore) ListChats(ctx context.Context) ([]*domain.Chat, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, encryption, last_message_ts FROM chats ORDER BY last_message_ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chats []*domain.Chat
	for rows.Next() {
		var c domain.Chat
		if err := rows.Scan(&c.ID, &c.Title, &c.Encryption, &c.LastMessageTS); err != nil {
			return nil, err
		}
		chats = append(chats, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return chats, nil
}

// Close closes the underlying database.
func (s *ChatStore) Close() error { return s.db.Close() }
