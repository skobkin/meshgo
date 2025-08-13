package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"meshgo/domain"
)

func TestMessageStore_InsertListUnread(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "msgs.db")
	ms, err := OpenMessageStore(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}

	m1 := &domain.Message{ChatID: "c1", SenderID: "s1", PortNum: 1, Text: "hi", Timestamp: time.Unix(1, 0), IsUnread: true}
	m2 := &domain.Message{ChatID: "c1", SenderID: "s2", PortNum: 1, Text: "hey", Timestamp: time.Unix(2, 0), IsUnread: true}
	if err := ms.InsertMessage(ctx, m1); err != nil {
		t.Fatalf("insert1: %v", err)
	}
	if err := ms.InsertMessage(ctx, m2); err != nil {
		t.Fatalf("insert2: %v", err)
	}

	msgs, err := ms.ListMessages(ctx, "c1", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Text != "hey" || msgs[1].Text != "hi" {
		t.Fatalf("unexpected msgs: %+v", msgs)
	}

	count, err := ms.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("unread: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 unread, got %d", count)
	}

	if err := ms.SetRead(ctx, "c1", m2.Timestamp); err != nil {
		t.Fatalf("setread: %v", err)
	}
	count, err = ms.UnreadCount(ctx)
	if err != nil {
		t.Fatalf("unread2: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 unread after SetRead, got %d", count)
	}
}

func TestMessageStoreUnreadCountByChat(t *testing.T) {
	ctx := context.Background()
	ms, err := OpenMessageStore(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	msgs := []*domain.Message{
		{ChatID: "c1", Text: "a", Timestamp: time.Unix(1, 0), IsUnread: true},
		{ChatID: "c1", Text: "b", Timestamp: time.Unix(2, 0), IsUnread: true},
		{ChatID: "c2", Text: "c", Timestamp: time.Unix(3, 0), IsUnread: false},
	}
	for i, m := range msgs {
		if err := ms.InsertMessage(ctx, m); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	counts, err := ms.UnreadCountByChat(ctx)
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts["c1"] != 2 || counts["c2"] != 0 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
	if err := ms.SetRead(ctx, "c1", time.Unix(2, 0)); err != nil {
		t.Fatalf("setread: %v", err)
	}
	counts, err = ms.UnreadCountByChat(ctx)
	if err != nil {
		t.Fatalf("counts2: %v", err)
	}
	if counts["c1"] != 0 {
		t.Fatalf("expected c1 count 0, got %d", counts["c1"])
	}
}
