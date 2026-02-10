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

type Node struct {
	NodeID       string
	LongName     string
	ShortName    string
	BatteryLevel *uint32
	Voltage      *float64
	BoardModel   string
	Role         string
	LastHeardAt  time.Time
	RSSI         *int
	SNR          *float64
	UpdatedAt    time.Time
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
