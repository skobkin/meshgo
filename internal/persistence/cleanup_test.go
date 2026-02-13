package persistence

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestClearDatabase_ClearsAllTables(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().Unix()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO chats(chat_key, type, title, last_sent_by_me_at, updated_at)
		VALUES(?, ?, ?, ?, ?)
	`, domain.ChatKeyForChannel(0), int(domain.ChatTypeChannel), "General", now, now); err != nil {
		t.Fatalf("seed chats: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO messages(chat_key, direction, body, status, at)
		VALUES(?, ?, ?, ?, ?)
	`, domain.ChatKeyForChannel(0), int(domain.MessageDirectionIn), "hello", int(domain.MessageStatusSent), now); err != nil {
		t.Fatalf("seed messages: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO nodes(node_id, last_heard_at, updated_at)
		VALUES(?, ?, ?)
	`, "!00000001", now, now); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO traceroutes(request_id, target_node_id, started_at, updated_at, status)
		VALUES(?, ?, ?, ?, ?)
	`, "request-1", "!00000001", now, now, "in_progress"); err != nil {
		t.Fatalf("seed traceroutes: %v", err)
	}

	if err := ClearDatabase(ctx, db); err != nil {
		t.Fatalf("clear database: %v", err)
	}

	tableChecks := []struct {
		name  string
		query string
	}{
		{name: "messages", query: "SELECT COUNT(*) FROM messages;"},
		{name: "chats", query: "SELECT COUNT(*) FROM chats;"},
		{name: "nodes", query: "SELECT COUNT(*) FROM nodes;"},
		{name: "traceroutes", query: "SELECT COUNT(*) FROM traceroutes;"},
	}
	for _, table := range tableChecks {
		var count int
		if err := db.QueryRowContext(ctx, table.query).Scan(&count); err != nil {
			t.Fatalf("count rows in %s: %v", table.name, err)
		}
		if count != 0 {
			t.Fatalf("expected %s to be empty after clear, got %d rows", table.name, count)
		}
	}
}
