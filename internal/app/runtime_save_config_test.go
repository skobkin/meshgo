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
	next.Connection.Transport = config.TransportSerial
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

func TestRuntimeDeleteDMChat_RemovesPersistenceAndState(t *testing.T) {
	ctx := context.Background()
	db, err := persistence.Open(ctx, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	now := time.Now()
	nowUnixMillis := now.UnixMilli()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO chats(chat_key, type, title, last_sent_by_me_at, updated_at)
		VALUES(?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`,
		domain.ChatKeyForDM("!12345678"), int(domain.ChatTypeDM), "Alice", nowUnixMillis, nowUnixMillis,
		domain.ChatKeyForChannel(0), int(domain.ChatTypeChannel), "General", nowUnixMillis, nowUnixMillis,
	); err != nil {
		t.Fatalf("seed chats: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO messages(chat_key, device_message_id, reply_to_device_message_id, emoji, direction, body, status, at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		domain.ChatKeyForDM("!12345678"), "100", "99", 1, int(domain.MessageDirectionIn), "hello", int(domain.MessageStatusSent), nowUnixMillis,
		domain.ChatKeyForChannel(0), "101", nil, 0, int(domain.MessageDirectionIn), "world", int(domain.MessageStatusSent), nowUnixMillis,
	); err != nil {
		t.Fatalf("seed messages: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Default()
	cfg.Connection.Transport = config.TransportIP
	cfg.Connection.Host = "192.168.1.10"
	cfg.UI.LastSelectedChat = domain.ChatKeyForDM("!12345678")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	chatStore := domain.NewChatStore()
	chatStore.Load(
		[]domain.Chat{
			{Key: domain.ChatKeyForDM("!12345678"), Title: "Alice", Type: domain.ChatTypeDM, UpdatedAt: now},
			{Key: domain.ChatKeyForChannel(0), Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: now},
		},
		map[string][]domain.ChatMessage{
			domain.ChatKeyForDM("!12345678"): {
				{ChatKey: domain.ChatKeyForDM("!12345678"), DeviceMessageID: "100", Body: "hello", Direction: domain.MessageDirectionIn, Status: domain.MessageStatusSent, At: now},
			},
			domain.ChatKeyForChannel(0): {
				{ChatKey: domain.ChatKeyForChannel(0), DeviceMessageID: "101", Body: "world", Direction: domain.MessageDirectionIn, Status: domain.MessageStatusSent, At: now},
			},
		},
	)

	rt := &Runtime{
		Core: RuntimeCore{
			Config: cfg,
			Paths: Paths{
				ConfigFile: configPath,
			},
		},
		Persistence: RuntimePersistence{
			DB: db,
		},
		Domain: RuntimeDomain{
			ChatStore: chatStore,
		},
	}

	if err := rt.DeleteDMChat(domain.ChatKeyForDM("!12345678")); err != nil {
		t.Fatalf("delete dm chat: %v", err)
	}

	var chatCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM chats WHERE chat_key = ?`, domain.ChatKeyForDM("!12345678")).Scan(&chatCount); err != nil {
		t.Fatalf("count dm chats: %v", err)
	}
	if chatCount != 0 {
		t.Fatalf("expected dm chat row to be removed, got %d", chatCount)
	}
	var messageCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE chat_key = ?`, domain.ChatKeyForDM("!12345678")).Scan(&messageCount); err != nil {
		t.Fatalf("count dm messages: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("expected dm messages to be removed, got %d", messageCount)
	}
	if _, ok := chatStore.ChatByKey(domain.ChatKeyForDM("!12345678")); ok {
		t.Fatalf("expected dm chat to be removed from store")
	}
	if got := len(chatStore.Messages(domain.ChatKeyForDM("!12345678"))); got != 0 {
		t.Fatalf("expected dm chat messages to be removed from store, got %d", got)
	}
	if rt.Core.Config.UI.LastSelectedChat != "" {
		t.Fatalf("expected last selected chat to be cleared, got %q", rt.Core.Config.UI.LastSelectedChat)
	}
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if loadedCfg.UI.LastSelectedChat != "" {
		t.Fatalf("expected persisted last selected chat to be cleared, got %q", loadedCfg.UI.LastSelectedChat)
	}
}

func TestRuntimeDeleteDMChat_RejectsNonDMChat(t *testing.T) {
	rt := &Runtime{}

	err := rt.DeleteDMChat(domain.ChatKeyForChannel(0))
	if err == nil {
		t.Fatalf("expected non-dm delete to fail")
	}
}

func newRuntimeForSaveConfigTests(t *testing.T) *Runtime {
	t.Helper()

	initial := config.Default()
	initial.Connection.Transport = config.TransportIP
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
