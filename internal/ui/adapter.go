package ui

import (
	"time"

	"meshgo/internal/core"
)

type Adapter interface {
	// Lifecycle
	Run() error
	Shutdown() error

	// Window management
	ShowMain()
	HideMain()
	IsVisible() bool

	// Tray integration
	SetTrayBadge(hasUnread bool)
	ShowTrayNotification(title, body string) error

	// Content updates
	UpdateChats(chats []*core.Chat)
	UpdateNodes(nodes []*core.Node) 
	UpdateConnectionStatus(state core.ConnectionState, endpoint string)
	UpdateSettings(settings *core.Settings)

	// Dialogs and popups
	ShowTraceroute(node *core.Node, hops []string)
	ShowError(title, message string)
	ShowInfo(title, message string)
	
	// Event callbacks - UI calls these when user interacts
	SetEventCallbacks(callbacks *EventCallbacks)
}

type EventCallbacks struct {
	// Connection events
	OnConnect     func(connType, endpoint string) error
	OnDisconnect  func() error
	
	// Message events  
	OnSendMessage func(chatID, text string) error
	OnMarkChatRead func(chatID string) error
	OnLoadChatMessages func(chatID string) ([]*core.Message, error)
	OnGetNodeName func(nodeID string) string
	
	// Node events
	OnToggleNodeFavorite  func(nodeID string) error
	OnToggleNodeIgnored   func(nodeID string) error
	OnRemoveNode          func(nodeID string) error
	OnExchangeUserInfo    func(nodeID string) error
	OnTraceroute          func(nodeID string) error
	OnOpenDirectMessage   func(nodeID string) error
	
	// Settings events
	OnUpdateConnection    func(connType, serialPort string, serialBaud int, ipHost string, ipPort int) error
	OnUpdateNotifications func(enabled bool) error
	OnToggleNotifications func(enabled bool) error
	OnUpdateConnectOnStartup func(enabled bool) error
	OnUpdateLogging       func(enabled, level string) error
	OnUpdateLogLevel      func(level string) error
	OnUpdateUI            func(startMinimized bool, theme string) error
	
	// Application events
	OnExit func()
	OnWindowVisibilityChanged func(visible bool)
	
	// Maintenance events
	OnClearChats func() error
	OnClearNodes func() error
}

// Chat represents a chat/conversation for the UI
type ChatViewModel struct {
	*core.Chat
	Messages     []*core.Message `json:"messages"`
	LastMessage  *core.Message   `json:"last_message"`
	Participants []*core.Node    `json:"participants,omitempty"`
}

// Node represents a node for the UI with additional display info
type NodeViewModel struct {
	*core.Node
	StatusText      string `json:"status_text"`
	SignalBars      int    `json:"signal_bars"`     // 0-3 bars
	BatteryPercent  int    `json:"battery_percent"`
	IsOnline        bool   `json:"is_online"`
	DistanceText    string `json:"distance_text,omitempty"`
}

// Connection state for UI display
type ConnectionViewModel struct {
	State            core.ConnectionState `json:"state"`
	Endpoint         string               `json:"endpoint"`
	StateText        string               `json:"state_text"`
	RetryCountdown   int                  `json:"retry_countdown,omitempty"`
	LastError        string               `json:"last_error,omitempty"`
	ConnectedSince   string               `json:"connected_since,omitempty"`
	PacketsRX        int64                `json:"packets_rx"`
	PacketsTX        int64                `json:"packets_tx"`
	PacketsDropped   int64                `json:"packets_dropped"`
}

// Settings for UI display
type SettingsViewModel struct {
	*core.Settings
	AvailablePorts   []string `json:"available_ports"`
	AvailableThemes  []string `json:"available_themes"`
	LogFilePath      string   `json:"log_file_path"`
	DatabasePath     string   `json:"database_path"`
	ConfigPath       string   `json:"config_path"`
	Version          string   `json:"version"`
	BuildInfo        string   `json:"build_info"`
}

// Traceroute result for UI display
type TracerouteViewModel struct {
	Destination *core.Node           `json:"destination"`
	Hops        []*TracerouteHop     `json:"hops"`
	Status      string               `json:"status"`
	StartTime   string               `json:"start_time"`
	Duration    string               `json:"duration"`
}

type TracerouteHop struct {
	HopNumber   int        `json:"hop_number"`
	Node        *core.Node `json:"node"`
	ResponseTime string    `json:"response_time"`
	RSSI        int        `json:"rssi"`
	SNR         float32    `json:"snr"`
}

// Helper functions to convert core models to view models

func ChatToViewModel(chat *core.Chat, messages []*core.Message, participants []*core.Node) *ChatViewModel {
	vm := &ChatViewModel{
		Chat:         chat,
		Messages:     messages,
		Participants: participants,
	}
	
	if len(messages) > 0 {
		vm.LastMessage = messages[len(messages)-1]
	}
	
	return vm
}

func NodeToViewModel(node *core.Node) *NodeViewModel {
	vm := &NodeViewModel{
		Node: node,
	}
	
	// Online status (node heard in last 5 minutes)
	vm.IsOnline = node.LastHeard.After(time.Now().Add(-5 * time.Minute))
	
	// Calculate signal quality for all nodes
	if vm.IsOnline {
		// Check if we have actual signal data
		if node.RSSI != 0 || node.SNR != 0 {
			// Node is online with real signal data - calculate quality from RSSI/SNR values
			switch core.CalculateSignalQuality(node.RSSI, node.SNR) {
			case core.SignalGood:
				vm.SignalBars = 3
				vm.StatusText = "Good"
			case core.SignalFair:
				vm.SignalBars = 2
				vm.StatusText = "Fair" 
			case core.SignalBad:
				vm.SignalBars = 1
				vm.StatusText = "Poor"
			}
		} else {
			// Node is online but no signal readings yet
			vm.SignalBars = 0
			vm.StatusText = "Unknown"
		}
	} else {
		// Node not heard recently
		vm.SignalBars = 0
		vm.StatusText = "Offline"
	}
	
	// Battery percentage
	if node.BatteryLevel != nil {
		vm.BatteryPercent = *node.BatteryLevel
		if vm.BatteryPercent > 100 {
			vm.BatteryPercent = 100 // Cap at 100% for display
		}
	}
	
	return vm
}

func ConnectionToViewModel(state core.ConnectionState, endpoint string) *ConnectionViewModel {
	vm := &ConnectionViewModel{
		State:     state,
		Endpoint:  endpoint,
		StateText: state.String(),
	}
	
	return vm
}

func SettingsToViewModel(settings *core.Settings, configManager *core.ConfigManager) *SettingsViewModel {
	vm := &SettingsViewModel{
		Settings: settings,
		AvailableThemes: []string{"system", "light", "dark"},
		LogFilePath:     "",
		DatabasePath:    "",
		ConfigPath:      configManager.ConfigDir(),
		Version:         "1.0.0", // Would be injected at build time
		BuildInfo:       "Development build",
	}
	
	if settings.Logging.Enabled {
		vm.LogFilePath = configManager.LogDir()
	}
	
	return vm
}