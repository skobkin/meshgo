package fyne

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"meshgo/internal/core"
	"meshgo/internal/ui"
)

type FyneUI struct {
	logger    *slog.Logger
	app       fyne.App
	window    fyne.Window
	callbacks *ui.EventCallbacks
	
	// UI state
	chats     map[string]*ui.ChatViewModel
	nodes     map[string]*ui.NodeViewModel
	connectionState core.ConnectionState
	endpoint  string
	windowVisible bool
	selectedChatID string
	
	// Data bindings
	statusBinding    binding.String
	connectionButton *widget.Button
	connectTypeSelect *widget.Select
	connectEntry     *widget.Entry
	
	// Content areas
	nodesList   *widget.List
	chatsList   *widget.List
	messageArea *widget.RichText
	diagnosticsLabel *widget.Label
}

func NewFyneUI(logger *slog.Logger) *FyneUI {
	fyneApp := app.NewWithID("com.meshgo.app")
	
	// For now, disable icon to avoid corruption issues - system tray will handle it
	fyneApp.SetIcon(nil)
	
	window := fyneApp.NewWindow("MeshGo - Meshtastic GUI")
	window.Resize(fyne.NewSize(800, 600))
	window.CenterOnScreen()
	
	ui := &FyneUI{
		logger:        logger,
		app:           fyneApp,
		window:        window,
		chats:         make(map[string]*ui.ChatViewModel),
		nodes:         make(map[string]*ui.NodeViewModel),
		statusBinding: binding.NewString(),
		windowVisible: true, // Window starts visible
	}

	// Add keyboard shortcut to quit when tray doesn't work (Ctrl+Q)
	ctrlQ := &desktop.CustomShortcut{KeyName: fyne.KeyQ, Modifier: fyne.KeyModifierControl}
	window.Canvas().AddShortcut(ctrlQ, func(shortcut fyne.Shortcut) {
		// Use the exit callback to ensure proper shutdown sequence
		logger.Info("Ctrl+Q pressed - initiating shutdown")
		if ui.callbacks != nil && ui.callbacks.OnExit != nil {
			ui.callbacks.OnExit()
		} else {
			// Fallback: force quit the app
			logger.Info("No exit callback - force quitting")
			fyneApp.Quit()
		}
	})
	
	ui.setupUI()
	return ui
}

func (f *FyneUI) setupUI() {
	// Status bar
	statusLabel := widget.NewLabelWithData(f.statusBinding)
	f.statusBinding.Set("Status: Disconnected")
	
	// Create tabs with vertical rail
	tabs := container.NewAppTabs()
	tabs.SetTabLocation(container.TabLocationLeading) // Vertical tabs on left
	
	// Chats tab
	chatsTab := f.createChatsTab()
	tabs.Append(container.NewTabItem("Chats", chatsTab))
	
	// Nodes tab
	nodesTab := f.createNodesTab()
	tabs.Append(container.NewTabItem("Nodes", nodesTab))
	
	// Settings tab
	settingsTab := f.createSettingsTab()
	tabs.Append(container.NewTabItem("Settings", settingsTab))
	
	// Main layout with status bar at bottom
	mainContent := container.NewBorder(nil, statusLabel, nil, nil, tabs)
	f.window.SetContent(mainContent)
	
	// Handle window close - properly exit application
	f.window.SetCloseIntercept(func() {
		f.logger.Info("Window close requested - shutting down application")
		if f.callbacks != nil && f.callbacks.OnExit != nil {
			f.callbacks.OnExit()
		} else {
			f.app.Quit()
		}
	})
}

func (f *FyneUI) createChatsTab() fyne.CanvasObject {
	// Chats list on the left with compact items
	f.chatsList = widget.NewList(
		func() int { return len(f.chats) },
		func() fyne.CanvasObject {
			// Use a more compact layout - single row with title and icon
			return container.NewBorder(
				nil, nil, nil,
				widget.NewLabel("🔒"), // Encryption icon on right
				widget.NewLabel(""), // Chat title takes center space
			)
		},
		f.updateChatItem,
	)
	
	// Chat selection handler
	f.chatsList.OnSelected = func(id widget.ListItemID) {
		// Convert chats map to slice for indexing with consistent order
		chatIDs := make([]string, 0, len(f.chats))
		for chatID := range f.chats {
			chatIDs = append(chatIDs, chatID)
		}
		// Sort to ensure consistent ordering (matches updateChatItem)
		sort.Strings(chatIDs)
		
		if id < len(chatIDs) {
			chatID := chatIDs[id]
			if chat, exists := f.chats[chatID]; exists {
				f.selectedChatID = chat.ID
				f.logger.Info("Chat selected", "chatID", chat.ID, "title", chat.Title)
				f.loadAndDisplayMessages(chat.ID, chat.Title)
			}
		}
	}
	
	// Message area and input on the right
	f.messageArea = widget.NewRichText()
	f.messageArea.Wrapping = fyne.TextWrapWord
	
	messageEntry := widget.NewMultiLineEntry()
	messageEntry.SetPlaceHolder("Type your message here...")
	sendButton := widget.NewButton("Send", func() {
		if f.callbacks != nil && f.callbacks.OnSendMessage != nil && messageEntry.Text != "" && f.selectedChatID != "" {
			if err := f.callbacks.OnSendMessage(f.selectedChatID, messageEntry.Text); err != nil {
				f.logger.Error("Failed to send message", "error", err)
			} else {
				messageEntry.SetText("")
				// Refresh the message display after sending
				if chat, exists := f.chats[f.selectedChatID]; exists {
					f.loadAndDisplayMessages(f.selectedChatID, chat.Title)
				}
			}
		}
	})
	
	messageInput := container.NewBorder(nil, nil, nil, sendButton, messageEntry)
	messagePane := container.NewBorder(nil, messageInput, nil, nil, 
		container.NewScroll(f.messageArea))
	
	// Split layout: use Border container for better space allocation
	chatListSection := container.NewBorder(
		widget.NewLabel("Chats"), // Fixed header
		nil, nil, nil,
		f.chatsList, // List takes remaining space
	)
	
	chatContent := container.NewHSplit(chatListSection, messagePane)
	chatContent.SetOffset(0.3) // 30% chat list, 70% messages
	
	return chatContent
}

func (f *FyneUI) createNodesTab() fyne.CanvasObject {
	// Filter box at top
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Filter nodes...")
	
	// Nodes list
	f.nodesList = widget.NewList(
		func() int { return len(f.nodes) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				container.NewHBox(
					widget.NewLabel(""), // Short name
					widget.NewLabel(""), // Long name
					widget.NewButton("⭐", nil), // Favorite button
				),
				container.NewHBox(
					widget.NewLabel("📶"), // Signal quality icon placeholder
					widget.NewLabel(""), // Signal quality text
					widget.NewLabel(""), // Battery status
					widget.NewLabel("🔒"), // Encryption status placeholder
				),
			)
		},
		f.updateNodeItem,
	)
	
	// Double-click handler for opening DMs
	f.nodesList.OnSelected = func(id widget.ListItemID) {
		// Convert nodes map to slice for indexing with consistent order
		nodeIDs := make([]string, 0, len(f.nodes))
		for nodeID := range f.nodes {
			nodeIDs = append(nodeIDs, nodeID)
		}
		// Sort to ensure consistent ordering
		sort.Strings(nodeIDs)
		
		if id < len(nodeIDs) {
			nodeID := nodeIDs[id]
			if node, exists := f.nodes[nodeID]; exists {
				if f.callbacks != nil && f.callbacks.OnOpenDirectMessage != nil {
					f.callbacks.OnOpenDirectMessage(node.ID)
				}
			}
		}
	}
	
	// Use BorderContainer to give the list most of the space
	nodesContent := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Nodes"),
			filterEntry,
		), // top
		nil, // bottom
		nil, // left  
		nil, // right
		f.nodesList, // center (takes remaining space)
	)
	
	return nodesContent
}

func (f *FyneUI) createSettingsTab() fyne.CanvasObject {
	// Connection section
	f.connectTypeSelect = widget.NewSelect([]string{"serial", "ip"}, nil)
	f.connectTypeSelect.SetSelected("serial")
	
	f.connectEntry = widget.NewEntry()
	f.connectEntry.SetPlaceHolder("/dev/ttyUSB0 or 192.168.1.100:4403")
	
	f.connectionButton = widget.NewButton("Connect", f.handleConnectButton)
	
	connectionForm := container.NewVBox(
		container.NewHBox(widget.NewLabel("Type:"), f.connectTypeSelect),
		container.NewVBox(
			widget.NewLabel("Endpoint:"),
			f.connectEntry,
		),
		f.connectionButton,
	)
	connectionCard := widget.NewCard("Connection", "", connectionForm)
	
	// Notifications section
	notificationsEnabled := widget.NewCheck("Enable notifications", func(checked bool) {
		if f.callbacks != nil && f.callbacks.OnToggleNotifications != nil {
			f.callbacks.OnToggleNotifications(checked)
		}
	})
	notificationsEnabled.SetChecked(true)
	
	notificationsCard := widget.NewCard("Notifications", "", 
		container.NewVBox(notificationsEnabled))
	
	// Logging section
	loggingEnabled := widget.NewCheck("Enable logging to file", func(checked bool) {
		// TODO: Implement logging toggle
	})
	
	loggingCard := widget.NewCard("Logging", "", 
		container.NewVBox(loggingEnabled))
	
	// About section
	aboutContent := widget.NewRichText()
	aboutContent.ParseMarkdown("## MeshGo\n\nA cross-platform Meshtastic GUI application built in Go.\n\n**Version:** dev\n**License:** MIT")
	
	aboutCard := widget.NewCard("About", "", aboutContent)
	
	// Diagnostics section
	f.diagnosticsLabel = widget.NewLabel("Connection status: Disconnected\nLast error: None")
	f.updateDiagnostics()
	
	// Clear data buttons
	clearChatsButton := widget.NewButton("Clear chats", func() {
		if f.callbacks != nil && f.callbacks.OnClearChats != nil {
			if err := f.callbacks.OnClearChats(); err != nil {
				dialog.ShowError(err, f.window)
			} else {
				dialog.ShowInformation("Success", "All chats and messages have been cleared", f.window)
			}
		}
	})
	
	clearNodesButton := widget.NewButton("Clear nodes", func() {
		if f.callbacks != nil && f.callbacks.OnClearNodes != nil {
			if err := f.callbacks.OnClearNodes(); err != nil {
				dialog.ShowError(err, f.window)
			} else {
				dialog.ShowInformation("Success", "All nodes have been cleared", f.window)
			}
		}
	})
	
	diagnosticsContent := container.NewVBox(
		f.diagnosticsLabel,
		widget.NewSeparator(),
		widget.NewLabel("Data Management:"),
		container.NewHBox(clearChatsButton, clearNodesButton),
	)
	
	diagnosticsCard := widget.NewCard("Diagnostics", "", diagnosticsContent)
	
	settingsContent := container.NewScroll(
		container.NewVBox(
			connectionCard,
			notificationsCard,
			loggingCard,
			aboutCard,
			diagnosticsCard,
		),
	)
	
	return settingsContent
}

func (f *FyneUI) updateDiagnostics() {
	if f.diagnosticsLabel == nil {
		return
	}
	
	status := f.connectionState.String()
	if f.endpoint != "" && f.connectionState != core.StateDisconnected {
		status += " to " + f.endpoint
	}
	
	diagnosticsText := fmt.Sprintf("Connection status: %s\nLast error: None", status)
	f.diagnosticsLabel.SetText(diagnosticsText)
}

func (f *FyneUI) handleConnectButton() {
	if f.callbacks == nil {
		return
	}
	
	// Allow disconnect if we're connected, connecting, or retrying
	if f.connectionState == core.StateConnected || f.connectionState == core.StateConnecting || f.connectionState == core.StateRetrying {
		// Disconnect
		if f.callbacks.OnDisconnect != nil {
			f.callbacks.OnDisconnect()
		}
	} else {
		// Connect
		connType := f.connectTypeSelect.Selected
		endpoint := f.connectEntry.Text
		
		if endpoint == "" {
			dialog.ShowError(fmt.Errorf("Please enter connection endpoint"), f.window)
			return
		}
		
		if f.callbacks.OnConnect != nil {
			if err := f.callbacks.OnConnect(connType, endpoint); err != nil {
				dialog.ShowError(err, f.window)
			}
		}
	}
}

func (f *FyneUI) updateNodeItem(i widget.ListItemID, obj fyne.CanvasObject) {
	// Convert nodes map to slice for indexing with consistent order
	nodeIDs := make([]string, 0, len(f.nodes))
	for nodeID := range f.nodes {
		nodeIDs = append(nodeIDs, nodeID)
	}
	// Sort to ensure consistent ordering (matches OnSelected)
	sort.Strings(nodeIDs)
	
	if i >= len(nodeIDs) {
		return
	}
	
	nodeID := nodeIDs[i]
	node, exists := f.nodes[nodeID]
	if !exists {
		return
	}
	vbox := obj.(*fyne.Container)
	
	if len(vbox.Objects) >= 2 {
		// First row: short name, long name, favorite button
		nameRow := vbox.Objects[0].(*fyne.Container)
		if len(nameRow.Objects) >= 3 {
			nameRow.Objects[0].(*widget.Label).SetText(node.Node.ShortName)
			nameRow.Objects[1].(*widget.Label).SetText(node.Node.LongName)
			
			// Favorite button
			favoriteBtn := nameRow.Objects[2].(*widget.Button)
			if node.Node.Favorite {
				favoriteBtn.SetText("⭐")
			} else {
				favoriteBtn.SetText("☆")
			}
			favoriteBtn.OnTapped = func() {
				if f.callbacks != nil && f.callbacks.OnToggleNodeFavorite != nil {
					f.callbacks.OnToggleNodeFavorite(node.Node.ID)
				}
			}
		}
		
		// Second row: signal quality, battery, encryption status
		statusRow := vbox.Objects[1].(*fyne.Container)
		if len(statusRow.Objects) >= 4 {
			// Signal quality icon
			signalIcon := "📶"
			if node.IsOnline {
				switch node.SignalBars {
				case 3:
					signalIcon = "📶"
				case 2:
					signalIcon = "📶"
				case 1:
					signalIcon = "📶"
				default:
					signalIcon = "📵"
				}
			} else {
				signalIcon = "📵"
			}
			statusRow.Objects[0].(*widget.Label).SetText(signalIcon)
			
			// Signal quality text
			signalText := "Unknown"
			if node.IsOnline {
				signalText = node.StatusText
			}
			statusRow.Objects[1].(*widget.Label).SetText(signalText)
			
			// Battery status
			batteryText := ""
			if node.BatteryPercent > 0 {
				batteryText = fmt.Sprintf("%d%%", node.BatteryPercent)
			}
			statusRow.Objects[2].(*widget.Label).SetText(batteryText)
			
			// Encryption status
			encryptionIcon := "🔓" // Default: unencrypted
			if node.Node != nil {
				if node.Node.EncCustomKey {
					encryptionIcon = "🔑" // Custom key
				} else if node.Node.EncDefaultKey {
					encryptionIcon = "🔒" // Default key
				} else if node.Node.Unencrypted {
					encryptionIcon = "🔓" // Explicitly unencrypted
				}
			}
			statusRow.Objects[3].(*widget.Label).SetText(encryptionIcon)
		}
	}
}

func (f *FyneUI) updateChatItem(i widget.ListItemID, obj fyne.CanvasObject) {
	// Convert chats map to slice for indexing with consistent order
	chatIDs := make([]string, 0, len(f.chats))
	for chatID := range f.chats {
		chatIDs = append(chatIDs, chatID)
	}
	// Sort to ensure consistent ordering (matches OnSelected)
	sort.Strings(chatIDs)
	
	if i >= len(chatIDs) {
		return
	}
	
	chatID := chatIDs[i]
	chat, exists := f.chats[chatID]
	if !exists {
		return
	}
	borderContainer := obj.(*fyne.Container)
	
	// The border container has center (title) and right (encryption icon) objects
	if len(borderContainer.Objects) >= 2 {
		// Center: chat title with unread indicator
		titleText := chat.Title
		if chat.UnreadCount > 0 {
			titleText += fmt.Sprintf(" (%d)", chat.UnreadCount)
		}
		borderContainer.Objects[0].(*widget.Label).SetText(titleText)
		
		// Right: encryption icon
		encryptionIcon := "🔓" // Default: unencrypted
		if chat.Chat != nil {
			switch chat.Chat.Encryption {
			case 1: // Default key
				encryptionIcon = "🔒"
			case 2: // Custom key
				encryptionIcon = "🔑"
			default: // Unencrypted
				encryptionIcon = "🔓"
			}
		}
		borderContainer.Objects[1].(*widget.Label).SetText(encryptionIcon)
	}
}

// UI Adapter interface implementation

func (f *FyneUI) Run() error {
	f.logger.Info("Starting Fyne GUI")
	f.window.Show()
	f.app.Run() // This blocks until window is closed
	return nil
}

func (f *FyneUI) Shutdown() error {
	f.logger.Info("Shutting down Fyne GUI")
	// Use fyne.Do to ensure proper thread safety
	if f.app != nil {
		fyne.Do(func() {
			f.app.Quit()
		})
	}
	return nil
}

func (f *FyneUI) ShowMain() {
	if f.window != nil {
		f.logger.Debug("ShowMain called")
		fyne.Do(func() {
			f.window.Show()
			f.window.RequestFocus() // Ensure window gets focus and comes to front
			f.windowVisible = true
			f.logger.Debug("Window shown, focused, and visibility set to true")
			// Notify about visibility change
			if f.callbacks != nil && f.callbacks.OnWindowVisibilityChanged != nil {
				f.callbacks.OnWindowVisibilityChanged(true)
			}
		})
	}
}

func (f *FyneUI) HideMain() {
	if f.window != nil {
		f.logger.Debug("HideMain called")
		fyne.Do(func() {
			f.window.Hide()
			f.windowVisible = false
			f.logger.Debug("Window hidden and visibility set to false")
			// Notify about visibility change
			if f.callbacks != nil && f.callbacks.OnWindowVisibilityChanged != nil {
				f.callbacks.OnWindowVisibilityChanged(false)
			}
		})
	}
}

func (f *FyneUI) IsVisible() bool {
	return f.window != nil && f.windowVisible
}

func (f *FyneUI) SetTrayBadge(hasUnread bool) {
	// Update window title to show unread indicator
	title := "MeshGo - Meshtastic GUI"
	if hasUnread {
		title = "● " + title
	}
	if f.window != nil {
		fyne.Do(func() {
			f.window.SetTitle(title)
		})
	}
}

func (f *FyneUI) ShowTrayNotification(title, body string) error {
	if f.window != nil {
		// Use system notifications through Fyne
		fyne.Do(func() {
			f.app.SendNotification(&fyne.Notification{
				Title:   title,
				Content: body,
			})
		})
	}
	return nil
}

func (f *FyneUI) UpdateChats(chats []*core.Chat) {
	f.logger.Debug("Fyne UI: Updating chats", "count", len(chats))
	
	f.chats = make(map[string]*ui.ChatViewModel)
	for _, chat := range chats {
		f.logger.Debug("Fyne UI: Adding chat", "id", chat.ID, "title", chat.Title, "unread", chat.UnreadCount)
		vm := ui.ChatToViewModel(chat, nil, nil)
		f.chats[chat.ID] = vm
	}
	
	if f.chatsList != nil {
		f.logger.Debug("Fyne UI: Refreshing chat list")
		fyne.Do(func() {
			f.chatsList.Refresh()
		})
	} else {
		f.logger.Debug("Fyne UI: Chat list widget is nil - not refreshing")
	}
	
	// If a chat is currently selected, refresh its messages to show new ones
	if f.selectedChatID != "" {
		f.logger.Debug("Fyne UI: Refreshing currently selected chat", "chat_id", f.selectedChatID)
		if chat, exists := f.chats[f.selectedChatID]; exists {
			f.loadAndDisplayMessages(f.selectedChatID, chat.Title)
		}
	}
}

func (f *FyneUI) UpdateNodes(nodes []*core.Node) {
	f.nodes = make(map[string]*ui.NodeViewModel)
	for _, node := range nodes {
		vm := ui.NodeToViewModel(node)
		f.nodes[node.ID] = vm
	}
	
	if f.nodesList != nil {
		fyne.Do(func() {
			f.nodesList.Refresh()
		})
	}
}

func (f *FyneUI) UpdateConnectionStatus(state core.ConnectionState, endpoint string) {
	f.connectionState = state
	f.endpoint = endpoint
	
	status := fmt.Sprintf("Status: %s", state.String())
	if endpoint != "" {
		status += fmt.Sprintf(" -> %s", endpoint)
	}
	
	// All UI updates must happen on the Fyne thread
	fyne.Do(func() {
		f.statusBinding.Set(status)
		
		// Update connection button based on state
		if f.connectionButton != nil {
			switch state {
			case core.StateDisconnected:
				f.connectionButton.SetText("Connect")
			case core.StateConnecting:
				f.connectionButton.SetText("Cancel")
			case core.StateConnected:
				f.connectionButton.SetText("Disconnect")
			case core.StateRetrying:
				f.connectionButton.SetText("Stop Retrying")
			}
		}
		
		// Update diagnostics
		f.updateDiagnostics()
	})
}

func (f *FyneUI) UpdateSettings(settings *core.Settings) {
	f.logger.Debug("Settings updated in Fyne UI")
	
	// Update connection form with saved settings
	fyne.Do(func() {
		if f.connectTypeSelect != nil {
			f.connectTypeSelect.SetSelected(settings.Connection.Type)
		}
		
		if f.connectEntry != nil {
			var endpoint string
			switch settings.Connection.Type {
			case "serial":
				endpoint = settings.Connection.Serial.Port
			case "ip":
				endpoint = fmt.Sprintf("%s:%d", settings.Connection.IP.Host, settings.Connection.IP.Port)
			}
			f.connectEntry.SetText(endpoint)
		}
	})
}

func (f *FyneUI) ShowTraceroute(node *core.Node, hops []string) {
	content := fmt.Sprintf("Traceroute to %s (%s):\n", node.LongName, node.ID)
	for i, hop := range hops {
		content += fmt.Sprintf("  %d. %s\n", i+1, hop)
	}
	
	dialog.ShowInformation("Traceroute Result", content, f.window)
}

func (f *FyneUI) ShowError(title, message string) {
	dialog.ShowError(fmt.Errorf("%s", message), f.window)
}

func (f *FyneUI) ShowInfo(title, message string) {
	dialog.ShowInformation(title, message, f.window)
}

func (f *FyneUI) SetEventCallbacks(callbacks *ui.EventCallbacks) {
	f.callbacks = callbacks
}

func (f *FyneUI) loadAndDisplayMessages(chatID, chatTitle string) {
	if f.callbacks == nil || f.callbacks.OnLoadChatMessages == nil {
		f.logger.Warn("No message loading callback available")
		f.messageArea.ParseMarkdown(fmt.Sprintf("**%s**\n\nMessage loading not available", chatTitle))
		return
	}
	
	// Load messages from backend
	messages, err := f.callbacks.OnLoadChatMessages(chatID)
	if err != nil {
		f.logger.Error("Failed to load messages", "error", err, "chat_id", chatID)
		f.messageArea.ParseMarkdown(fmt.Sprintf("**%s**\n\nError loading messages: %v", chatTitle, err))
		return
	}
	
	// Display messages in markdown format
	var messageText strings.Builder
	messageText.WriteString(fmt.Sprintf("# %s\n\n", chatTitle))
	
	if len(messages) == 0 {
		messageText.WriteString("*No messages yet*\n")
	} else {
		// Reverse messages to show oldest first (GetMessages returns newest first)
		for i := len(messages) - 1; i >= 0; i-- {
			msg := messages[i]
			timestamp := msg.Timestamp.Format("15:04")
			
			// Get sender name from the message's sender ID
			senderName := msg.SenderID
			if f.callbacks != nil && f.callbacks.OnGetNodeName != nil {
				// Look up the actual node name
				nodeName := f.callbacks.OnGetNodeName(msg.SenderID)
				if nodeName != "" && nodeName != msg.SenderID {
					senderName = nodeName
				}
			}
			
			// If still using raw ID, truncate for display
			if senderName == msg.SenderID && len(senderName) > 8 {
				senderName = senderName[:8] // Truncate long IDs
			}
			
			if strings.HasPrefix(chatID, "channel_") {
				// For channel messages, show sender name
				messageText.WriteString(fmt.Sprintf("**%s** %s: %s\n\n", senderName, timestamp, msg.Text))
			} else {
				// For direct messages, just show time and message
				messageText.WriteString(fmt.Sprintf("**%s**: %s\n\n", timestamp, msg.Text))
			}
		}
	}
	
	f.messageArea.ParseMarkdown(messageText.String())
	
	f.logger.Debug("Displayed messages", "chat_id", chatID, "count", len(messages))
}

func (f *FyneUI) GetApp() fyne.App {
	return f.app
}