package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"meshgo/internal/core"
	"meshgo/internal/protocol"
	"meshgo/internal/storage"
	"meshgo/internal/system"
	"meshgo/internal/transport"
	"meshgo/internal/ui"
	"meshgo/internal/ui/console"
)

const Version = "1.0.0"

type App struct {
	logger       *slog.Logger
	configMgr    *core.ConfigManager
	storage      *storage.SQLiteStore
	notifier     *system.Notifier
	tray         *system.SystemTray
	ui           ui.Adapter
	radioClient  *protocol.RadioClient
	reconnectMgr *core.ReconnectManager
	
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		app.logger.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func NewApp() (*App, error) {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Initialize config manager
	configMgr, err := core.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	// Initialize storage
	store, err := storage.NewSQLiteStore(configMgr.ConfigDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Initialize system components
	notifier := system.NewNotifier(logger)
	tray := system.NewSystemTray(logger)
	
	// Initialize UI (console for now)
	ui := console.NewConsoleUI(logger)
	
	// Initialize radio client
	radioClient := protocol.NewRadioClient(logger)

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		logger:      logger,
		configMgr:   configMgr,
		storage:     store,
		notifier:    notifier,
		tray:        tray,
		ui:          ui,
		radioClient: radioClient,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Set up UI callbacks
	app.setupUICallbacks()

	// Set up tray callbacks
	app.setupTrayCallbacks()

	return app, nil
}

func (app *App) Run() error {
	app.logger.Info("Starting MeshGo", "version", Version)

	// Start background services
	app.startServices()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.logger.Info("Received signal", "signal", sig)
		app.Shutdown()
	}()

	// Start UI
	if err := app.ui.Run(); err != nil {
		return fmt.Errorf("UI error: %w", err)
	}

	// Wait for shutdown
	app.wg.Wait()
	app.logger.Info("MeshGo stopped")

	return nil
}

func (app *App) Shutdown() {
	app.logger.Info("Shutting down...")
	
	app.cancel()

	if app.reconnectMgr != nil {
		app.reconnectMgr.Stop()
	}

	if err := app.radioClient.Stop(); err != nil {
		app.logger.Error("Failed to stop radio client", "error", err)
	}

	if err := app.storage.Close(); err != nil {
		app.logger.Error("Failed to close storage", "error", err)
	}

	if err := app.ui.Shutdown(); err != nil {
		app.logger.Error("Failed to shutdown UI", "error", err)
	}
}

func (app *App) startServices() {
	// Start event processing
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		app.processEvents()
	}()

	// Update UI with initial settings
	settings := app.configMgr.Settings()
	app.ui.UpdateSettings(settings)
	app.notifier.SetEnabled(settings.Notifications.Enabled)

	// Load initial data
	app.loadInitialData()
}

func (app *App) processEvents() {
	app.logger.Info("Starting event processor")

	for {
		select {
		case <-app.ctx.Done():
			app.logger.Info("Event processor stopped")
			return

		case event := <-app.radioClient.Events():
			app.handleRadioEvent(event)

		case event := <-app.getReconnectEvents():
			app.handleReconnectEvent(event)
		}
	}
}

func (app *App) getReconnectEvents() <-chan core.Event {
	if app.reconnectMgr != nil {
		return app.reconnectMgr.Events()
	}
	// Return a channel that never sends to avoid blocking
	ch := make(chan core.Event)
	close(ch)
	return ch
}

func (app *App) handleRadioEvent(event core.Event) {
	switch event.Type {
	case core.EventMessageReceived:
		app.handleMessageReceived(event.Data.(*core.Message))
	case core.EventNodeUpdated:
		app.handleNodeUpdated(event.Data.(*core.Node))
	}
}

func (app *App) handleReconnectEvent(event core.Event) {
	switch event.Type {
	case core.EventConnectionStateChanged:
		data := event.Data.(core.ConnectionStateData)
		app.ui.UpdateConnectionStatus(data.State, data.Endpoint)
	}
}

func (app *App) handleMessageReceived(msg *core.Message) {
	app.logger.Info("Message received", 
		"from", msg.SenderID, 
		"chat", msg.ChatID, 
		"text", msg.Text)

	// Save to database
	if err := app.storage.SaveMessage(app.ctx, msg); err != nil {
		app.logger.Error("Failed to save message", "error", err)
		return
	}

	// Update unread counts
	app.updateUnreadCounts()

	// Send notification
	if msg.IsUnread {
		chatTitle := msg.ChatID
		if node, _ := app.storage.GetNode(app.ctx, msg.SenderID); node != nil {
			chatTitle = node.LongName
			if chatTitle == "" {
				chatTitle = node.ShortName
			}
		}

		if err := app.notifier.NotifyNewMessage(msg.ChatID, chatTitle, msg.Text, msg.Timestamp); err != nil {
			app.logger.Warn("Failed to send notification", "error", err)
		}
	}

	// Refresh chats UI
	app.refreshChats()
}

func (app *App) handleNodeUpdated(node *core.Node) {
	app.logger.Debug("Node updated", "id", node.ID, "name", node.LongName)

	// Save to database
	if err := app.storage.SaveNode(app.ctx, node); err != nil {
		app.logger.Error("Failed to save node", "error", err)
		return
	}

	// Refresh nodes UI
	app.refreshNodes()
}

func (app *App) updateUnreadCounts() {
	totalUnread, err := app.storage.GetTotalUnreadCount(app.ctx)
	if err != nil {
		app.logger.Error("Failed to get unread count", "error", err)
		return
	}

	hasUnread := totalUnread > 0
	app.ui.SetTrayBadge(hasUnread)
}

func (app *App) refreshChats() {
	// For now, just update with empty list
	// In a real implementation, would load from storage
	app.ui.UpdateChats([]*core.Chat{})
}

func (app *App) refreshNodes() {
	nodes, err := app.storage.GetAllNodes(app.ctx)
	if err != nil {
		app.logger.Error("Failed to load nodes", "error", err)
		return
	}

	app.ui.UpdateNodes(nodes)
}

func (app *App) loadInitialData() {
	app.refreshChats()
	app.refreshNodes()
	app.updateUnreadCounts()
}

func (app *App) setupUICallbacks() {
	callbacks := &ui.EventCallbacks{
		OnConnect:    app.handleConnect,
		OnDisconnect: app.handleDisconnect,
		OnSendMessage: app.handleSendMessage,
		OnToggleNodeFavorite: app.handleToggleNodeFavorite,
		OnTraceroute: app.handleTraceroute,
		OnUpdateNotifications: app.handleUpdateNotifications,
		OnExit: app.handleExit,
	}
	
	app.ui.SetEventCallbacks(callbacks)
}

func (app *App) setupTrayCallbacks() {
	app.tray.OnShowHide(func() {
		if app.ui.IsVisible() {
			app.ui.HideMain()
		} else {
			app.ui.ShowMain()
		}
	})

	app.tray.OnToggleNotifications(func(enabled bool) {
		app.notifier.SetEnabled(enabled)
		app.configMgr.UpdateNotifications(enabled)
	})

	app.tray.OnExit(app.handleExit)
}

func (app *App) handleConnect(connType, endpoint string) error {
	app.logger.Info("Connection requested", "type", connType, "endpoint", endpoint)

	var transport core.Transport

	switch strings.ToLower(connType) {
	case "serial":
		settings := app.configMgr.Settings()
		transport = transport.NewSerialTransport(endpoint, settings.Connection.Serial.Baud)

	case "ip", "tcp":
		parts := strings.Split(endpoint, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid IP endpoint format, expected host:port")
		}
		
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid port number: %w", err)
		}
		
		transport = transport.NewTCPTransport(parts[0], port)

	default:
		return fmt.Errorf("unsupported connection type: %s", connType)
	}

	// Stop existing connection
	if app.reconnectMgr != nil {
		app.reconnectMgr.Stop()
	}

	// Start radio client with new transport
	if err := app.radioClient.Start(app.ctx, transport); err != nil {
		return fmt.Errorf("failed to start radio client: %w", err)
	}

	// Setup reconnection manager
	settings := app.configMgr.Settings()
	app.reconnectMgr = core.NewReconnectManager(&settings.Reconnect, transport, app.logger)
	app.reconnectMgr.Start(app.ctx)

	return nil
}

func (app *App) handleDisconnect() error {
	app.logger.Info("Disconnect requested")

	if app.reconnectMgr != nil {
		app.reconnectMgr.Stop()
		app.reconnectMgr = nil
	}

	return app.radioClient.Stop()
}

func (app *App) handleSendMessage(chatID, text string) error {
	if !app.reconnectMgr.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Parse node ID from chat ID - simplified logic
	nodeID := uint32(0)
	if strings.HasPrefix(chatID, "channel_") {
		nodeID = 0xFFFFFFFF // Broadcast
	} else {
		id, err := strconv.ParseUint(chatID, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid chat ID: %w", err)
		}
		nodeID = uint32(id)
	}

	sendCtx, cancel := context.WithTimeout(app.ctx, 10*time.Second)
	defer cancel()

	return app.radioClient.SendText(sendCtx, chatID, nodeID, text)
}

func (app *App) handleToggleNodeFavorite(nodeID string) error {
	node, err := app.storage.GetNode(app.ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	newFavorite := !node.Favorite
	if err := app.storage.UpdateNodeFavorite(app.ctx, nodeID, newFavorite); err != nil {
		return fmt.Errorf("failed to update favorite: %w", err)
	}

	app.logger.Info("Node favorite toggled", "node", nodeID, "favorite", newFavorite)
	app.refreshNodes()

	return nil
}

func (app *App) handleTraceroute(nodeID string) error {
	if !app.reconnectMgr.IsConnected() {
		return fmt.Errorf("not connected")
	}

	id, err := strconv.ParseUint(nodeID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid node ID: %w", err)
	}

	traceCtx, cancel := context.WithTimeout(app.ctx, 30*time.Second)
	defer cancel()

	return app.radioClient.SendTraceroute(traceCtx, uint32(id))
}

func (app *App) handleUpdateNotifications(enabled bool) error {
	app.notifier.SetEnabled(enabled)
	return app.configMgr.UpdateNotifications(enabled)
}

func (app *App) handleExit() {
	app.logger.Info("Exit requested")
	app.Shutdown()
}