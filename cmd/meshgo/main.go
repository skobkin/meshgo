package main

import (
	"context"
	"flag"
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
	"meshgo/internal/ui/fyne"
)

const Version = "1.0.0"

type SystemTrayInterface interface {
	SetUnread(bool)
	SetWindowVisible(bool)
	OnShowHide(func())
	OnToggleNotifications(func(bool))
	OnExit(func())
	Run()
	Quit()
}

type App struct {
	logger       *slog.Logger
	configMgr    *core.ConfigManager
	storage      *storage.SQLiteStore
	notifier     *system.Notifier
	tray         SystemTrayInterface
	ui           ui.Adapter
	radioClient  *protocol.RadioClient
	reconnectMgr *core.ReconnectManager
	
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	
	consoleMode  bool
}

func main() {
	var consoleMode = flag.Bool("console", false, "Run in console mode for debugging")
	flag.Parse()

	app, err := NewApp(*consoleMode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize app: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		app.logger.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func NewApp(consoleMode bool) (*App, error) {
	// Setup logger with debug level
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
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
	
	// Initialize UI - Fyne GUI by default, console only in debug mode
	var ui ui.Adapter
	if consoleMode {
		ui = console.NewConsoleUI(logger)
	} else {
		ui = fyne.NewFyneUI(logger)
	}
	
	// Initialize system tray with Fyne integration
	var tray SystemTrayInterface
	if fyneUI, ok := ui.(*fyne.FyneUI); ok && !consoleMode {
		tray = system.NewFyneSystemTray(logger, fyneUI.GetApp())
	} else {
		tray = system.NewSystemTray(logger)
	}
	
	// Initialize radio client
	radioClient := protocol.NewRadioClient(logger)

	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		logger:      logger,
		configMgr:   configMgr,
		storage:     store,
		notifier:    notifier,
		ui:          ui,
		radioClient: radioClient,
		ctx:         ctx,
		cancel:      cancel,
		consoleMode: consoleMode,
		tray:        tray,
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

	// Choose UI mode based on flags
	if app.consoleMode {
		app.logger.Info("Starting in console mode")
		// Run console UI in a goroutine so signals can interrupt
		app.wg.Add(1)
		go func() {
			defer app.wg.Done()
			if err := app.ui.Run(); err != nil {
				app.logger.Error("Console UI error", "error", err)
			}
		}()
		
		// Wait for shutdown signal
		<-app.ctx.Done()
	} else {
		app.logger.Info("Starting GUI mode with Fyne")
		// Start GUI (this blocks until window is closed)
		if err := app.ui.Run(); err != nil {
			return fmt.Errorf("GUI error: %w", err)
		}
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
	
	if app.tray != nil {
		app.tray.Quit()
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
		if msg, ok := event.Data.(*core.Message); ok && msg != nil {
			app.handleMessageReceived(msg)
		} else {
			app.logger.Debug("Received EventMessageReceived with invalid data", "data", event.Data)
		}
	case core.EventNodeUpdated:
		if node, ok := event.Data.(*core.Node); ok && node != nil {
			app.handleNodeUpdated(node)
		} else {
			app.logger.Debug("Received EventNodeUpdated with invalid data", "data", event.Data)
		}
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

	// Don't save nodes to database - keep them only in memory for live mesh data
	// Only save to DB if needed for chat identity mapping
	
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
	chats, err := app.storage.GetAllChats(app.ctx)
	if err != nil {
		app.logger.Error("Failed to load chats", "error", err)
		return
	}
	
	app.ui.UpdateChats(chats)
}

func (app *App) refreshNodes() {
	// Get nodes from radio client in-memory cache instead of database
	nodes := app.radioClient.GetNodes()
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
		OnWindowVisibilityChanged: func(visible bool) {
			app.tray.SetWindowVisible(visible)
		},
	}
	
	app.ui.SetEventCallbacks(callbacks)
}

func (app *App) setupTrayCallbacks() {
	app.tray.OnShowHide(func() {
		isVisible := app.ui.IsVisible()
		app.logger.Info("Tray Show/Hide clicked", "currentlyVisible", isVisible)
		if isVisible {
			app.logger.Info("Hiding window")
			app.ui.HideMain()
			app.tray.SetWindowVisible(false)
		} else {
			app.logger.Info("Showing window")
			app.ui.ShowMain()
			app.tray.SetWindowVisible(true)
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

	// Smart connection type detection - override user selection if endpoint format suggests different type
	if strings.Contains(endpoint, ":") && (strings.Contains(endpoint, ".") || strings.Contains(endpoint, "localhost")) {
		// Looks like host:port format, treat as IP connection
		connType = "ip"
		app.logger.Info("Auto-detected connection type as IP based on endpoint format", "endpoint", endpoint)
	}

	var t core.Transport

	switch strings.ToLower(connType) {
	case "serial":
		settings := app.configMgr.Settings()
		t = transport.NewSerialTransport(endpoint, settings.Connection.Serial.Baud)
		
		// Save serial endpoint to settings
		settings.Connection.Type = "serial"
		settings.Connection.Serial.Port = endpoint

	case "ip", "tcp":
		parts := strings.Split(endpoint, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid IP endpoint format, expected host:port")
		}
		
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid port number: %w", err)
		}
		
		t = transport.NewTCPTransport(parts[0], port)
		
		// Save IP endpoint to settings
		settings := app.configMgr.Settings()
		settings.Connection.Type = "ip"
		settings.Connection.IP.Host = parts[0]
		settings.Connection.IP.Port = port

	default:
		return fmt.Errorf("unsupported connection type: %s", connType)
	}
	
	// Save updated settings
	if err := app.configMgr.Save(); err != nil {
		app.logger.Warn("Failed to save connection settings", "error", err)
	}

	// Stop existing connection
	if app.reconnectMgr != nil {
		app.reconnectMgr.Stop()
	}

	// Start radio client with new transport
	if err := app.radioClient.Start(app.ctx, t); err != nil {
		return fmt.Errorf("failed to start radio client: %w", err)
	}

	// Setup reconnection manager
	settings := app.configMgr.Settings()
	app.reconnectMgr = core.NewReconnectManager(&settings.Reconnect, t, app.logger)
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
	// Force process termination to ensure complete shutdown
	os.Exit(0)
}