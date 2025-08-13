package storage

import (
	"context"
	"path/filepath"
	"testing"

	"meshgo/domain"
)

func TestChannelStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "db.sqlite")
	s, err := OpenChannelStore(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	if err := s.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	ch := &domain.Channel{Name: "foo", PSKClass: 2}
	if err := s.UpsertChannel(context.Background(), ch); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	ch.PSKClass = 1
	if err := s.UpsertChannel(context.Background(), ch); err != nil {
		t.Fatalf("update: %v", err)
	}
	channels, err := s.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(channels) != 1 || channels[0].PSKClass != 1 {
		t.Fatalf("unexpected channels: %#v", channels)
	}
	if err := s.RemoveChannel(context.Background(), "foo"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	channels, err = s.ListChannels(context.Background())
	if err != nil {
		t.Fatalf("list2: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("expected empty, got %#v", channels)
	}
}
