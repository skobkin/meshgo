package domain

import "time"

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
	ChatKey         string
	Direction       MessageDirection
	Body            string
	Status          MessageStatus
	StatusReason    string
	At              time.Time
	MetaJSON        string
}

// MessageStatusUpdate updates delivery status by device message id.
type MessageStatusUpdate struct {
	DeviceMessageID string
	Status          MessageStatus
	Reason          string
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
	NodeID          string
	LongName        string
	ShortName       string
	Latitude        *float64
	Longitude       *float64
	BatteryLevel    *uint32
	Voltage         *float64
	Temperature     *float64
	Humidity        *float64
	Pressure        *float64
	AirQualityIndex *float64
	PowerVoltage    *float64
	PowerCurrent    *float64
	BoardModel      string
	Role            string
	IsUnmessageable *bool
	LastHeardAt     time.Time
	RSSI            *int
	SNR             *float64
	UpdatedAt       time.Time
}

// NodeUpdate is a bus event with node data and update source metadata.
type NodeUpdate struct {
	Node       Node
	LastHeard  time.Time
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
