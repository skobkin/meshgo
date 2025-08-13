package transport

import (
	"context"
	"errors"
)

// SerialTransport is a placeholder for serial connections. Real implementation
// will use go.bug.st/serial to interact with Meshtastic devices.
type SerialTransport struct {
	port string
}

// NewSerial creates a serial transport for the given port.
func NewSerial(port string) *SerialTransport { return &SerialTransport{port: port} }

func (s *SerialTransport) Connect(ctx context.Context) error {
	return errors.New("serial transport not implemented")
}
func (s *SerialTransport) Close() error { return nil }
func (s *SerialTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	return nil, errors.New("serial transport not implemented")
}
func (s *SerialTransport) WritePacket(ctx context.Context, b []byte) error {
	return errors.New("serial transport not implemented")
}
func (s *SerialTransport) IsConnected() bool { return false }
func (s *SerialTransport) Endpoint() string  { return s.port }
