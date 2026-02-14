package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/logging"
	"github.com/skobkin/meshgo/internal/persistence"
)

func TestRuntimeSaveAndApplyConfig_TransportSwitchResetsInMemoryStores(t *testing.T) {
	rt := newRuntimeForSaveConfigTests(t)

	next := rt.Core.Config
	next.Connection.Connector = config.ConnectorSerial
	next.Connection.SerialPort = "/dev/ttyUSB0"
	next.Connection.SerialBaud = config.DefaultSerialBaud

	if err := rt.SaveAndApplyConfig(next); err != nil {
		t.Fatalf("save and apply config: %v", err)
	}

	if got := len(rt.Domain.ChatStore.ChatListSorted()); got != 0 {
		t.Fatalf("expected chat store to be reset after transport switch, got %d chats", got)
	}
	if got := len(rt.Domain.NodeStore.SnapshotSorted()); got != 0 {
		t.Fatalf("expected node store to be reset after transport switch, got %d nodes", got)
	}
}

func TestRuntimeSaveAndApplyConfig_SameTransportKeepsInMemoryStores(t *testing.T) {
	rt := newRuntimeForSaveConfigTests(t)

	next := rt.Core.Config
	next.Connection.Host = "192.168.1.20"

	if err := rt.SaveAndApplyConfig(next); err != nil {
		t.Fatalf("save and apply config: %v", err)
	}

	if got := len(rt.Domain.ChatStore.ChatListSorted()); got == 0 {
		t.Fatalf("expected chat store to keep data when transport is unchanged")
	}
	if got := len(rt.Domain.NodeStore.SnapshotSorted()); got == 0 {
		t.Fatalf("expected node store to keep data when transport is unchanged")
	}
}

func TestRuntimeSaveAndApplyConfig_UnchangedConnectionSkipsTransportReapply(t *testing.T) {
	rt := newRuntimeForSaveConfigTests(t)
	before := rt.Connectivity.ConnectionTransport.current()

	next := rt.Core.Config
	next.Logging.Level = "debug"

	if err := rt.SaveAndApplyConfig(next); err != nil {
		t.Fatalf("save and apply config: %v", err)
	}

	after := rt.Connectivity.ConnectionTransport.current()
	if before != after {
		t.Fatalf("expected transport instance to stay the same when connection config is unchanged")
	}
}

func TestRuntimeSaveAndApplyConfig_ChangedConnectionReappliesTransport(t *testing.T) {
	rt := newRuntimeForSaveConfigTests(t)
	before := rt.Connectivity.ConnectionTransport.current()

	next := rt.Core.Config
	next.Connection.Host = "192.168.1.20"

	if err := rt.SaveAndApplyConfig(next); err != nil {
		t.Fatalf("save and apply config: %v", err)
	}

	after := rt.Connectivity.ConnectionTransport.current()
	if before == after {
		t.Fatalf("expected transport instance to change when connection config changes")
	}
}

func TestRuntimeClearDatabase_ClearsAllTables(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Now()
	nowUnix := now.Unix()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO chats(chat_key, type, title, last_sent_by_me_at, updated_at)
		VALUES(?, ?, ?, ?, ?)
	`, domain.ChatKeyForChannel(0), int(domain.ChatTypeChannel), "General", nowUnix, nowUnix); err != nil {
		t.Fatalf("seed chats: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO messages(chat_key, direction, body, status, at)
		VALUES(?, ?, ?, ?, ?)
	`, domain.ChatKeyForChannel(0), int(domain.MessageDirectionIn), "hello", int(domain.MessageStatusSent), nowUnix); err != nil {
		t.Fatalf("seed messages: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO nodes(node_id, last_heard_at, updated_at)
		VALUES(?, ?, ?)
	`, "!00000001", nowUnix, nowUnix); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO traceroutes(request_id, target_node_id, started_at, updated_at, status)
		VALUES(?, ?, ?, ?, ?)
	`, "request-1", "!00000001", nowUnix, nowUnix, "in_progress"); err != nil {
		t.Fatalf("seed traceroutes: %v", err)
	}

	chatStore := domain.NewChatStore()
	chatStore.UpsertChat(domain.Chat{
		Key:       domain.ChatKeyForChannel(0),
		Type:      domain.ChatTypeChannel,
		Title:     "General",
		UpdatedAt: now,
	})
	chatStore.AppendMessage(domain.ChatMessage{
		ChatKey:   domain.ChatKeyForChannel(0),
		Direction: domain.MessageDirectionIn,
		Body:      "hello",
		Status:    domain.MessageStatusSent,
		At:        now,
	})

	nodeStore := domain.NewNodeStore()
	nodeStore.Upsert(domain.Node{
		NodeID:      "!00000001",
		ShortName:   "N1",
		LastHeardAt: now,
		UpdatedAt:   now,
	})

	rt := &Runtime{
		Persistence: RuntimePersistence{
			DB: db,
		},
		Domain: RuntimeDomain{
			ChatStore: chatStore,
			NodeStore: nodeStore,
		},
	}

	if err := rt.ClearDatabase(); err != nil {
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

	if got := len(chatStore.ChatListSorted()); got != 0 {
		t.Fatalf("expected chat store to be reset after clear, got %d chats", got)
	}
	if got := len(nodeStore.SnapshotSorted()); got != 0 {
		t.Fatalf("expected node store to be reset after clear, got %d nodes", got)
	}
}

func TestRuntimeCurrentConfigReturnsCopy(t *testing.T) {
	rt := newRuntimeForSaveConfigTests(t)

	cfg := rt.CurrentConfig()
	cfg.Connection.Host = "10.0.0.1"

	current := rt.CurrentConfig()
	if current.Connection.Host != rt.Core.Config.Connection.Host {
		t.Fatalf("expected runtime config to be immutable copy, got host %q", current.Connection.Host)
	}
}

func newRuntimeForSaveConfigTests(t *testing.T) *Runtime {
	t.Helper()

	initial := config.Default()
	initial.Connection.Connector = config.ConnectorIP
	initial.Connection.Host = "192.168.1.10"

	chatStore := domain.NewChatStore()
	nodeStore := domain.NewNodeStore()
	now := time.Now()
	chatStore.UpsertChat(domain.Chat{
		Key:       domain.ChatKeyForChannel(0),
		Title:     "General",
		Type:      domain.ChatTypeChannel,
		UpdatedAt: now,
	})
	chatStore.AppendMessage(domain.ChatMessage{
		ChatKey:      domain.ChatKeyForChannel(0),
		Direction:    domain.MessageDirectionIn,
		Body:         "hello",
		Status:       domain.MessageStatusSent,
		At:           now,
		MetaJSON:     "",
		StatusReason: "",
	})
	nodeStore.Upsert(domain.Node{
		NodeID:      "!00000001",
		ShortName:   "N1",
		LastHeardAt: now,
		UpdatedAt:   now,
	})

	connTr, err := NewConnectionTransport(initial.Connection)
	if err != nil {
		t.Fatalf("new connection transport: %v", err)
	}

	logMgr := logging.NewManager()
	t.Cleanup(func() {
		_ = logMgr.Close()
	})

	rt := &Runtime{
		Core: RuntimeCore{
			Paths: Paths{
				ConfigFile: filepath.Join(t.TempDir(), "config.json"),
				LogFile:    filepath.Join(t.TempDir(), "app.log"),
			},
			Config:     initial,
			LogManager: logMgr,
		},
		Domain: RuntimeDomain{
			ChatStore: chatStore,
			NodeStore: nodeStore,
		},
		Connectivity: RuntimeConnectivity{
			ConnectionTransport: connTr,
		},
	}

	return rt
}
