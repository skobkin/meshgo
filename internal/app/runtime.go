package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/logging"
	"github.com/skobkin/meshgo/internal/persistence"
	"github.com/skobkin/meshgo/internal/platform"
	"github.com/skobkin/meshgo/internal/radio"
)

type Runtime struct {
	mu sync.RWMutex

	Ctx    context.Context
	cancel context.CancelFunc

	Paths  Paths
	Config config.AppConfig

	LogManager *logging.Manager
	Bus        *bus.PubSubBus
	DB         *sql.DB

	NodeRepo    *persistence.NodeRepo
	ChatRepo    *persistence.ChatRepo
	MessageRepo *persistence.MessageRepo
	WriterQueue *persistence.WriterQueue

	NodeStore *domain.NodeStore
	ChatStore *domain.ChatStore

	ConnectionTransport *SwitchableTransport
	Radio               *radio.Service
	AutostartManager    platform.AutostartManager

	connStatusMu    sync.RWMutex
	connStatus      connectors.ConnectionStatus
	connStatusKnown bool
}

func Initialize(parent context.Context) (*Runtime, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(paths.ConfigFile)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parent)
	rt := &Runtime{
		Ctx:    ctx,
		cancel: cancel,
		Paths:  paths,
		Config: cfg,
	}
	rt.AutostartManager = platform.NewAutostartManager()

	logMgr := logging.NewManager()
	if err := logMgr.Configure(cfg.Logging, paths.LogFile); err != nil {
		_ = logMgr.Close()
		cancel()
		return nil, fmt.Errorf("configure logging: %w", err)
	}
	rt.LogManager = logMgr
	slog.Info("starting meshgo runtime", "version", BuildVersion(), "build_date", BuildDateYMD())
	if err := rt.syncAutostart(cfg, "startup"); err != nil {
		slog.Warn("sync autostart on startup", "error", err)
	}

	db, err := persistence.Open(ctx, paths.DBFile)
	if err != nil {
		_ = rt.Close()
		return nil, err
	}
	rt.DB = db

	rt.NodeRepo = persistence.NewNodeRepo(db)
	rt.ChatRepo = persistence.NewChatRepo(db)
	rt.MessageRepo = persistence.NewMessageRepo(db)

	nodeStore := domain.NewNodeStore()
	chatStore := domain.NewChatStore()
	if err := domain.LoadStoresFromRepositories(ctx, nodeStore, chatStore, rt.NodeRepo, rt.ChatRepo, rt.MessageRepo); err != nil {
		_ = rt.Close()
		return nil, err
	}
	rt.NodeStore = nodeStore
	rt.ChatStore = chatStore

	b := bus.New(logMgr.Logger("bus"))
	rt.Bus = b
	connSub := b.Subscribe(connectors.TopicConnStatus)
	go rt.captureConnStatus(ctx, connSub)
	nodeStore.Start(ctx, b)
	chatStore.Start(ctx, b)

	writerQueue := persistence.NewWriterQueue(logMgr.Logger("persistence"), 512)
	writerQueue.Start(ctx)
	rt.WriterQueue = writerQueue
	domain.StartPersistenceProjection(ctx, b, writerQueue, rt.NodeRepo, rt.ChatRepo, rt.MessageRepo)

	codec, err := radio.NewMeshtasticCodec()
	if err != nil {
		_ = rt.Close()
		return nil, fmt.Errorf("initialize meshtastic codec: %w", err)
	}

	connTransport, err := NewConnectionTransport(cfg.Connection)
	if err != nil {
		_ = rt.Close()
		return nil, fmt.Errorf("initialize transport: %w", err)
	}
	rt.ConnectionTransport = connTransport

	rt.Radio = radio.NewService(logMgr.Logger("radio"), b, rt.ConnectionTransport, codec)
	rt.Radio.Start(ctx)

	return rt, nil
}

func (r *Runtime) captureConnStatus(ctx context.Context, sub bus.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-sub:
			if !ok {
				return
			}
			status, ok := raw.(connectors.ConnectionStatus)
			if !ok {
				continue
			}
			r.setConnStatus(status)
		}
	}
}

func (r *Runtime) setConnStatus(status connectors.ConnectionStatus) {
	r.connStatusMu.Lock()
	r.connStatus = status
	r.connStatusKnown = true
	r.connStatusMu.Unlock()
}

func (r *Runtime) CurrentConnStatus() (connectors.ConnectionStatus, bool) {
	r.connStatusMu.RLock()
	status := r.connStatus
	known := r.connStatusKnown
	r.connStatusMu.RUnlock()
	return status, known
}

func (r *Runtime) SaveAndApplyConfig(cfg config.AppConfig) error {
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	cfg.UI.LastSelectedChat = r.Config.UI.LastSelectedChat
	if err := config.Save(r.Paths.ConfigFile, cfg); err != nil {
		r.mu.Unlock()
		return err
	}
	r.Config = cfg
	r.mu.Unlock()

	if err := r.LogManager.Configure(cfg.Logging, r.Paths.LogFile); err != nil {
		return err
	}

	if r.ConnectionTransport != nil {
		if err := r.ConnectionTransport.Apply(cfg.Connection); err != nil {
			return err
		}
	}
	if err := r.syncAutostart(cfg, "settings_save"); err != nil {
		slog.Warn("sync autostart after save", "error", err)
		return &AutostartSyncWarning{Err: err}
	}

	return nil
}

func (r *Runtime) RememberSelectedChat(chatKey string) {
	normalized := strings.TrimSpace(chatKey)

	r.mu.Lock()
	if r.Config.UI.LastSelectedChat == normalized {
		r.mu.Unlock()
		return
	}
	cfg := r.Config
	cfg.UI.LastSelectedChat = normalized
	if err := config.Save(r.Paths.ConfigFile, cfg); err != nil {
		r.mu.Unlock()
		slog.Warn("save selected chat", "error", err)
		return
	}
	r.Config = cfg
	r.mu.Unlock()
}

func (r *Runtime) ClearDatabase() error {
	if r.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clear db tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []string{
		`DELETE FROM messages;`,
		`DELETE FROM chats;`,
		`DELETE FROM nodes;`,
	}
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("clear database tables: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clear db tx: %w", err)
	}

	if r.ChatStore != nil {
		r.ChatStore.Reset()
	}
	if r.NodeStore != nil {
		r.NodeStore.Reset()
	}
	slog.Info("database cleared")

	return nil
}

func (r *Runtime) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.Bus != nil {
		r.Bus.Close()
	}
	if r.ConnectionTransport != nil {
		_ = r.ConnectionTransport.Close()
	}
	if r.DB != nil {
		_ = r.DB.Close()
	}
	if r.LogManager != nil {
		_ = r.LogManager.Close()
	}
	return nil
}
