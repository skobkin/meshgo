package storage

import (
	"context"
	"database/sql"

	"meshgo/domain"
	_ "modernc.org/sqlite"
)

// ChannelStore persists channel metadata to SQLite.
type ChannelStore struct {
	db *sql.DB
}

// OpenChannelStore opens the SQLite database at the provided path.
func OpenChannelStore(path string) (*ChannelStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	return &ChannelStore{db: db}, nil
}

// Init ensures the schema exists.
func (s *ChannelStore) Init(ctx context.Context) error {
	schema := `CREATE TABLE IF NOT EXISTS channels (
        name TEXT PRIMARY KEY,
        psk_class INTEGER
    );`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// UpsertChannel inserts or updates a channel record.
func (s *ChannelStore) UpsertChannel(ctx context.Context, c *domain.Channel) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO channels (name, psk_class) VALUES (?,?)
        ON CONFLICT(name) DO UPDATE SET psk_class=excluded.psk_class`, c.Name, c.PSKClass)
	return err
}

// ListChannels returns all channels in the store.
func (s *ChannelStore) ListChannels(ctx context.Context) ([]*domain.Channel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, psk_class FROM channels`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var channels []*domain.Channel
	for rows.Next() {
		var c domain.Channel
		if err := rows.Scan(&c.Name, &c.PSKClass); err != nil {
			return nil, err
		}
		channels = append(channels, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return channels, nil
}

// RemoveChannel deletes the channel by name.
func (s *ChannelStore) RemoveChannel(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM channels WHERE name=?`, name)
	return err
}

// Close closes the database.
func (s *ChannelStore) Close() error { return s.db.Close() }
