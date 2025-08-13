package storage

import (
	"context"
	"database/sql"
	"time"

	"meshgo/domain"

	_ "modernc.org/sqlite"
)

// MessageStore persists chat messages to SQLite.
type MessageStore struct {
	db *sql.DB
}

// OpenMessageStore opens the SQLite database at the provided path.
func OpenMessageStore(path string) (*MessageStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &MessageStore{db: db}, nil
}

// Init ensures the schema exists.
func (s *MessageStore) Init(ctx context.Context) error {
	schema := `CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY,
        chat_id TEXT,
        sender_id TEXT,
        portnum INTEGER,
        text TEXT,
        rx_snr REAL,
        rx_rssi INTEGER,
        timestamp INTEGER,
        is_unread BOOLEAN
    );
    CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages(chat_id, timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(is_unread);`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// InsertMessage stores a message.
func (s *MessageStore) InsertMessage(ctx context.Context, m *domain.Message) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (chat_id, sender_id, portnum, text, rx_snr, rx_rssi, timestamp, is_unread) VALUES (?,?,?,?,?,?,?,?)`,
		m.ChatID, m.SenderID, m.PortNum, m.Text, m.RxSNR, m.RxRSSI, m.Timestamp.Unix(), m.IsUnread)
	return err
}

// ListMessages returns up to `limit` most recent messages for a chat.
func (s *MessageStore) ListMessages(ctx context.Context, chatID string, limit int) ([]*domain.Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, chat_id, sender_id, portnum, text, rx_snr, rx_rssi, timestamp, is_unread FROM messages WHERE chat_id=? ORDER BY timestamp DESC LIMIT ?`, chatID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*domain.Message
	for rows.Next() {
		var m domain.Message
		var ts int64
		if err := rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.PortNum, &m.Text, &m.RxSNR, &m.RxRSSI, &ts, &m.IsUnread); err != nil {
			return nil, err
		}
		m.Timestamp = time.Unix(ts, 0)
		msgs = append(msgs, &m)
	}
	return msgs, rows.Err()
}

// UnreadCount returns the number of unread messages.
func (s *MessageStore) UnreadCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE is_unread=1`).Scan(&count)
	return count, err
}

// Close closes the underlying database.
func (s *MessageStore) Close() error { return s.db.Close() }

// Vacuum periodically compacts the database.
func (s *MessageStore) Vacuum(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `VACUUM`)
	return err
}

// SetRead marks messages in a chat as read up to the provided time.
func (s *MessageStore) SetRead(ctx context.Context, chatID string, upTo time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE messages SET is_unread=0 WHERE chat_id=? AND timestamp<=?`, chatID, upTo.Unix())
	return err
}
