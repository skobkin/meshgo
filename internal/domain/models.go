package domain

import "time"

type ChatType int

const (
	ChatTypeChannel ChatType = iota + 1
	ChatTypeDM
)

type MessageDirection int

const (
	MessageDirectionIn MessageDirection = iota + 1
	MessageDirectionOut
)

type MessageStatus int

const (
	MessageStatusPending MessageStatus = iota + 1
	MessageStatusSent
	MessageStatusAcked
	MessageStatusFailed
)

type Chat struct {
	Key            string
	Title          string
	Type           ChatType
	LastSentByMeAt time.Time
	UpdatedAt      time.Time
}

type ChatMessage struct {
	LocalID         int64
	DeviceMessageID string
	ChatKey         string
	Direction       MessageDirection
	Body            string
	Status          MessageStatus
	At              time.Time
	MetaJSON        string
}

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

type Node struct {
	NodeID          string
	LongName        string
	ShortName       string
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

type NodeUpdate struct {
	Node       Node
	LastHeard  time.Time
	FromPacket bool
}

type ChannelList struct {
	Items []ChannelInfo
}

type ChannelInfo struct {
	Index int
	Title string
}
