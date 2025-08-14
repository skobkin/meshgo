package transport

import (
	"context"
	"log/slog"
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
	slog.Info("tcp connect", "addr", t.addr)
	var err error
	t.conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", t.addr)
	if err != nil {
		slog.Error("tcp connect failed", "addr", t.addr, "err", err)
		return err
	}
	slog.Info("tcp connected", "addr", t.addr)
	return nil
}

// Close closes the underlying connection if open.
func (t *TCPTransport) Close() error {
	if t.conn != nil {
		slog.Info("tcp close", "addr", t.addr)
		err := t.conn.Close()
		t.conn = nil
		if err != nil {
			slog.Error("tcp close error", "addr", t.addr, "err", err)
		}
		return err
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
		slog.Error("tcp read", "err", err)
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
	if err != nil {
		slog.Error("tcp write", "err", err)
	}
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
