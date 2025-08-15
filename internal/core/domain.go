package core

import (
	"context"
	"time"
)

type Node struct {
	ID            string         `json:"id"`
	ShortName     string         `json:"short_name"`
	LongName      string         `json:"long_name"`
	Favorite      bool           `json:"favorite"`
	Ignored       bool           `json:"ignored"`
	Unencrypted   bool           `json:"unencrypted"`
	EncDefaultKey bool           `json:"enc_default_key"`
	EncCustomKey  bool           `json:"enc_custom_key"`
	RSSI          int            `json:"rssi"`
	SNR           float32        `json:"snr"`
	SignalQuality int            `json:"signal_quality"`
	BatteryLevel  *int           `json:"battery_level"`
	IsCharging    *bool          `json:"is_charging"`
	LastHeard     time.Time      `json:"last_heard"`
	Position      *Position      `json:"position,omitempty"`
	DeviceMetrics *DeviceMetrics `json:"device_metrics,omitempty"`
}

type Position struct {
	LatitudeI      int32   `json:"latitude_i"`
	LongitudeI     int32   `json:"longitude_i"`
	Altitude       int32   `json:"altitude"`
	Time           uint32  `json:"time"`
	LocationSource int     `json:"location_source"`
	AltitudeSource int     `json:"altitude_source"`
	GPSAccuracy    float32 `json:"gps_accuracy"`
}

func (p *Position) Latitude() float64 {
	return float64(p.LatitudeI) * 1e-7
}

func (p *Position) Longitude() float64 {
	return float64(p.LongitudeI) * 1e-7
}

type DeviceMetrics struct {
	BatteryLevel uint32  `json:"battery_level"`
	Voltage      float32 `json:"voltage"`
}

func (dm *DeviceMetrics) IsCharging() bool {
	return dm.BatteryLevel > 100
}

type User struct {
	ID         string `json:"id"`
	LongName   string `json:"long_name"`
	ShortName  string `json:"short_name"`
	MacAddr    []byte `json:"mac_addr"`
	HWModel    int    `json:"hw_model"`
	IsLicensed bool   `json:"is_licensed"`
}

type Chat struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	Encryption    int       `json:"encryption"`
	LastMessageTS time.Time `json:"last_message_ts"`
	UnreadCount   int       `json:"unread_count"`
	IsChannel     bool      `json:"is_channel"`
}

type Message struct {
	ID        int64     `json:"id"`
	ChatID    string    `json:"chat_id"`
	SenderID  string    `json:"sender_id"`
	PortNum   int       `json:"portnum"`
	Text      string    `json:"text"`
	RXSNR     *float32  `json:"rx_snr"`
	RXRSSI    *int      `json:"rx_rssi"`
	Timestamp time.Time `json:"timestamp"`
	IsUnread  bool      `json:"is_unread"`
}

type Channel struct {
	Name     string `json:"name"`
	PSKClass int    `json:"psk_class"`
	PSK      []byte `json:"psk"`
	ID       uint32 `json:"id"`
}

type SignalQuality int

const (
	SignalBad SignalQuality = iota
	SignalFair
	SignalGood
)

func (sq SignalQuality) String() string {
	switch sq {
	case SignalGood:
		return "Good"
	case SignalFair:
		return "Fair"
	default:
		return "Bad"
	}
}

func CalculateSignalQuality(rssi int, snr float32) SignalQuality {
	// If we have no signal info, it means the node is offline/not heard recently
	if rssi == 0 && snr == 0 {
		return SignalBad // Offline/no signal
	}

	// Good signal: Strong RSSI AND good SNR
	if rssi >= -95 && snr >= 8 {
		return SignalGood
	}

	// Fair signal: Moderate RSSI and SNR (not both poor)
	if rssi >= -110 && snr >= 2 && !(rssi <= -120 || snr <= 1) {
		return SignalFair
	}

	// Bad signal: Poor RSSI or poor SNR
	return SignalBad
}

type EncryptionState int

const (
	EncryptionNone EncryptionState = iota
	EncryptionDefault
	EncryptionCustom
)

func (es EncryptionState) String() string {
	switch es {
	case EncryptionNone:
		return "Not encrypted"
	case EncryptionDefault:
		return "Encrypted (default)"
	case EncryptionCustom:
		return "Encrypted (custom)"
	default:
		return "Unknown"
	}
}

func DetermineEncryption(psk []byte) EncryptionState {
	switch len(psk) {
	case 0:
		return EncryptionNone
	case 1:
		if psk[0] >= 1 && psk[0] <= 10 {
			return EncryptionDefault
		}
		return EncryptionNone
	case 16, 32:
		return EncryptionCustom
	default:
		return EncryptionNone
	}
}

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateRetrying
)

func (cs ConnectionState) String() string {
	switch cs {
	case StateConnected:
		return "Connected"
	case StateConnecting:
		return "Connecting"
	case StateRetrying:
		return "Retrying"
	default:
		return "Disconnected"
	}
}

type EventType int

const (
	EventMessageReceived EventType = iota
	EventNodeUpdated
	EventConnectionStateChanged
	EventUnreadCountChanged
	EventChatUpdated
)

type Event struct {
	Type EventType `json:"type"`
	Data any       `json:"data"`
}

type Transport interface {
	Connect(ctx context.Context) error
	Close() error
	ReadPacket(ctx context.Context) ([]byte, error)
	WritePacket(ctx context.Context, data []byte) error
	IsConnected() bool
	Endpoint() string
}

type RadioClient interface {
	Start(ctx context.Context, transport Transport) error
	SendText(ctx context.Context, chatID string, toNode uint32, text string) error
	SendExchangeUserInfo(ctx context.Context, node uint32) error
	SendTraceroute(ctx context.Context, node uint32) error
	Events() <-chan Event
	Stop() error
}

type MessageStore interface {
	SaveMessage(ctx context.Context, msg *Message) error
	GetMessages(ctx context.Context, chatID string, limit int, offset int) ([]*Message, error)
	GetUnreadCount(ctx context.Context, chatID string) (int, error)
	MarkAsRead(ctx context.Context, chatID string) error
	GetTotalUnreadCount(ctx context.Context) (int, error)
}

type NodeStore interface {
	SaveNode(ctx context.Context, node *Node) error
	GetNode(ctx context.Context, id string) (*Node, error)
	GetAllNodes(ctx context.Context) ([]*Node, error)
	DeleteNode(ctx context.Context, id string) error
	UpdateNodeFavorite(ctx context.Context, id string, favorite bool) error
	UpdateNodeIgnored(ctx context.Context, id string, ignored bool) error
}

type SettingsStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	GetBool(key string, defaultVal bool) bool
	SetBool(key string, value bool) error
	GetInt(key string, defaultVal int) int
	SetInt(key string, value int) error
}

type Notifier interface {
	NotifyNewMessage(chatID, title, body string, timestamp time.Time) error
}

type Tray interface {
	SetUnread(hasUnread bool)
	OnShowHide(fn func())
	OnToggleNotifications(fn func(enabled bool))
	OnExit(fn func())
	Run()
}

type UI interface {
	Run()
	ShowMain()
	SetTrayBadge(hasUnread bool)
	Notify(chatID, title, body string)
	UpdateChats(chats []*Chat)
	UpdateNodes(nodes []*Node)
	ShowTraceroute(node *Node, hops []string)
}
