package domain

import (
	"time"

	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

// ChatType classifies chat destination kind.
type ChatType int

const (
	ChatTypeChannel ChatType = iota + 1
	ChatTypeDM
)

// MessageDirection indicates whether a message was received or sent locally.
type MessageDirection int

const (
	MessageDirectionIn MessageDirection = iota + 1
	MessageDirectionOut
)

// MessageStatus tracks delivery progress for a chat message.
type MessageStatus int

const (
	MessageStatusPending MessageStatus = iota + 1
	MessageStatusSent
	MessageStatusAcked
	MessageStatusFailed
)

// Chat is a UI-facing chat summary record.
type Chat struct {
	Key            string
	Title          string
	Type           ChatType
	LastSentByMeAt time.Time
	UpdatedAt      time.Time
}

// ChatMessage is a single message item stored and shown in a chat timeline.
type ChatMessage struct {
	LocalID         int64
	DeviceMessageID string
	// ReplyToDeviceMessageID links this message to the original message packet id.
	ReplyToDeviceMessageID string
	// Emoji is non-zero for reaction-style text packets in Meshtastic protocol.
	Emoji        uint32
	ChatKey      string
	Direction    MessageDirection
	Body         string
	Status       MessageStatus
	StatusReason string
	At           time.Time
	MetaJSON     string
}

// MessageStatusUpdate updates delivery status by device message id.
type MessageStatusUpdate struct {
	DeviceMessageID string
	Status          MessageStatus
	Reason          string
	FromNodeNum     uint32
}

func ShouldTransitionMessageStatus(current, next MessageStatus) bool {
	if next == 0 || current == next {
		return false
	}
	if current == 0 {
		return true
	}

	switch next {
	case MessageStatusAcked:
		return current != MessageStatusAcked
	case MessageStatusFailed:
		return current != MessageStatusAcked && current != MessageStatusFailed
	case MessageStatusSent:
		return current == MessageStatusPending
	case MessageStatusPending:
		return false
	default:
		return false
	}
}

// Node stores the latest known node metadata and telemetry.
type Node struct {
	NodeID    string
	LongName  string
	ShortName string
	PublicKey []byte
	Channel   *uint32
	// Coordinates are kept as decimal degrees in memory; codec logic converts
	// to/from Meshtastic fixed-point int32 values (degrees * 1e7).
	Latitude              *float64
	Longitude             *float64
	Altitude              *int32
	PositionPrecisionBits *uint32
	BatteryLevel          *uint32
	Voltage               *float64
	UptimeSeconds         *uint32
	ChannelUtilization    *float64
	AirUtilTx             *float64
	Temperature           *float64
	Humidity              *float64
	Pressure              *float64
	SoilTemperature       *float64
	SoilMoisture          *uint32
	GasResistance         *float64
	Lux                   *float64
	UVLux                 *float64
	Radiation             *float64
	AirQualityIndex       *float64
	PowerVoltage          *float64
	PowerCurrent          *float64
	BoardModel            string
	FirmwareVersion       string
	Role                  string
	IsUnmessageable       *bool
	PositionUpdatedAt     time.Time
	LastHeardAt           time.Time
	RSSI                  *int
	SNR                   *float64
	UpdatedAt             time.Time
}

// NodeCore stores primary identity/activity snapshot fields.
type NodeCore struct {
	NodeID          string
	LongName        string
	ShortName       string
	PublicKey       []byte
	Channel         *uint32
	BoardModel      string
	FirmwareVersion string
	Role            string
	IsUnmessageable *bool
	LastHeardAt     time.Time
	RSSI            *int
	SNR             *float64
	UpdatedAt       time.Time
}

// NodePosition stores latest known node geospatial data and related metadata.
type NodePosition struct {
	NodeID                string
	Channel               *uint32
	Latitude              *float64
	Longitude             *float64
	Altitude              *int32
	PositionPrecisionBits *uint32
	PositionUpdatedAt     time.Time
	ObservedAt            time.Time
	UpdatedAt             time.Time
}

// NodeTelemetry stores latest known node telemetry metrics and related metadata.
type NodeTelemetry struct {
	NodeID             string
	Channel            *uint32
	BatteryLevel       *uint32
	Voltage            *float64
	UptimeSeconds      *uint32
	ChannelUtilization *float64
	AirUtilTx          *float64
	Temperature        *float64
	Humidity           *float64
	Pressure           *float64
	SoilTemperature    *float64
	SoilMoisture       *uint32
	GasResistance      *float64
	Lux                *float64
	UVLux              *float64
	Radiation          *float64
	AirQualityIndex    *float64
	PowerVoltage       *float64
	PowerCurrent       *float64
	ObservedAt         time.Time
	UpdatedAt          time.Time
}

// NodeCoreUpdate carries node core data with source metadata.
type NodeCoreUpdate struct {
	Core       NodeCore
	FromPacket bool
	Type       NodeUpdateType
}

// NodePositionUpdate carries node position data with source metadata.
type NodePositionUpdate struct {
	Position   NodePosition
	FromPacket bool
	Type       NodeUpdateType
}

// NodeTelemetryUpdate carries node telemetry data with source metadata.
type NodeTelemetryUpdate struct {
	Telemetry  NodeTelemetry
	FromPacket bool
	Type       NodeUpdateType
}

// NodeUpdate is a bus event with node data and update source metadata.
type NodeUpdate struct {
	Node       Node
	LastHeard  time.Time
	FromPacket bool
	Type       NodeUpdateType
}

// NodeUpdateType identifies which radio frame kind produced a node update.
type NodeUpdateType string

const (
	NodeUpdateTypeUnknown          NodeUpdateType = ""
	NodeUpdateTypeNodeInfoSnapshot NodeUpdateType = "nodeinfo_snapshot"
	NodeUpdateTypeNodeInfoPacket   NodeUpdateType = "nodeinfo_packet"
	NodeUpdateTypeTelemetryPacket  NodeUpdateType = "telemetry_packet"
	NodeUpdateTypePositionPacket   NodeUpdateType = "position_packet"
	NodeUpdateTypeMetadata         NodeUpdateType = "metadata"
)

// NodeDiscovered is emitted when a previously unknown node is seen in live traffic.
type NodeDiscovered struct {
	Node         Node
	NodeID       string
	DiscoveredAt time.Time
	Source       string
}

// NodePositionHistoryEntry is one persisted position history point for a node.
type NodePositionHistoryEntry struct {
	RowID      int64
	NodeID     string
	Channel    *uint32
	Latitude   *float64
	Longitude  *float64
	Altitude   *int32
	Precision  *uint32
	ObservedAt time.Time
	WrittenAt  time.Time
	UpdateType NodeUpdateType
	FromPacket bool
}

// NodeTelemetryHistoryEntry is one persisted telemetry history point for a node.
type NodeTelemetryHistoryEntry struct {
	RowID              int64
	NodeID             string
	Channel            *uint32
	BatteryLevel       *uint32
	Voltage            *float64
	UptimeSeconds      *uint32
	ChannelUtilization *float64
	AirUtilTx          *float64
	Temperature        *float64
	Humidity           *float64
	Pressure           *float64
	SoilTemperature    *float64
	SoilMoisture       *uint32
	GasResistance      *float64
	Lux                *float64
	UVLux              *float64
	Radiation          *float64
	AirQualityIndex    *float64
	PowerVoltage       *float64
	PowerCurrent       *float64
	ObservedAt         time.Time
	WrittenAt          time.Time
	UpdateType         NodeUpdateType
	FromPacket         bool
}

// NodeIdentityHistoryEntry is one persisted identity history point for a node.
type NodeIdentityHistoryEntry struct {
	RowID      int64
	NodeID     string
	LongName   string
	ShortName  string
	PublicKey  []byte
	ObservedAt time.Time
	WrittenAt  time.Time
	UpdateType NodeUpdateType
	FromPacket bool
}

// ChannelList carries known device channels published by the radio.
type ChannelList struct {
	Items []ChannelInfo
}

// ChannelInfo describes one mesh channel index and title.
type ChannelInfo struct {
	Index int
	Title string
}

// TracerouteRecord stores one traceroute run state for future history UI.
type TracerouteRecord struct {
	RequestID    string
	TargetNodeID string
	StartedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  time.Time
	Status       busmsg.TracerouteStatus
	ForwardRoute []string
	ForwardSNR   []int32
	ReturnRoute  []string
	ReturnSNR    []int32
	ErrorText    string
	DurationMS   int64
}
