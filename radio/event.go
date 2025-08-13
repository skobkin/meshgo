package radio

import (
	"time"

	"meshgo/domain"
)

type EventType int

const (
	EventConnecting EventType = iota
	EventConnected
	EventDisconnected
	EventRetrying
	EventPacket
	EventNode
)

type Event struct {
	Type   EventType
	Err    error
	Delay  time.Duration
	Packet []byte
	Node   *domain.Node
}
