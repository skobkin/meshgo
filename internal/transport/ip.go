package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"
)

var frameHeader = [2]byte{0x94, 0xC3}

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

	if err := t.resyncToHeader(conn); err != nil {
		return nil, err
	}

	var lenBuf [2]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read frame length: %w", err)
	}
	ln := int(binary.BigEndian.Uint16(lenBuf[:]))
	if ln <= 0 {
		return nil, fmt.Errorf("invalid frame length: %d", ln)
	}

	payload := make([]byte, ln)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, fmt.Errorf("read frame payload: %w", err)
	}
	return payload, nil
}

func (t *IPTransport) WriteFrame(ctx context.Context, payload []byte) error {
	conn, err := t.currentConn()
	if err != nil {
		return err
	}
	if len(payload) > math.MaxUint16 {
		return fmt.Errorf("payload too large: %d", len(payload))
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	frame := make([]byte, 4+len(payload))
	frame[0] = frameHeader[0]
	frame[1] = frameHeader[1]
	// #nosec G115 -- length is bounded by math.MaxUint16 above.
	payloadLen := uint16(len(payload))
	binary.BigEndian.PutUint16(frame[2:4], payloadLen)
	copy(frame[4:], payload)

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

func (t *IPTransport) resyncToHeader(conn net.Conn) error {
	buf := make([]byte, 1)
	for {
		if _, err := io.ReadFull(conn, buf); err != nil {
			return fmt.Errorf("read frame header byte 1: %w", err)
		}
		if buf[0] != frameHeader[0] {
			continue
		}
		if _, err := io.ReadFull(conn, buf); err != nil {
			return fmt.Errorf("read frame header byte 2: %w", err)
		}
		if buf[0] == frameHeader[1] {
			return nil
		}
	}
}
