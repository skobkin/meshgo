package connectors

import "time"

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
