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

func TestMessageRepoDeleteByChat_RemovesOnlyTargetMessages(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewMessageRepo(db)
	now := time.Now().UTC().Truncate(time.Second)
	insertMessage := func(chatKey, deviceID string) {
		t.Helper()
		if _, err := repo.Insert(ctx, domain.ChatMessage{
			DeviceMessageID:        deviceID,
			ReplyToDeviceMessageID: "99",
			Emoji:                  1,
			ChatKey:                chatKey,
			Direction:              domain.MessageDirectionIn,
			Body:                   "hello",
			Status:                 domain.MessageStatusSent,
			At:                     now,
		}); err != nil {
			t.Fatalf("insert message for %s: %v", chatKey, err)
		}
	}

	insertMessage("dm:!1234abcd", "100")
	insertMessage("channel:0", "101")

	if err := repo.DeleteByChat(ctx, "dm:!1234abcd"); err != nil {
		t.Fatalf("delete messages by chat: %v", err)
	}

	dmMessages, err := repo.ListRecentByChat(ctx, "dm:!1234abcd", 10)
	if err != nil {
		t.Fatalf("list dm messages: %v", err)
	}
	if len(dmMessages) != 0 {
		t.Fatalf("expected dm messages to be deleted, got %d", len(dmMessages))
	}

	channelMessages, err := repo.ListRecentByChat(ctx, "channel:0", 10)
	if err != nil {
		t.Fatalf("list channel messages: %v", err)
	}
	if len(channelMessages) != 1 {
		t.Fatalf("expected non-target messages to remain, got %d", len(channelMessages))
	}
}
