package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
}

// RuntimeCore contains app-level configuration, paths, logging, and startup integrations.
type RuntimeCore struct {
	Paths            Paths
	Config           config.AppConfig
	LogManager       *logging.Manager
	AutostartManager platform.AutostartManager
	UpdateChecker    *UpdateChecker
}

// RuntimePersistence contains database handles, repositories, and write projection queue.
type RuntimePersistence struct {
	DB             *sql.DB
	NodeRepo       *persistence.NodeRepo
	ChatRepo       *persistence.ChatRepo
	MessageRepo    *persistence.MessageRepo
	TracerouteRepo *persistence.TracerouteRepo
	WriterQueue    *persistence.WriterQueue
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
	Traceroute          *TracerouteService
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
	rt.Persistence.TracerouteRepo = persistence.NewTracerouteRepo(db)

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
		rt.Persistence.TracerouteRepo,
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
	rt.Connectivity.Traceroute = NewTracerouteService(
		b,
		rt.Connectivity.Radio,
		rt.Domain.NodeStore,
		rt.CurrentConnStatus,
		logMgr.Logger("traceroute"),
	)
	rt.Connectivity.Traceroute.Start(ctx)

	rt.Core.UpdateChecker = NewUpdateChecker(UpdateCheckerConfig{
		CurrentVersion: BuildVersion(),
		Logger:         logMgr.Logger("updates"),
	})
	rt.Core.UpdateChecker.Start(ctx)

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
	if r == nil || r.Core.UpdateChecker == nil {
		return UpdateSnapshot{}, false
	}

	return r.Core.UpdateChecker.CurrentSnapshot()
}

func (r *Runtime) UpdateSnapshots() <-chan UpdateSnapshot {
	if r == nil || r.Core.UpdateChecker == nil {
		return nil
	}

	return r.Core.UpdateChecker.Snapshots()
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

	//goland:noinspection SqlWithoutWhere
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

func (r *Runtime) ClearCache() error {
	cacheDir := r.Core.Paths.CacheDir
	if cacheDir == "" {
		return fmt.Errorf("cache dir is not configured")
	}
	if err := validateCacheClearTarget(cacheDir); err != nil {
		return err
	}

	mapTilesDir := r.Core.Paths.MapTilesDir
	if mapTilesDir != "" && !isPathWithinDir(cacheDir, mapTilesDir) {
		return fmt.Errorf("map tiles cache dir %q is outside cache dir %q", mapTilesDir, cacheDir)
	}
	slog.Info("cache clear requested", "cache_dir", cacheDir)

	if err := clearDirectoryContents(cacheDir); err != nil {
		return fmt.Errorf("clear app cache dir %q contents: %w", cacheDir, err)
	}
	if err := os.MkdirAll(cacheDir, 0o750); err != nil {
		return fmt.Errorf("recreate app cache dir %q: %w", cacheDir, err)
	}
	if mapTilesDir != "" {
		if err := os.MkdirAll(mapTilesDir, 0o750); err != nil {
			return fmt.Errorf("recreate map tiles cache dir %q: %w", mapTilesDir, err)
		}
	}

	slog.Info("cache cleared", "cache_dir", cacheDir)

	return nil
}

func validateCacheClearTarget(target string) error {
	if !filepath.IsAbs(target) {
		return fmt.Errorf("cache dir %q must be absolute", target)
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("resolve user cache dir: %w", err)
	}
	expected := filepath.Join(cacheRoot, Name)
	cleanTarget := filepath.Clean(target)
	cleanExpected := filepath.Clean(expected)
	if cleanTarget != cleanExpected {
		return fmt.Errorf("cache dir %q is outside app cache dir %q", target, cleanExpected)
	}

	info, err := os.Lstat(cleanTarget)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("cache dir %q must not be a symlink", cleanTarget)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect cache dir %q: %w", cleanTarget, err)
	}

	return nil
}

func clearDirectoryContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func isPathWithinDir(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
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
