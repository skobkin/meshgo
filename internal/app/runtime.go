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

// Runtime wires app services, persistence, transport, and UI-facing stores together.
type Runtime struct {
	mu sync.RWMutex

	Ctx    context.Context
	cancel context.CancelFunc

	Core         RuntimeCore
	Persistence  RuntimePersistence
	Domain       RuntimeDomain
	Connectivity RuntimeConnectivity

	connStatusMu    sync.RWMutex
	connStatus      connectors.ConnectionStatus
	connStatusKnown bool

	updateChecker *UpdateChecker
}

// RuntimeCore contains app-level configuration, paths, logging, and startup integrations.
type RuntimeCore struct {
	Paths            Paths
	Config           config.AppConfig
	LogManager       *logging.Manager
	AutostartManager platform.AutostartManager
}

// RuntimePersistence contains database handles, repositories, and write projection queue.
type RuntimePersistence struct {
	DB          *sql.DB
	NodeRepo    *persistence.NodeRepo
	ChatRepo    *persistence.ChatRepo
	MessageRepo *persistence.MessageRepo
	WriterQueue *persistence.WriterQueue
}

// RuntimeDomain contains in-memory stores and message bus projections used by the app/UI.
type RuntimeDomain struct {
	Bus       *bus.PubSubBus
	NodeStore *domain.NodeStore
	ChatStore *domain.ChatStore
}

// RuntimeConnectivity contains transport and radio services used for device communication.
type RuntimeConnectivity struct {
	ConnectionTransport *SwitchableTransport
	Radio               *radio.Service
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
		Core: RuntimeCore{
			Paths:            paths,
			Config:           cfg,
			AutostartManager: platform.NewAutostartManager(),
		},
	}

	logMgr := logging.NewManager()
	if err := logMgr.Configure(cfg.Logging, paths.LogFile); err != nil {
		_ = logMgr.Close()
		cancel()

		return nil, fmt.Errorf("configure logging: %w", err)
	}
	rt.Core.LogManager = logMgr
	slog.Info("starting meshgo runtime", "version", BuildVersion(), "build_date", BuildDateYMD())
	if err := rt.syncAutostart(cfg, "startup"); err != nil {
		slog.Warn("sync autostart on startup", "error", err)
	}

	db, err := persistence.Open(ctx, paths.DBFile)
	if err != nil {
		_ = rt.Close()

		return nil, err
	}
	rt.Persistence.DB = db

	rt.Persistence.NodeRepo = persistence.NewNodeRepo(db)
	rt.Persistence.ChatRepo = persistence.NewChatRepo(db)
	rt.Persistence.MessageRepo = persistence.NewMessageRepo(db)

	nodeStore := domain.NewNodeStore()
	chatStore := domain.NewChatStore()
	if err := domain.LoadStoresFromRepositories(
		ctx,
		nodeStore,
		chatStore,
		rt.Persistence.NodeRepo,
		rt.Persistence.ChatRepo,
		rt.Persistence.MessageRepo,
	); err != nil {
		_ = rt.Close()

		return nil, err
	}
	rt.Domain.NodeStore = nodeStore
	rt.Domain.ChatStore = chatStore

	b := bus.New(logMgr.Logger("bus"))
	rt.Domain.Bus = b
	connSub := b.Subscribe(connectors.TopicConnStatus)
	go rt.captureConnStatus(ctx, connSub)
	nodeStore.Start(ctx, b)
	chatStore.Start(ctx, b)

	writerQueue := persistence.NewWriterQueue(logMgr.Logger("persistence"), 512)
	writerQueue.Start(ctx)
	rt.Persistence.WriterQueue = writerQueue
	domain.StartPersistenceProjection(
		ctx,
		b,
		writerQueue,
		rt.Persistence.NodeRepo,
		rt.Persistence.ChatRepo,
		rt.Persistence.MessageRepo,
	)

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
	rt.Connectivity.ConnectionTransport = connTransport

	rt.Connectivity.Radio = radio.NewService(logMgr.Logger("radio"), b, rt.Connectivity.ConnectionTransport, codec)
	rt.Connectivity.Radio.Start(ctx)

	rt.updateChecker = NewUpdateChecker(UpdateCheckerConfig{
		CurrentVersion: BuildVersion(),
		Logger:         logMgr.Logger("updates"),
	})
	rt.updateChecker.Start(ctx)

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

func (r *Runtime) CurrentUpdateSnapshot() (UpdateSnapshot, bool) {
	if r == nil || r.updateChecker == nil {
		return UpdateSnapshot{}, false
	}

	return r.updateChecker.CurrentSnapshot()
}

func (r *Runtime) UpdateSnapshots() <-chan UpdateSnapshot {
	if r == nil || r.updateChecker == nil {
		return nil
	}

	return r.updateChecker.Snapshots()
}

func (r *Runtime) SaveAndApplyConfig(cfg config.AppConfig) error {
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	cfg.UI.LastSelectedChat = r.Core.Config.UI.LastSelectedChat
	cfg.UI.MapViewport = r.Core.Config.UI.MapViewport
	if err := config.Save(r.Core.Paths.ConfigFile, cfg); err != nil {
		r.mu.Unlock()

		return err
	}
	r.Core.Config = cfg
	r.mu.Unlock()

	if err := r.Core.LogManager.Configure(cfg.Logging, r.Core.Paths.LogFile); err != nil {
		return err
	}

	if r.Connectivity.ConnectionTransport != nil {
		if err := r.Connectivity.ConnectionTransport.Apply(cfg.Connection); err != nil {
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
	if r.Core.Config.UI.LastSelectedChat == normalized {
		r.mu.Unlock()

		return
	}
	cfg := r.Core.Config
	cfg.UI.LastSelectedChat = normalized
	if err := config.Save(r.Core.Paths.ConfigFile, cfg); err != nil {
		r.mu.Unlock()
		slog.Warn("save selected chat", "error", err)

		return
	}
	r.Core.Config = cfg
	r.mu.Unlock()
}

func (r *Runtime) RememberMapViewport(zoom, x, y int) {
	if zoom < 0 {
		zoom = 0
	}
	if zoom > 19 {
		zoom = 19
	}

	r.mu.Lock()
	current := r.Core.Config.UI.MapViewport
	if current.Set && current.Zoom == zoom && current.X == x && current.Y == y {
		r.mu.Unlock()

		return
	}

	cfg := r.Core.Config
	cfg.UI.MapViewport = config.MapViewportConfig{
		Set:  true,
		Zoom: zoom,
		X:    x,
		Y:    y,
	}
	if err := config.Save(r.Core.Paths.ConfigFile, cfg); err != nil {
		r.mu.Unlock()
		slog.Warn("save map viewport", "error", err, "zoom", zoom, "x", x, "y", y)

		return
	}
	r.Core.Config = cfg
	r.mu.Unlock()
}

func (r *Runtime) ClearDatabase() error {
	if r.Persistence.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := r.Persistence.DB.BeginTx(ctx, nil)
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

	if r.Domain.ChatStore != nil {
		r.Domain.ChatStore.Reset()
	}
	if r.Domain.NodeStore != nil {
		r.Domain.NodeStore.Reset()
	}
	slog.Info("database cleared")

	return nil
}

func (r *Runtime) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.Domain.Bus != nil {
		r.Domain.Bus.Close()
	}
	if r.Connectivity.ConnectionTransport != nil {
		_ = r.Connectivity.ConnectionTransport.Close()
	}
	if r.Persistence.DB != nil {
		_ = r.Persistence.DB.Close()
	}
	if r.Core.LogManager != nil {
		_ = r.Core.LogManager.Close()
	}

	return nil
}
