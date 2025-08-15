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
	connected bool
	endpoint  string
	
	// Data bindings
	statusBinding    binding.String
	connectionButton *widget.Button
	connectTypeSelect *widget.Select
	connectEntry     *widget.Entry
	
	// Content areas
	nodesList   *widget.List
	chatsList   *widget.List
	messageArea *widget.RichText
}

func NewFyneUI(logger *slog.Logger) *FyneUI {
	fyneApp := app.NewWithID("com.meshgo.app")
	
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
	}
	
	ui.setupUI()
	return ui
}

func (f *FyneUI) setupUI() {
	// Connection section
	f.connectTypeSelect = widget.NewSelect([]string{"serial", "ip"}, nil)
	f.connectTypeSelect.SetSelected("serial")
	
	f.connectEntry = widget.NewEntry()
	f.connectEntry.SetPlaceHolder("/dev/ttyUSB0 or 192.168.1.100:4403")
	
	f.connectionButton = widget.NewButton("Connect", f.handleConnectButton)
	
	connectionForm := container.NewBorder(nil, nil, 
		widget.NewLabel("Type:"), f.connectionButton,
		container.NewHBox(f.connectTypeSelect, f.connectEntry))
	
	// Status bar
	statusLabel := widget.NewLabelWithData(f.statusBinding)
	f.statusBinding.Set("Status: Disconnected")
	
	// Nodes list
	f.nodesList = widget.NewList(
		func() int { return len(f.nodes) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(nil),
				widget.NewLabel(""),
				widget.NewLabel(""),
				widget.NewButton("⭐", nil),
			)
		},
		f.updateNodeItem,
	)
	
	// Chats list  
	f.chatsList = widget.NewList(
		func() int { return len(f.chats) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				widget.NewLabel(""),
			)
		},
		f.updateChatItem,
	)
	
	// Message area
	f.messageArea = widget.NewRichText()
	f.messageArea.Wrapping = fyne.TextWrapWord
	
	// Message input
	messageEntry := widget.NewMultiLineEntry()
	messageEntry.SetPlaceHolder("Type your message here...")
	sendButton := widget.NewButton("Send", func() {
		// TODO: Implement send message
		if f.callbacks != nil && f.callbacks.OnSendMessage != nil {
			// For now, send to broadcast channel
			f.callbacks.OnSendMessage("channel_0", messageEntry.Text)
			messageEntry.SetText("")
		}
	})
	
	messageInput := container.NewBorder(nil, nil, nil, sendButton, messageEntry)
	
	// Layout
	leftPanel := container.NewVBox(
		widget.NewCard("Connection", "", connectionForm),
		widget.NewCard("Nodes", "", f.nodesList),
	)
	
	rightPanel := container.NewVBox(
		widget.NewCard("Chats", "", f.chatsList),
		widget.NewCard("Messages", "", 
			container.NewBorder(nil, messageInput, nil, nil, 
				container.NewScroll(f.messageArea))),
	)
	
	content := container.NewHSplit(leftPanel, rightPanel)
	content.SetOffset(0.3) // 30% left, 70% right
	
	mainContent := container.NewBorder(nil, statusLabel, nil, nil, content)
	f.window.SetContent(mainContent)
	
	// Handle window close
	f.window.SetCloseIntercept(func() {
		if f.callbacks != nil && f.callbacks.OnExit != nil {
			f.callbacks.OnExit()
		} else {
			f.app.Quit()
		}
	})
}

func (f *FyneUI) handleConnectButton() {
	if f.callbacks == nil {
		return
	}
	
	if f.connected {
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
	hbox := obj.(*fyne.Container)
	
	// Update node display
	if len(hbox.Objects) >= 4 {
		// Signal quality icon (placeholder)
		// container.Objects[0].(*widget.Icon).SetResource(...)
		
		// Node name
		hbox.Objects[1].(*widget.Label).SetText(node.LongName)
		
		// Status
		status := "Offline"
		if node.IsOnline {
			status = fmt.Sprintf("Online (%s)", node.SignalQuality)
		}
		hbox.Objects[2].(*widget.Label).SetText(status)
		
		// Favorite button
		favoriteBtn := hbox.Objects[3].(*widget.Button)
		if node.Favorite {
			favoriteBtn.SetText("⭐")
		} else {
			favoriteBtn.SetText("☆")
		}
		favoriteBtn.OnTapped = func() {
			if f.callbacks != nil && f.callbacks.OnToggleNodeFavorite != nil {
				f.callbacks.OnToggleNodeFavorite(node.ID)
			}
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
	hbox := obj.(*fyne.Container)
	
	if len(hbox.Objects) >= 2 {
		// Chat title
		hbox.Objects[0].(*widget.Label).SetText(chat.Title)
		
		// Unread count
		unreadText := ""
		if chat.UnreadCount > 0 {
			unreadText = fmt.Sprintf("(%d unread)", chat.UnreadCount)
		}
		hbox.Objects[1].(*widget.Label).SetText(unreadText)
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
	if f.window != nil {
		f.window.Close()
	}
	return nil
}

func (f *FyneUI) ShowMain() {
	if f.window != nil {
		f.window.Show()
	}
}

func (f *FyneUI) HideMain() {
	if f.window != nil {
		f.window.Hide()
	}
}

func (f *FyneUI) IsVisible() bool {
	return f.window != nil && f.window.Content().Visible()
}

func (f *FyneUI) SetTrayBadge(hasUnread bool) {
	// Update window title to show unread indicator
	title := "MeshGo - Meshtastic GUI"
	if hasUnread {
		title = "● " + title
	}
	if f.window != nil {
		f.window.SetTitle(title)
	}
}

func (f *FyneUI) ShowTrayNotification(title, body string) error {
	if f.window != nil {
		// Use system notifications through Fyne
		f.app.SendNotification(&fyne.Notification{
			Title:   title,
			Content: body,
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
		f.chatsList.Refresh()
	}
}

func (f *FyneUI) UpdateNodes(nodes []*core.Node) {
	f.nodes = make(map[string]*ui.NodeViewModel)
	for _, node := range nodes {
		vm := ui.NodeToViewModel(node)
		f.nodes[node.ID] = vm
	}
	
	if f.nodesList != nil {
		f.nodesList.Refresh()
	}
}

func (f *FyneUI) UpdateConnectionStatus(state core.ConnectionState, endpoint string) {
	f.connected = state == core.StateConnected
	f.endpoint = endpoint
	
	status := fmt.Sprintf("Status: %s", state.String())
	if endpoint != "" {
		status += fmt.Sprintf(" -> %s", endpoint)
	}
	
	f.statusBinding.Set(status)
	
	// Update connection button
	if f.connectionButton != nil {
		if f.connected {
			f.connectionButton.SetText("Disconnect")
		} else {
			f.connectionButton.SetText("Connect")
		}
	}
}

func (f *FyneUI) UpdateSettings(settings *core.Settings) {
	f.logger.Debug("Settings updated in Fyne UI")
	// Update UI based on settings if needed
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