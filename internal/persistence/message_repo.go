package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Insert(ctx context.Context, m domain.ChatMessage) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO messages(chat_key, device_message_id, direction, body, status, at, meta_json)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, m.ChatKey, nullableString(m.DeviceMessageID), int(m.Direction), m.Body, int(m.Status), timeToUnixMillis(m.At), nullableString(m.MetaJSON))
	if err != nil {
		return 0, fmt.Errorf("insert message: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return 0, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get message local id: %w", err)
	}
	return id, nil
}

func (r *MessageRepo) ListRecentByChat(ctx context.Context, chatKey string, limit int) ([]domain.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT local_id, chat_key, device_message_id, direction, body, status, at, meta_json
		FROM messages
		WHERE chat_key = ?
		ORDER BY at DESC
		LIMIT ?
	`, chatKey, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages by chat: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var out []domain.ChatMessage
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages by chat: %w", err)
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *MessageRepo) LoadRecentPerChat(ctx context.Context, limit int) (map[string][]domain.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT chat_key FROM chats`)
	if err != nil {
		return nil, fmt.Errorf("list chat keys: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make(map[string][]domain.ChatMessage)
	for rows.Next() {
		var chatKey string
		if err := rows.Scan(&chatKey); err != nil {
			return nil, fmt.Errorf("scan chat key: %w", err)
		}
		msgs, err := r.ListRecentByChat(ctx, chatKey, limit)
		if err != nil {
			return nil, err
		}
		if len(msgs) > 0 {
			result[chatKey] = msgs
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat keys: %w", err)
	}
	return result, nil
}

func (r *MessageRepo) UpdateStatusByDeviceMessageID(ctx context.Context, deviceMessageID string, status domain.MessageStatus) error {
	deviceMessageID = strings.TrimSpace(deviceMessageID)
	if deviceMessageID == "" || status == 0 {
		return nil
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT local_id, status
		FROM messages
		WHERE device_message_id = ?
	`, deviceMessageID)
	if err != nil {
		return fmt.Errorf("query messages by device id: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	type rowState struct {
		id     int64
		status domain.MessageStatus
	}
	toUpdate := make([]rowState, 0, 1)
	for rows.Next() {
		var (
			id        int64
			statusRaw int
		)
		if err := rows.Scan(&id, &statusRaw); err != nil {
			return fmt.Errorf("scan message status row: %w", err)
		}
		current := domain.MessageStatus(statusRaw)
		if !domain.ShouldTransitionMessageStatus(current, status) {
			continue
		}
		toUpdate = append(toUpdate, rowState{id: id, status: status})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate message status rows: %w", err)
	}
	if len(toUpdate) == 0 {
		return nil
	}

	for _, item := range toUpdate {
		if _, err := r.db.ExecContext(ctx, `
			UPDATE messages
			SET status = ?
			WHERE local_id = ?
		`, int(item.status), item.id); err != nil {
			return fmt.Errorf("update message status: %w", err)
		}
	}
	return nil
}

func scanMessage(scanner interface {
	Scan(dest ...any) error
}) (domain.ChatMessage, error) {
	var (
		m           domain.ChatMessage
		atMs        int64
		direction   int
		status      int
		deviceIDRaw sql.NullString
		metaRaw     sql.NullString
	)
	if err := scanner.Scan(&m.LocalID, &m.ChatKey, &deviceIDRaw, &direction, &m.Body, &status, &atMs, &metaRaw); err != nil {
		return domain.ChatMessage{}, fmt.Errorf("scan message: %w", err)
	}
	m.Direction = domain.MessageDirection(direction)
	m.Status = domain.MessageStatus(status)
	m.At = unixMillisToTime(atMs)
	if deviceIDRaw.Valid {
		m.DeviceMessageID = deviceIDRaw.String
	}
	if metaRaw.Valid {
		m.MetaJSON = metaRaw.String
	}
	return m, nil
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
