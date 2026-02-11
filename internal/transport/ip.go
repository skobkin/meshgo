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

// IPTransport sends and receives framed traffic over a TCP socket.
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

	target := ""
	if t.host != "" {
		target = net.JoinHostPort(t.host, fmt.Sprintf("%d", t.port))
	}
	logger := transportLogger("ip", "target", target)

	if t.conn != nil {
		logger.Debug("connect skipped: already connected")

		return nil
	}

	if t.host == "" {
		logger.Warn("connect failed: host is empty")

		return errors.New("ip host is empty")
	}

	dialer := net.Dialer{Timeout: 6 * time.Second}
	logger.Info("connecting")
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(t.host, fmt.Sprintf("%d", t.port)))
	if err != nil {
		logger.Warn("connect failed", "error", err)

		return fmt.Errorf("dial tcp: %w", err)
	}
	t.conn = conn
	logger.Info("connected", "remote", conn.RemoteAddr().String())

	return nil
}

func (t *IPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	target := ""
	if t.host != "" {
		target = net.JoinHostPort(t.host, fmt.Sprintf("%d", t.port))
	}
	logger := transportLogger("ip", "target", target)

	if t.conn == nil {
		logger.Debug("close skipped: not connected")

		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	if err != nil {
		logger.Warn("close failed", "error", err)

		return err
	}
	logger.Info("closed")

	return err
}

func (t *IPTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	logger := transportLogger("ip")
	conn, err := t.currentConn()
	if err != nil {
		logger.Debug("read frame failed: not connected", "error", err)

		return nil, err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	} else {
		_ = conn.SetReadDeadline(time.Time{})
	}

	payload, err := readFrame(ioReadFullFunc(conn))
	if err != nil {
		logger.Debug("read frame failed", "error", err)

		return nil, err
	}
	logger.Debug("read frame", "len", len(payload))

	return payload, nil
}

func (t *IPTransport) WriteFrame(ctx context.Context, payload []byte) error {
	logger := transportLogger("ip")
	conn, err := t.currentConn()
	if err != nil {
		logger.Debug("write frame failed: not connected", "error", err)

		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	frame, err := encodeFrame(payload)
	if err != nil {
		logger.Warn("encode frame failed", "payload_len", len(payload), "error", err)

		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	if _, err := conn.Write(frame); err != nil {
		logger.Warn("write frame failed", "payload_len", len(payload), "frame_len", len(frame), "error", err)

		return fmt.Errorf("write frame: %w", err)
	}
	logger.Debug("write frame", "payload_len", len(payload), "frame_len", len(frame))

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
