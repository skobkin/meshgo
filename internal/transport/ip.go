package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const defaultIPPort = 4403

type IPTransport struct {
	host string
	port int

	mu      sync.Mutex
	conn    net.Conn
	writeMu sync.Mutex
}

func NewIPTransport(host string, port int) *IPTransport {
	if port == 0 {
		port = defaultIPPort
	}
	return &IPTransport{host: host, port: port}
}

func (t *IPTransport) Name() string {
	return "ip"
}

func (t *IPTransport) SetHost(host string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.host = host
}

func (t *IPTransport) Host() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.host
}

func (t *IPTransport) StatusTarget() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.host == "" {
		return ""
	}
	return net.JoinHostPort(t.host, fmt.Sprintf("%d", t.port))
}

func (t *IPTransport) Connected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

func (t *IPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		return nil
	}

	if t.host == "" {
		return errors.New("ip host is empty")
	}

	dialer := net.Dialer{Timeout: 6 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(t.host, fmt.Sprintf("%d", t.port)))
	if err != nil {
		return fmt.Errorf("dial tcp: %w", err)
	}
	t.conn = conn
	return nil
}

func (t *IPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	return err
}

func (t *IPTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	conn, err := t.currentConn()
	if err != nil {
		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(time.Time{})
	}

	payload, err := readFrame(ioReadFullFunc(conn))
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (t *IPTransport) WriteFrame(ctx context.Context, payload []byte) error {
	conn, err := t.currentConn()
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	frame, err := encodeFrame(payload)
	if err != nil {
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if _, err := conn.Write(frame); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	return nil
}

func (t *IPTransport) currentConn() (net.Conn, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil, errors.New("transport is not connected")
	}
	return t.conn, nil
}
