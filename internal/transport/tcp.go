package transport

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type TCPTransport struct {
	host string
	port int
	conn net.Conn
	mu   sync.RWMutex
}

func NewTCPTransport(host string, port int) *TCPTransport {
	if port <= 0 {
		port = 4403 // Default Meshtastic TCP port
	}
	return &TCPTransport{
		host: host,
		port: port,
	}
}

func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		return nil // Already connected
	}

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", t.host, t.port)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	// Enable keepalive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			conn.Close()
			return fmt.Errorf("failed to set keepalive: %w", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			conn.Close()
			return fmt.Errorf("failed to set keepalive period: %w", err)
		}
	}

	t.conn = conn
	return nil
}

func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return nil
	}

	err := t.conn.Close()
	t.conn = nil
	return err
}

func (t *TCPTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return nil, errors.New("not connected")
	}

	done := make(chan struct{})
	var result []byte
	var err error

	go func() {
		defer close(done)
		result, err = t.readFramedPacket(conn)
	}()

	select {
	case <-done:
		return result, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *TCPTransport) WritePacket(ctx context.Context, data []byte) error {
	t.mu.RLock()
	conn := t.conn
	t.mu.RUnlock()

	if conn == nil {
		return errors.New("not connected")
	}

	done := make(chan error, 1)

	go func() {
		framed := t.framePacket(data)
		_, err := conn.Write(framed)
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *TCPTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.conn != nil
}

func (t *TCPTransport) Endpoint() string {
	return fmt.Sprintf("tcp://%s:%d", t.host, t.port)
}

func (t *TCPTransport) readFramedPacket(conn net.Conn) ([]byte, error) {
	const maxPacketSize = 1024
	
	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read packet header (4 bytes for Meshtastic framing)
	header := make([]byte, 4)
	n, err := conn.Read(header)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet header: %w", err)
	}
	if n < 4 {
		return nil, errors.New("incomplete packet header")
	}

	// Validate magic bytes
	if header[0] != 0x94 || header[1] != 0xC3 {
		return nil, errors.New("invalid packet header magic")
	}

	// Extract packet size
	packetSize := int(header[2])<<8 | int(header[3])
	if packetSize > maxPacketSize {
		return nil, fmt.Errorf("packet size too large: %d", packetSize)
	}

	// Read packet data
	buffer := make([]byte, packetSize)
	n, err = conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet data: %w", err)
	}
	if n < packetSize {
		return nil, errors.New("incomplete packet data")
	}

	return buffer, nil
}

func (t *TCPTransport) framePacket(data []byte) []byte {
	// Meshtastic TCP framing: magic bytes + size + data
	framed := make([]byte, len(data)+4)
	framed[0] = 0x94 // Start delimiter
	framed[1] = 0xC3 // Start delimiter  
	framed[2] = byte(len(data) >> 8)   // Size high byte
	framed[3] = byte(len(data) & 0xFF) // Size low byte
	copy(framed[4:], data)
	return framed
}

func (t *TCPTransport) ping(ctx context.Context) error {
	// Send keepalive ping if connection is idle
	// This would send a proper Meshtastic ping packet in real implementation
	return nil
}