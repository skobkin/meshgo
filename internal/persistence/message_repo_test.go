package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestMessageRepoInsertAndLoad_RoundTripsReplyAndEmoji(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewMessageRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	_, err = repo.Insert(ctx, domain.ChatMessage{
		DeviceMessageID:        "100",
		ReplyToDeviceMessageID: "99",
		Emoji:                  1,
		ChatKey:                "channel:0",
		Direction:              domain.MessageDirectionIn,
		Body:                   "👍",
		Status:                 domain.MessageStatusSent,
		At:                     now,
	})
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}

	loaded, err := repo.ListRecentByChat(ctx, "channel:0", 10)
	if err != nil {
		t.Fatalf("load messages: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected one message, got %d", len(loaded))
	}
	if loaded[0].ReplyToDeviceMessageID != "99" {
		t.Fatalf("expected reply id to roundtrip, got %q", loaded[0].ReplyToDeviceMessageID)
	}
	if loaded[0].Emoji != 1 {
		t.Fatalf("expected emoji to roundtrip, got %d", loaded[0].Emoji)
	}
}
