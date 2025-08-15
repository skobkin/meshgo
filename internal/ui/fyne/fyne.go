package fyne

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
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
	
	// Disable icon to avoid loading errors (no icon assets available)
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
	
	// Handle window close - minimize to tray instead of exit
	f.window.SetCloseIntercept(func() {
		f.logger.Info("Window close requested - minimizing to tray")
		f.window.Hide()
		f.windowVisible = false
		// Notify tray about visibility change via callback
		if f.callbacks != nil && f.callbacks.OnWindowVisibilityChanged != nil {
			f.callbacks.OnWindowVisibilityChanged(false)
		}
	})
}

func (f *FyneUI) createChatsTab() fyne.CanvasObject {
	// Chats list on the left
	f.chatsList = widget.NewList(
		func() int { return len(f.chats) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				container.NewHBox(
					widget.NewLabel(""), // Chat title
					widget.NewLabel("🔒"), // Encryption icon placeholder
				),
				widget.NewLabel(""), // Last message snippet
			)
		},
		f.updateChatItem,
	)
	
	// Message area and input on the right
	f.messageArea = widget.NewRichText()
	f.messageArea.Wrapping = fyne.TextWrapWord
	
	messageEntry := widget.NewMultiLineEntry()
	messageEntry.SetPlaceHolder("Type your message here...")
	sendButton := widget.NewButton("Send", func() {
		if f.callbacks != nil && f.callbacks.OnSendMessage != nil && messageEntry.Text != "" {
			// For now, send to broadcast channel
			f.callbacks.OnSendMessage("channel_0", messageEntry.Text)
			messageEntry.SetText("")
		}
	})
	
	messageInput := container.NewBorder(nil, nil, nil, sendButton, messageEntry)
	messagePane := container.NewBorder(nil, messageInput, nil, nil, 
		container.NewScroll(f.messageArea))
	
	// Split layout: chat list on left, message pane on right
	chatContent := container.NewHSplit(
		container.NewVBox(
			widget.NewLabel("Chats"),
			f.chatsList,
		),
		messagePane,
	)
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
		// Convert nodes map to slice for indexing
		nodes := make([]*ui.NodeViewModel, 0, len(f.nodes))
		for _, node := range f.nodes {
			nodes = append(nodes, node)
		}
		
		if id < len(nodes) {
			node := nodes[id]
			if f.callbacks != nil && f.callbacks.OnOpenDirectMessage != nil {
				f.callbacks.OnOpenDirectMessage(node.ID)
			}
		}
	}
	
	nodesContent := container.NewVBox(
		widget.NewLabel("Nodes"),
		filterEntry,
		f.nodesList,
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
	diagnosticsCard := widget.NewCard("Diagnostics", "", f.diagnosticsLabel)
	
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
	// Convert nodes map to slice for indexing
	nodes := make([]*ui.NodeViewModel, 0, len(f.nodes))
	for _, node := range f.nodes {
		nodes = append(nodes, node)
	}
	
	if i >= len(nodes) {
		return
	}
	
	node := nodes[i]
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
	// Convert chats map to slice for indexing
	chats := make([]*ui.ChatViewModel, 0, len(f.chats))
	for _, chat := range f.chats {
		chats = append(chats, chat)
	}
	
	if i >= len(chats) {
		return
	}
	
	chat := chats[i]
	vbox := obj.(*fyne.Container)
	
	if len(vbox.Objects) >= 2 {
		// First row: title and encryption icon
		titleRow := vbox.Objects[0].(*fyne.Container)
		if len(titleRow.Objects) >= 2 {
			titleRow.Objects[0].(*widget.Label).SetText(chat.Title)
			// Set encryption icon based on chat encryption status
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
			titleRow.Objects[1].(*widget.Label).SetText(encryptionIcon)
		}
		
		// Second row: last message snippet
		lastMessageText := "No messages"
		if chat.LastMessage != nil && chat.LastMessage.Text != "" {
			lastMessageText = chat.LastMessage.Text
			if len(lastMessageText) > 50 {
				lastMessageText = lastMessageText[:47] + "..."
			}
		}
		if chat.UnreadCount > 0 {
			lastMessageText += fmt.Sprintf(" (%d unread)", chat.UnreadCount)
		}
		vbox.Objects[1].(*widget.Label).SetText(lastMessageText)
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
	f.chats = make(map[string]*ui.ChatViewModel)
	for _, chat := range chats {
		vm := ui.ChatToViewModel(chat, nil, nil)
		f.chats[chat.ID] = vm
	}
	
	if f.chatsList != nil {
		fyne.Do(func() {
			f.chatsList.Refresh()
		})
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

func (f *FyneUI) GetApp() fyne.App {
	return f.app
}