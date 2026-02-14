package connectors

import (
	"time"

	"github.com/skobkin/meshgo/internal/traceroute"
)

// ConnectionState describes the connector lifecycle state shown in UI.
type ConnectionState string

const (
	ConnectionStateDisconnected ConnectionState = "disconnected"
	ConnectionStateConnecting   ConnectionState = "connecting"
	ConnectionStateConnected    ConnectionState = "connected"
	ConnectionStateReconnecting ConnectionState = "reconnecting"
)

// ConnectionStatus is a bus event snapshot of current connector status.
type ConnectionStatus struct {
	State         ConnectionState
	Err           string
	TransportName string
	Target        string
	Timestamp     time.Time
}

// RawFrame carries frame diagnostics for debug/log views.
type RawFrame struct {
	Hex string
	Len int
}

// ConfigSnapshot contains parsed device config values needed by UI.
type ConfigSnapshot struct {
	ChannelTitles []string
}

// TracerouteEvent is a decoded TRACEROUTE_APP payload from the radio.
type TracerouteEvent struct {
	From       uint32
	To         uint32
	PacketID   uint32
	RequestID  uint32
	ReplyID    uint32
	Route      []uint32
	SnrTowards []int32
	RouteBack  []uint32
	SnrBack    []int32
	IsComplete bool
}

// TracerouteUpdate is a UI-facing traceroute progress snapshot.
type TracerouteUpdate struct {
	RequestID    uint32
	TargetNodeID string
	StartedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  time.Time
	Status       traceroute.Status
	ForwardRoute []string
	ForwardSNR   []int32
	ReturnRoute  []string
	ReturnSNR    []int32
	Error        string
	DurationMS   int64
}
