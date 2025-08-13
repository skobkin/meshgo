package transport

import "context"

// Transport describes a connection to a Meshtastic device or service.
type Transport interface {
	Connect(ctx context.Context) error
	Close() error
	ReadPacket(ctx context.Context) ([]byte, error)
	WritePacket(ctx context.Context, b []byte) error
	IsConnected() bool
	Endpoint() string
}
