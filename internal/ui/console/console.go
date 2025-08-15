package console

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"meshgo/internal/core"
	"meshgo/internal/ui"
)

type ConsoleUI struct {
	logger    *slog.Logger
	callbacks *ui.EventCallbacks
	chats     map[string]*ui.ChatViewModel
	nodes     map[string]*ui.NodeViewModel
	connected bool
	endpoint  string
	running   bool
}

func NewConsoleUI(logger *slog.Logger) *ConsoleUI {
	return &ConsoleUI{
		logger: logger,
		chats:  make(map[string]*ui.ChatViewModel),
		nodes:  make(map[string]*ui.NodeViewModel),
	}
}

func (c *ConsoleUI) Run() error {
	c.running = true
	c.logger.Info("Starting console UI")
	
	fmt.Println("=== MeshGo Console UI ===")
	fmt.Println("Type 'help' for available commands")
	
	scanner := bufio.NewScanner(os.Stdin)
	
	for c.running {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		c.handleCommand(line)
	}
	
	return scanner.Err()
}

func (c *ConsoleUI) Shutdown() error {
	c.running = false
	c.logger.Info("Console UI shutdown")
	return nil
}

func (c *ConsoleUI) ShowMain() {
	fmt.Println("Main window shown")
}

func (c *ConsoleUI) HideMain() {
	fmt.Println("Main window hidden")
}

func (c *ConsoleUI) IsVisible() bool {
	return true // Console is always "visible"
}

func (c *ConsoleUI) SetTrayBadge(hasUnread bool) {
	if hasUnread {
		fmt.Println("[TRAY] Unread messages indicator ON")
	} else {
		fmt.Println("[TRAY] Unread messages indicator OFF")
	}
}

func (c *ConsoleUI) ShowTrayNotification(title, body string) error {
	fmt.Printf("[NOTIFICATION] %s: %s\n", title, body)
	return nil
}

func (c *ConsoleUI) UpdateChats(chats []*core.Chat) {
	fmt.Printf("Updated %d chats\n", len(chats))
	for _, chat := range chats {
		vm := ui.ChatToViewModel(chat, nil, nil)
		c.chats[chat.ID] = vm
	}
}

func (c *ConsoleUI) UpdateNodes(nodes []*core.Node) {
	fmt.Printf("Updated %d nodes\n", len(nodes))
	for _, node := range nodes {
		vm := ui.NodeToViewModel(node)
		c.nodes[node.ID] = vm
	}
}

func (c *ConsoleUI) UpdateConnectionStatus(state core.ConnectionState, endpoint string) {
	c.connected = state == core.StateConnected
	c.endpoint = endpoint
	fmt.Printf("Connection status: %s -> %s\n", state.String(), endpoint)
}

func (c *ConsoleUI) UpdateSettings(settings *core.Settings) {
	fmt.Println("Settings updated")
}

func (c *ConsoleUI) ShowTraceroute(node *core.Node, hops []string) {
	fmt.Printf("Traceroute to %s (%s):\n", node.LongName, node.ID)
	for i, hop := range hops {
		fmt.Printf("  %d. %s\n", i+1, hop)
	}
}

func (c *ConsoleUI) ShowError(title, message string) {
	fmt.Printf("[ERROR] %s: %s\n", title, message)
}

func (c *ConsoleUI) ShowInfo(title, message string) {
	fmt.Printf("[INFO] %s: %s\n", title, message)
}

func (c *ConsoleUI) SetEventCallbacks(callbacks *ui.EventCallbacks) {
	c.callbacks = callbacks
}

func (c *ConsoleUI) handleCommand(cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch strings.ToLower(parts[0]) {
	case "help":
		c.showHelp()
	case "connect":
		c.handleConnect(parts[1:])
	case "disconnect":
		c.handleDisconnect()
	case "status":
		c.showStatus()
	case "chats":
		c.listChats()
	case "nodes":
		c.listNodes()
	case "send":
		c.handleSend(parts[1:])
	case "trace":
		c.handleTraceroute(parts[1:])
	case "favorite":
		c.handleFavorite(parts[1:])
	case "exit", "quit":
		c.handleExit()
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
	}
}

func (c *ConsoleUI) showHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  help                    - Show this help")
	fmt.Println("  connect serial <port>   - Connect via serial")
	fmt.Println("  connect ip <host:port>  - Connect via TCP/IP")
	fmt.Println("  disconnect              - Disconnect")
	fmt.Println("  status                  - Show connection status")
	fmt.Println("  chats                   - List chats")
	fmt.Println("  nodes                   - List nodes")
	fmt.Println("  send <chat> <message>   - Send message")
	fmt.Println("  trace <nodeID>          - Traceroute to node")
	fmt.Println("  favorite <nodeID>       - Toggle node favorite")
	fmt.Println("  exit                    - Exit application")
}

func (c *ConsoleUI) handleConnect(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: connect <type> <endpoint>")
		return
	}

	connType := strings.ToLower(args[0])
	endpoint := args[1]

	if c.callbacks == nil || c.callbacks.OnConnect == nil {
		fmt.Println("No connection handler available")
		return
	}

	if err := c.callbacks.OnConnect(connType, endpoint); err != nil {
		fmt.Printf("Connection failed: %v\n", err)
	} else {
		fmt.Printf("Connecting to %s via %s...\n", endpoint, connType)
	}
}

func (c *ConsoleUI) handleDisconnect() {
	if c.callbacks == nil || c.callbacks.OnDisconnect == nil {
		fmt.Println("No disconnect handler available")
		return
	}

	if err := c.callbacks.OnDisconnect(); err != nil {
		fmt.Printf("Disconnect failed: %v\n", err)
	} else {
		fmt.Println("Disconnecting...")
	}
}

func (c *ConsoleUI) showStatus() {
	if c.connected {
		fmt.Printf("Status: Connected to %s\n", c.endpoint)
	} else {
		fmt.Println("Status: Disconnected")
	}
	
	fmt.Printf("Chats: %d\n", len(c.chats))
	fmt.Printf("Nodes: %d\n", len(c.nodes))
}

func (c *ConsoleUI) listChats() {
	if len(c.chats) == 0 {
		fmt.Println("No chats available")
		return
	}
	
	fmt.Println("Chats:")
	for _, chat := range c.chats {
		unread := ""
		if chat.UnreadCount > 0 {
			unread = fmt.Sprintf(" (%d unread)", chat.UnreadCount)
		}
		fmt.Printf("  %s: %s%s\n", chat.ID, chat.Title, unread)
	}
}

func (c *ConsoleUI) listNodes() {
	if len(c.nodes) == 0 {
		fmt.Println("No nodes available")
		return
	}
	
	fmt.Println("Nodes:")
	for _, node := range c.nodes {
		status := "offline"
		if node.IsOnline {
			status = "online"
		}
		
		battery := ""
		if node.BatteryPercent > 0 {
			battery = fmt.Sprintf(" [%d%%]", node.BatteryPercent)
		}
		
		favorite := ""
		if node.Favorite {
			favorite = " ⭐"
		}
		
		fmt.Printf("  %s: %s (%s) - %s%s%s\n", 
			node.ID, node.LongName, node.ShortName, status, battery, favorite)
	}
}

func (c *ConsoleUI) handleSend(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: send <chatID> <message>")
		return
	}
	
	chatID := args[0]
	message := strings.Join(args[1:], " ")
	
	if c.callbacks == nil || c.callbacks.OnSendMessage == nil {
		fmt.Println("No send handler available")
		return
	}
	
	if err := c.callbacks.OnSendMessage(chatID, message); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	} else {
		fmt.Printf("Sent message to %s: %s\n", chatID, message)
	}
}

func (c *ConsoleUI) handleTraceroute(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: trace <nodeID>")
		return
	}
	
	nodeID := args[0]
	
	if c.callbacks == nil || c.callbacks.OnTraceroute == nil {
		fmt.Println("No traceroute handler available")
		return
	}
	
	if err := c.callbacks.OnTraceroute(nodeID); err != nil {
		fmt.Printf("Traceroute failed: %v\n", err)
	} else {
		fmt.Printf("Starting traceroute to %s...\n", nodeID)
	}
}

func (c *ConsoleUI) handleFavorite(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: favorite <nodeID>")
		return
	}
	
	nodeID := args[0]
	
	if c.callbacks == nil || c.callbacks.OnToggleNodeFavorite == nil {
		fmt.Println("No favorite handler available")
		return
	}
	
	if err := c.callbacks.OnToggleNodeFavorite(nodeID); err != nil {
		fmt.Printf("Failed to toggle favorite: %v\n", err)
	} else {
		fmt.Printf("Toggled favorite for %s\n", nodeID)
	}
}

func (c *ConsoleUI) handleExit() {
	fmt.Println("Exiting...")
	c.running = false
	
	if c.callbacks != nil && c.callbacks.OnExit != nil {
		c.callbacks.OnExit()
	}
}