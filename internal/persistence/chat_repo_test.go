package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestChatRepoUpsert_PreservesNamedChannelTitleOnKeyFallbackUpdate(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "app.db")

	db, err := Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewChatRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	initial := domain.Chat{
		Key:       "channel:1",
		Type:      domain.ChatTypeChannel,
		Title:     "LongFast",
		UpdatedAt: now,
	}
	if err := repo.Upsert(ctx, initial); err != nil {
		t.Fatalf("upsert initial chat: %v", err)
	}

	overwriteAttempt := domain.Chat{
		Key:       "channel:1",
		Type:      domain.ChatTypeChannel,
		Title:     "channel:1",
		UpdatedAt: now.Add(5 * time.Second),
	}
	if err := repo.Upsert(ctx, overwriteAttempt); err != nil {
		t.Fatalf("upsert fallback-title chat: %v", err)
	}

	chats, err := repo.ListSortedByLastSentByMe(ctx)
	if err != nil {
		t.Fatalf("list chats: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected one chat, got %d", len(chats))
	}
	if chats[0].Title != "LongFast" {
		t.Fatalf("expected title to remain LongFast, got %q", chats[0].Title)
	}
}
