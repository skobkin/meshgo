package connectors

import "time"

type ConnectionState string

const (
	ConnectionStateDisconnected ConnectionState = "disconnected"
	ConnectionStateConnecting   ConnectionState = "connecting"
	ConnectionStateConnected    ConnectionState = "connected"
	ConnectionStateReconnecting ConnectionState = "reconnecting"
)

type ConnStatus struct {
	State         ConnectionState
	Err           string
	TransportName string
	Timestamp     time.Time
}

type RawFrame struct {
	Hex string
	Len int
}

type ConfigSnapshot struct {
	ChannelTitles []string
}
