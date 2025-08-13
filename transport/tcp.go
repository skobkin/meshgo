package transport

import (
	"context"
	"net"
	"time"
)

// TCPTransport connects to a Meshtastic device over TCP.
type TCPTransport struct {
	addr string
	conn net.Conn
}

// NewTCP creates a TCP transport for the given address.
func NewTCP(addr string) *TCPTransport { return &TCPTransport{addr: addr} }

// Connect dials the configured TCP endpoint.
func (t *TCPTransport) Connect(ctx context.Context) error {
	var err error
	t.conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", t.addr)
	return err
}

// Close closes the underlying connection if open.
func (t *TCPTransport) Close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

// ReadPacket reads up to 1024 bytes from the connection. Real implementation
// will handle framing and variable packet sizes.
func (t *TCPTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	if t.conn == nil {
		return nil, net.ErrClosed
	}
	_ = t.conn.SetReadDeadline(deadlineFromContext(ctx))
	buf := make([]byte, 1024)
	n, err := t.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// WritePacket writes raw bytes to the connection.
func (t *TCPTransport) WritePacket(ctx context.Context, b []byte) error {
	if t.conn == nil {
		return net.ErrClosed
	}
	_ = t.conn.SetWriteDeadline(deadlineFromContext(ctx))
	_, err := t.conn.Write(b)
	return err
}

// IsConnected reports whether the transport has an active connection.
func (t *TCPTransport) IsConnected() bool { return t.conn != nil }

// Endpoint returns the configured address.
func (t *TCPTransport) Endpoint() string { return t.addr }

func deadlineFromContext(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Time{}
}
