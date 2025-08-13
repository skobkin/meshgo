package storage

import (
	"context"
	"path/filepath"
	"testing"

	"meshgo/domain"
)

func TestChatStoreUpsertAndList(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "chats.db")
	cs, err := OpenChatStore(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}

	c1 := &domain.Chat{ID: "c1", Title: "Chat1", Encryption: 1, LastMessageTS: 2}
	c2 := &domain.Chat{ID: "c2", Title: "Chat2", Encryption: 0, LastMessageTS: 1}
	if err := cs.UpsertChat(ctx, c1); err != nil {
		t.Fatalf("upsert1: %v", err)
	}
	if err := cs.UpsertChat(ctx, c2); err != nil {
		t.Fatalf("upsert2: %v", err)
	}

	chats, err := cs.ListChats(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(chats) != 2 || chats[0].ID != "c1" || chats[1].ID != "c2" {
		t.Fatalf("unexpected chats order: %+v", chats)
	}

	c1.Title = "NewTitle"
	c1.LastMessageTS = 3
	if err := cs.UpsertChat(ctx, c1); err != nil {
		t.Fatalf("upsert3: %v", err)
	}
	chats, err = cs.ListChats(ctx)
	if err != nil {
		t.Fatalf("list2: %v", err)
	}
	if chats[0].Title != "NewTitle" || chats[0].LastMessageTS != 3 {
		t.Fatalf("chat not updated: %+v", chats[0])
	}
}
