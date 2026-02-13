package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/logging"
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
