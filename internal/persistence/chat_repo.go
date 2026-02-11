package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/skobkin/meshgo/internal/domain"
)

// ChatRepo implements domain.ChatRepository using SQLite.
type ChatRepo struct {
	db *sql.DB
}

func NewChatRepo(db *sql.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) Upsert(ctx context.Context, c domain.Chat) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chats(chat_key, type, title, last_sent_by_me_at, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(chat_key) DO UPDATE SET
			type = excluded.type,
			title = CASE
				WHEN excluded.title = excluded.chat_key
					AND COALESCE(chats.title, '') <> ''
					AND chats.title <> chats.chat_key
				THEN chats.title
				ELSE excluded.title
			END,
			last_sent_by_me_at = CASE
				WHEN excluded.last_sent_by_me_at > COALESCE(chats.last_sent_by_me_at, 0)
				THEN excluded.last_sent_by_me_at
				ELSE chats.last_sent_by_me_at
			END,
			updated_at = CASE
				WHEN excluded.updated_at > chats.updated_at THEN excluded.updated_at
				ELSE chats.updated_at
			END
	`, c.Key, int(c.Type), c.Title, timeToUnixMillis(c.LastSentByMeAt), timeToUnixMillis(c.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert chat: %w", err)
	}

	return nil
}

func (r *ChatRepo) ListSortedByLastSentByMe(ctx context.Context) ([]domain.Chat, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT chat_key, type, title, last_sent_by_me_at, updated_at
		FROM chats
		ORDER BY last_sent_by_me_at DESC, updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := make([]domain.Chat, 0)
	for rows.Next() {
		var (
			chat       domain.Chat
			lastSentMs sql.NullInt64
			updatedMs  int64
			typeInt    int
		)
		if err := rows.Scan(&chat.Key, &typeInt, &chat.Title, &lastSentMs, &updatedMs); err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		chat.Type = domain.ChatType(typeInt)
		if lastSentMs.Valid {
			chat.LastSentByMeAt = unixMillisToTime(lastSentMs.Int64)
		}
		chat.UpdatedAt = unixMillisToTime(updatedMs)
		out = append(out, chat)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chats: %w", err)
	}

	return out, nil
}
