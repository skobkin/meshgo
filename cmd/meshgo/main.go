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
	logLevel     *slog.LevelVar
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
	// Initialize config manager first to get log level
	configMgr, err := core.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}
	
	// Create programmable level for dynamic log level changes
	logLevel := &slog.LevelVar{}
	
	// Set initial log level from config
	levelStr := strings.ToLower(configMgr.Settings().Logging.Level)
	switch levelStr {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "info":
		logLevel.Set(slog.LevelInfo)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo) // Default fallback
	}
	
	// Setup logger with programmable level
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

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
		logLevel:    logLevel,
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
	
	// Auto-connect if enabled
	if settings.Connection.ConnectOnStartup {
		app.logger.Info("Auto-connecting on startup")
		go func() {
			// Small delay to ensure UI is ready
			time.Sleep(1 * time.Second)
			if err := app.handleConnect(settings.Connection.Type, app.getEndpointFromSettings(settings)); err != nil {
				app.logger.Warn("Auto-connect failed", "error", err)
			}
		}()
	}
}

func (app *App) getEndpointFromSettings(settings *core.Settings) string {
	switch settings.Connection.Type {
	case "serial":
		return settings.Connection.Serial.Port
	case "ip":
		return fmt.Sprintf("%s:%d", settings.Connection.IP.Host, settings.Connection.IP.Port)
	default:
		return ""
	}
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
	case core.EventChatUpdated:
		if chatData, ok := event.Data.(map[string]interface{}); ok {
			app.handleChatUpdated(chatData)
		} else {
			app.logger.Debug("Received EventChatUpdated with invalid data", "data", event.Data)
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
	app.logger.Debug("Message saved to database", "chat_id", msg.ChatID)
	
	// Update chat title with real channel name if it's a channel message
	app.updateChatTitle(msg.ChatID)

	// Update unread counts
	app.updateUnreadCounts()

	// Send notification
	if msg.IsUnread {
		var notificationTitle string
		var senderName string
		
		// Get sender name for the message content
		if node, _ := app.storage.GetNode(app.ctx, msg.SenderID); node != nil {
			senderName = node.LongName
			if senderName == "" {
				senderName = node.ShortName
			}
		}
		if senderName == "" {
			senderName = msg.SenderID // fallback to ID
		}
		
		// Generate notification title based on chat type
		if strings.HasPrefix(msg.ChatID, "channel_") {
			// For channel messages, get real channel name from RadioClient
			channelNum := strings.TrimPrefix(msg.ChatID, "channel_")
			if channelIndex, err := strconv.ParseInt(channelNum, 10, 32); err == nil {
				realChannelName := app.radioClient.GetChannelName(int32(channelIndex))
				if realChannelName != "" {
					notificationTitle = realChannelName
				} else {
					// Fallback to generic name if channel name not available yet
					notificationTitle = fmt.Sprintf("Channel %s", channelNum)
				}
			} else {
				notificationTitle = fmt.Sprintf("Channel %s", channelNum)
			}
		} else {
			// For direct messages, show sender name as title
			notificationTitle = senderName
		}
		
		// For channel messages, prepend sender name to message text
		notificationText := msg.Text
		if strings.HasPrefix(msg.ChatID, "channel_") {
			notificationText = fmt.Sprintf("%s: %s", senderName, msg.Text)
		}

		app.logger.Debug("Sending notification", "title", notificationTitle, "text", notificationText)
		if err := app.notifier.NotifyNewMessage(msg.ChatID, notificationTitle, notificationText, msg.Timestamp); err != nil {
			app.logger.Warn("Failed to send notification", "error", err)
		} else {
			app.logger.Debug("Notification sent successfully")
		}
	} else {
		app.logger.Debug("Message is not marked as unread - skipping notification")
	}

	// Refresh chats UI
	app.refreshChats()
}

func (app *App) handleNodeUpdated(node *core.Node) {
	app.logger.Debug("Node updated", "id", node.ID, "name", node.LongName)

	// Save nodes to database to support favorites and persistent node info
	if err := app.storage.SaveNode(app.ctx, node); err != nil {
		app.logger.Debug("Failed to save node to database", "error", err, "node_id", node.ID)
		// Don't fail the whole operation if database save fails
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
	app.logger.Debug("Updated unread counts", "total_unread", totalUnread, "has_unread", hasUnread)
	app.ui.SetTrayBadge(hasUnread)
}

func (app *App) refreshChats() {
	chats, err := app.storage.GetAllChats(app.ctx)
	if err != nil {
		app.logger.Error("Failed to load chats", "error", err)
		return
	}
	
	app.logger.Debug("Refreshing chats", "chat_count", len(chats))
	for _, chat := range chats {
		app.logger.Debug("Chat found", "id", chat.ID, "title", chat.Title, "unread", chat.UnreadCount)
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

func (app *App) updateChatTitle(chatID string) {
	if !strings.HasPrefix(chatID, "channel_") {
		return // Only update channel chat titles
	}
	
	channelNum := strings.TrimPrefix(chatID, "channel_")
	if channelIndex, err := strconv.ParseInt(channelNum, 10, 32); err == nil {
		realChannelName := app.radioClient.GetChannelName(int32(channelIndex))
		if realChannelName != "" {
			// Update the chat title in database
			if err := app.storage.UpdateChatTitle(app.ctx, chatID, realChannelName); err != nil {
				app.logger.Debug("Failed to update chat title", "error", err, "chat_id", chatID, "title", realChannelName)
			} else {
				app.logger.Debug("Updated chat title", "chat_id", chatID, "title", realChannelName)
			}
		}
	}
}

func (app *App) handleChatUpdated(chatData map[string]interface{}) {
	chatID, _ := chatData["chat_id"].(string)
	encryption, _ := chatData["encryption"].(int)
	title, _ := chatData["title"].(string)
	
	app.logger.Debug("Handling chat update", "chat_id", chatID, "encryption", encryption, "title", title)
	
	if chatID == "" {
		app.logger.Debug("Chat update missing chat_id")
		return
	}
	
	// Update chat encryption in database
	if err := app.storage.UpdateChatEncryption(app.ctx, chatID, encryption); err != nil {
		app.logger.Error("Failed to update chat encryption", "error", err, "chat_id", chatID, "encryption", encryption)
	} else {
		app.logger.Debug("Updated chat encryption", "chat_id", chatID, "encryption", encryption)
	}
	
	// Update chat title if provided
	if title != "" {
		if err := app.storage.UpdateChatTitle(app.ctx, chatID, title); err != nil {
			app.logger.Error("Failed to update chat title", "error", err, "chat_id", chatID, "title", title)
		} else {
			app.logger.Debug("Updated chat title", "chat_id", chatID, "title", title)
		}
	}
	
	// Refresh chats UI to show updated encryption indicators
	app.refreshChats()
}

func (app *App) setupUICallbacks() {
	callbacks := &ui.EventCallbacks{
		OnConnect:    app.handleConnect,
		OnDisconnect: app.handleDisconnect,
		OnSendMessage: app.handleSendMessage,
		OnLoadChatMessages: app.handleLoadChatMessages,
		OnGetNodeName: app.handleGetNodeName,
		OnToggleNodeFavorite: app.handleToggleNodeFavorite,
		OnTraceroute: app.handleTraceroute,
		OnUpdateNotifications: app.handleUpdateNotifications,
		OnUpdateConnectOnStartup: app.handleUpdateConnectOnStartup,
		OnUpdateLogLevel: app.handleUpdateLogLevel,
		OnExit: app.handleExit,
		OnWindowVisibilityChanged: func(visible bool) {
			app.tray.SetWindowVisible(visible)
		},
		OnClearChats: app.handleClearChats,
		OnClearNodes: app.handleClearNodes,
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

func (app *App) handleLoadChatMessages(chatID string) ([]*core.Message, error) {
	// Load messages from database with reasonable limit
	messages, err := app.storage.GetMessages(app.ctx, chatID, 50, 0)
	if err != nil {
		app.logger.Error("Failed to load chat messages", "error", err, "chat_id", chatID)
		return nil, fmt.Errorf("failed to load messages: %w", err)
	}
	
	app.logger.Debug("Loaded chat messages", "chat_id", chatID, "count", len(messages))
	
	// Mark messages as read when loading them
	if err := app.storage.MarkAsRead(app.ctx, chatID); err != nil {
		app.logger.Warn("Failed to mark messages as read", "error", err, "chat_id", chatID)
	}
	
	// Update unread counts after marking as read
	app.updateUnreadCounts()
	
	return messages, nil
}

func (app *App) handleGetNodeName(nodeID string) string {
	// First try to get from radio client (live nodes)
	nodes := app.radioClient.GetNodes()
	for _, node := range nodes {
		if node.ID == nodeID {
			if node.LongName != "" {
				return node.LongName
			}
			if node.ShortName != "" {
				return node.ShortName
			}
		}
	}
	
	// Fallback to database
	if node, err := app.storage.GetNode(app.ctx, nodeID); err == nil && node != nil {
		if node.LongName != "" {
			return node.LongName
		}
		if node.ShortName != "" {
			return node.ShortName
		}
	}
	
	// If it's our own node, check radio client's own node ID
	if ownID := fmt.Sprintf("%d", app.radioClient.GetOwnNodeID()); ownID == nodeID {
		return "You"
	}
	
	// Fallback to nodeID if no name found
	return nodeID
}

func (app *App) handleToggleNodeFavorite(nodeID string) error {
	// First try to get from live nodes
	nodes := app.radioClient.GetNodes()
	var targetNode *core.Node
	for _, node := range nodes {
		if node.ID == nodeID {
			targetNode = node
			break
		}
	}
	
	// Fallback to database if not in live nodes
	if targetNode == nil {
		var err error
		targetNode, err = app.storage.GetNode(app.ctx, nodeID)
		if err != nil {
			return fmt.Errorf("failed to get node: %w", err)
		}
		if targetNode == nil {
			return fmt.Errorf("node not found: %s", nodeID)
		}
	}

	newFavorite := !targetNode.Favorite
	
	// Update in database
	if err := app.storage.UpdateNodeFavorite(app.ctx, nodeID, newFavorite); err != nil {
		return fmt.Errorf("failed to update favorite in database: %w", err)
	}
	
	// Also update the in-memory version
	targetNode.Favorite = newFavorite

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

func (app *App) handleUpdateConnectOnStartup(enabled bool) error {
	app.logger.Debug("Connect on startup setting changed", "enabled", enabled)
	return app.configMgr.UpdateConnectOnStartup(enabled)
}

func (app *App) handleUpdateLogLevel(level string) error {
	app.logger.Info("Log level changed", "level", level)
	
	// Update the current logger level immediately
	levelStr := strings.ToLower(level)
	
	// Set the new log level
	switch levelStr {
	case "debug":
		app.logLevel.Set(slog.LevelDebug)
	case "info":
		app.logLevel.Set(slog.LevelInfo)
	case "warn":
		app.logLevel.Set(slog.LevelWarn)
	case "error":
		app.logLevel.Set(slog.LevelError)
	default:
		app.logLevel.Set(slog.LevelInfo) // Default fallback
	}
	
	app.logger.Debug("Log level updated", "level", levelStr)
	
	// Save to config
	return app.configMgr.UpdateLogging(app.configMgr.Settings().Logging.Enabled, levelStr)
}

func (app *App) handleClearChats() error {
	app.logger.Info("Clear chats requested")
	
	if err := app.storage.ClearAllChats(app.ctx); err != nil {
		app.logger.Error("Failed to clear chats", "error", err)
		return fmt.Errorf("failed to clear chats: %w", err)
	}
	
	app.logger.Info("All chats cleared successfully")
	
	// Refresh UI after clearing
	app.refreshChats()
	app.updateUnreadCounts()
	
	return nil
}

func (app *App) handleClearNodes() error {
	app.logger.Info("Clear nodes requested")
	
	if err := app.storage.ClearAllNodes(app.ctx); err != nil {
		app.logger.Error("Failed to clear nodes", "error", err)
		return fmt.Errorf("failed to clear nodes: %w", err)
	}
	
	app.logger.Info("All nodes cleared successfully")
	
	// Refresh UI after clearing
	app.refreshNodes()
	
	return nil
}

func (app *App) handleExit() {
	app.logger.Info("Exit requested")
	app.Shutdown()
	// Force process termination to ensure complete shutdown
	os.Exit(0)
}