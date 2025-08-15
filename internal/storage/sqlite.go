package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"meshgo/internal/core"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db   *sql.DB
	path string
}

func NewSQLiteStore(configDir string) (*SQLiteStore, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	dbPath := filepath.Join(configDir, "meshgo.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{
		db:   db,
		path: dbPath,
	}

	if err := store.migrate(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to migrate database: %w (also failed to close: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	// Read migration file
	migrationFile := "migrations/001_initial_schema.sql"
	migrationSQL, err := os.ReadFile(migrationFile)
	if err != nil {
		// If migration file doesn't exist, create schema inline
		return s.createInitialSchema()
	}

	_, err = s.db.Exec(string(migrationSQL))
	return err
}

func (s *SQLiteStore) createInitialSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    encryption INTEGER NOT NULL DEFAULT 0,
    last_message_ts INTEGER NOT NULL DEFAULT 0,
    unread_count INTEGER NOT NULL DEFAULT 0,
    is_channel BOOLEAN NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id TEXT NOT NULL,
    sender_id TEXT NOT NULL,
    portnum INTEGER NOT NULL,
    text TEXT NOT NULL,
    rx_snr REAL,
    rx_rssi INTEGER,
    timestamp INTEGER NOT NULL,
    is_unread BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (chat_id) REFERENCES chats(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_timestamp ON messages(chat_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(is_unread) WHERE is_unread = 1;

CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    short_name TEXT,
    long_name TEXT,
    favorite BOOLEAN NOT NULL DEFAULT 0,
    ignored BOOLEAN NOT NULL DEFAULT 0,
    unencrypted BOOLEAN NOT NULL DEFAULT 0,
    enc_default_key BOOLEAN NOT NULL DEFAULT 0,
    enc_custom_key BOOLEAN NOT NULL DEFAULT 0,
    rssi INTEGER DEFAULT 0,
    snr REAL DEFAULT 0.0,
    signal_quality INTEGER DEFAULT 0,
    battery_level INTEGER,
    is_charging BOOLEAN,
    last_heard INTEGER NOT NULL DEFAULT 0,
    position_data TEXT,
    device_metrics_data TEXT
);

CREATE INDEX IF NOT EXISTS idx_nodes_favorite ON nodes(favorite) WHERE favorite = 1;
CREATE INDEX IF NOT EXISTS idx_nodes_last_heard ON nodes(last_heard DESC);

CREATE TABLE IF NOT EXISTS channels (
    name TEXT PRIMARY KEY,
    psk_class INTEGER NOT NULL DEFAULT 0,
    psk_data BLOB,
    channel_id INTEGER
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT OR IGNORE INTO schema_version (version) VALUES (1);
`
	_, err := s.db.Exec(schema)
	return err
}

// MessageStore implementation

func (s *SQLiteStore) SaveMessage(ctx context.Context, msg *core.Message) error {
	query := `
INSERT INTO messages (chat_id, sender_id, portnum, text, rx_snr, rx_rssi, timestamp, is_unread)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	_, err := s.db.ExecContext(ctx, query,
		msg.ChatID, msg.SenderID, msg.PortNum, msg.Text,
		msg.RXSNR, msg.RXRSSI, msg.Timestamp.Unix(), msg.IsUnread)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Update chat's last message timestamp and unread count
	return s.updateChatAfterMessage(ctx, msg)
}

func (s *SQLiteStore) GetMessages(ctx context.Context, chatID string, limit int, offset int) ([]*core.Message, error) {
	query := `
SELECT id, chat_id, sender_id, portnum, text, rx_snr, rx_rssi, timestamp, is_unread
FROM messages
WHERE chat_id = ?
ORDER BY timestamp DESC
LIMIT ? OFFSET ?
`
	rows, err := s.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*core.Message
	for rows.Next() {
		msg := &core.Message{}
		var timestamp int64

		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderID, &msg.PortNum,
			&msg.Text, &msg.RXSNR, &msg.RXRSSI, &timestamp, &msg.IsUnread)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.Timestamp = time.Unix(timestamp, 0)
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (s *SQLiteStore) GetUnreadCount(ctx context.Context, chatID string) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE chat_id = ? AND is_unread = 1`

	var count int
	err := s.db.QueryRowContext(ctx, query, chatID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}

	return count, nil
}

func (s *SQLiteStore) MarkAsRead(ctx context.Context, chatID string) error {
	query := `UPDATE messages SET is_unread = 0 WHERE chat_id = ? AND is_unread = 1`

	_, err := s.db.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}

	// Update chat unread count
	updateQuery := `UPDATE chats SET unread_count = 0 WHERE id = ?`
	_, err = s.db.ExecContext(ctx, updateQuery, chatID)

	return err
}

func (s *SQLiteStore) GetTotalUnreadCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE is_unread = 1`

	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total unread count: %w", err)
	}

	return count, nil
}

func (s *SQLiteStore) updateChatAfterMessage(ctx context.Context, msg *core.Message) error {
	// Insert or update chat record - encryption will be updated by UpdateChatEncryption later
	query := `
INSERT INTO chats (id, title, encryption, last_message_ts, unread_count, is_channel)
VALUES (?, ?, 0, ?, 1, ?)
ON CONFLICT(id) DO UPDATE SET
    last_message_ts = excluded.last_message_ts,
    unread_count = unread_count + CASE WHEN excluded.unread_count > 0 THEN 1 ELSE 0 END
`
	isChannel := msg.ChatID != msg.SenderID // Simple heuristic

	// Generate human-readable chat title
	// Note: This will use a placeholder title since we don't have access to RadioClient here
	// The real channel name will be set when we have proper channel configuration
	var chatTitle string
	if strings.HasPrefix(msg.ChatID, "channel_") {
		// Extract channel number - will be updated with real name later
		channelNum := strings.TrimPrefix(msg.ChatID, "channel_")
		chatTitle = fmt.Sprintf("Channel %s", channelNum)
	} else {
		// For direct messages, use the chat ID as title for now
		// TODO: Could be improved by looking up node name
		chatTitle = msg.ChatID
	}

	_, err := s.db.ExecContext(ctx, query, msg.ChatID, chatTitle,
		msg.Timestamp.Unix(), isChannel)

	return err
}

// NodeStore implementation

func (s *SQLiteStore) SaveNode(ctx context.Context, node *core.Node) error {
	var positionData, metricsData []byte
	var err error

	if node.Position != nil {
		positionData, err = json.Marshal(node.Position)
		if err != nil {
			return fmt.Errorf("failed to marshal position data: %w", err)
		}
	}

	if node.DeviceMetrics != nil {
		metricsData, err = json.Marshal(node.DeviceMetrics)
		if err != nil {
			return fmt.Errorf("failed to marshal device metrics: %w", err)
		}
	}

	query := `
INSERT INTO nodes (id, short_name, long_name, favorite, ignored, unencrypted,
    enc_default_key, enc_custom_key, rssi, snr, signal_quality, battery_level,
    is_charging, last_heard, position_data, device_metrics_data)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    short_name = excluded.short_name,
    long_name = excluded.long_name,
    rssi = excluded.rssi,
    snr = excluded.snr,
    signal_quality = excluded.signal_quality,
    battery_level = excluded.battery_level,
    is_charging = excluded.is_charging,
    last_heard = excluded.last_heard,
    position_data = excluded.position_data,
    device_metrics_data = excluded.device_metrics_data
`
	_, err = s.db.ExecContext(ctx, query, node.ID, node.ShortName, node.LongName,
		node.Favorite, node.Ignored, node.Unencrypted, node.EncDefaultKey,
		node.EncCustomKey, node.RSSI, node.SNR, node.SignalQuality,
		node.BatteryLevel, node.IsCharging, node.LastHeard.Unix(),
		positionData, metricsData)

	if err != nil {
		return fmt.Errorf("failed to save node: %w", err)
	}

	return nil
}

func (s *SQLiteStore) GetNode(ctx context.Context, id string) (*core.Node, error) {
	query := `
SELECT id, short_name, long_name, favorite, ignored, unencrypted,
    enc_default_key, enc_custom_key, rssi, snr, signal_quality, battery_level,
    is_charging, last_heard, position_data, device_metrics_data
FROM nodes WHERE id = ?
`
	row := s.db.QueryRowContext(ctx, query, id)

	node := &core.Node{}
	var lastHeard int64
	var positionData, metricsData []byte

	err := row.Scan(&node.ID, &node.ShortName, &node.LongName, &node.Favorite,
		&node.Ignored, &node.Unencrypted, &node.EncDefaultKey, &node.EncCustomKey,
		&node.RSSI, &node.SNR, &node.SignalQuality, &node.BatteryLevel,
		&node.IsCharging, &lastHeard, &positionData, &metricsData)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	node.LastHeard = time.Unix(lastHeard, 0)

	if len(positionData) > 0 {
		if err := json.Unmarshal(positionData, &node.Position); err != nil {
			return nil, fmt.Errorf("failed to unmarshal position data: %w", err)
		}
	}

	if len(metricsData) > 0 {
		if err := json.Unmarshal(metricsData, &node.DeviceMetrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal device metrics: %w", err)
		}
	}

	return node, nil
}

func (s *SQLiteStore) GetAllNodes(ctx context.Context) ([]*core.Node, error) {
	query := `
SELECT id, short_name, long_name, favorite, ignored, unencrypted,
    enc_default_key, enc_custom_key, rssi, snr, signal_quality, battery_level,
    is_charging, last_heard, position_data, device_metrics_data
FROM nodes
ORDER BY last_heard DESC
`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*core.Node
	for rows.Next() {
		node := &core.Node{}
		var lastHeard int64
		var positionData, metricsData []byte

		err := rows.Scan(&node.ID, &node.ShortName, &node.LongName, &node.Favorite,
			&node.Ignored, &node.Unencrypted, &node.EncDefaultKey, &node.EncCustomKey,
			&node.RSSI, &node.SNR, &node.SignalQuality, &node.BatteryLevel,
			&node.IsCharging, &lastHeard, &positionData, &metricsData)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		node.LastHeard = time.Unix(lastHeard, 0)

		if len(positionData) > 0 {
			json.Unmarshal(positionData, &node.Position)
		}

		if len(metricsData) > 0 {
			json.Unmarshal(metricsData, &node.DeviceMetrics)
		}

		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (s *SQLiteStore) DeleteNode(ctx context.Context, id string) error {
	query := `DELETE FROM nodes WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *SQLiteStore) UpdateNodeFavorite(ctx context.Context, id string, favorite bool) error {
	query := `UPDATE nodes SET favorite = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, favorite, id)
	return err
}

func (s *SQLiteStore) UpdateNodeIgnored(ctx context.Context, id string, ignored bool) error {
	query := `UPDATE nodes SET ignored = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, ignored, id)
	return err
}

// SettingsStore implementation

func (s *SQLiteStore) Get(key string) (string, error) {
	query := `SELECT value FROM settings WHERE key = ?`
	var value string
	err := s.db.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *SQLiteStore) Set(key, value string) error {
	query := `INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := s.db.Exec(query, key, value)
	return err
}

func (s *SQLiteStore) GetBool(key string, defaultVal bool) bool {
	value, err := s.Get(key)
	if err != nil || value == "" {
		return defaultVal
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultVal
	}

	return result
}

func (s *SQLiteStore) SetBool(key string, value bool) error {
	return s.Set(key, strconv.FormatBool(value))
}

func (s *SQLiteStore) GetInt(key string, defaultVal int) int {
	value, err := s.Get(key)
	if err != nil || value == "" {
		return defaultVal
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}

	return result
}

func (s *SQLiteStore) SetInt(key string, value int) error {
	return s.Set(key, strconv.Itoa(value))
}

func (s *SQLiteStore) UpdateChatTitle(ctx context.Context, chatID, title string) error {
	query := `UPDATE chats SET title = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, title, chatID)
	if err != nil {
		return fmt.Errorf("failed to update chat title: %w", err)
	}
	return nil
}

func (s *SQLiteStore) UpdateChatEncryption(ctx context.Context, chatID string, encryption int) error {
	query := `UPDATE chats SET encryption = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, encryption, chatID)
	if err != nil {
		return fmt.Errorf("failed to update chat encryption: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ClearAllChats(ctx context.Context) error {
	// Clear messages first due to foreign key constraints
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages`)
	if err != nil {
		return fmt.Errorf("failed to clear messages: %w", err)
	}

	// Then clear chats
	_, err = s.db.ExecContext(ctx, `DELETE FROM chats`)
	if err != nil {
		return fmt.Errorf("failed to clear chats: %w", err)
	}

	return nil
}

func (s *SQLiteStore) ClearAllNodes(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM nodes`)
	if err != nil {
		return fmt.Errorf("failed to clear nodes: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetAllChats(ctx context.Context) ([]*core.Chat, error) {
	query := `
SELECT id, title, encryption, last_message_ts, unread_count, is_channel
FROM chats 
ORDER BY last_message_ts DESC
`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer rows.Close()

	var chats []*core.Chat
	for rows.Next() {
		chat := &core.Chat{}
		var lastMessageTs int64
		err := rows.Scan(&chat.ID, &chat.Title, &chat.Encryption,
			&lastMessageTs, &chat.UnreadCount, &chat.IsChannel)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat row: %w", err)
		}
		chat.LastMessageTS = time.Unix(lastMessageTs, 0)
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chat rows: %w", err)
	}

	return chats, nil
}
