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
			_ = conn.Close()
			return fmt.Errorf("failed to set keepalive: %w", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			_ = conn.Close()
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
	const maxPacketSize = 512 // Meshtastic max packet size for BLE compatibility

	// Set read deadline but not too aggressive - Meshtastic devices are slow
	if err := conn.SetReadDeadline(time.Now().Add(35 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Use a simple blocking read approach that waits for data
	// This is more efficient than polling and handles the "no data available" case better
	buffer := make([]byte, 4+maxPacketSize) // Header + max payload

	// Read at least the 4-byte header first
	headerRead := 0
	for headerRead < 4 {
		n, err := conn.Read(buffer[headerRead:4])
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		headerRead += n
	}

	// Check magic bytes in the header we just read
	if buffer[0] != 0x94 || buffer[1] != 0xC3 {
		// Return error to trigger retry - header logging removed to reduce noise
		return nil, fmt.Errorf("invalid frame header - out of sync")
	}

	// Extract packet size from header bytes 2-3 (big-endian)
	packetSize := int(buffer[2])<<8 | int(buffer[3])

	// Debug logging removed to reduce verbosity

	if packetSize <= 0 || packetSize > maxPacketSize {
		// Packet size validation - error logging removed to reduce verbosity
		return nil, fmt.Errorf("invalid packet size: %d", packetSize)
	}

	// Read the packet data
	payloadRead := 0
	for payloadRead < packetSize {
		n, err := conn.Read(buffer[4+payloadRead : 4+packetSize])
		if err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
		payloadRead += n
	}

	// Extract just the payload (skip the 4-byte header)
	payload := make([]byte, packetSize)
	copy(payload, buffer[4:4+packetSize])

	// Debug logging removed to reduce verbosity

	return payload, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *TCPTransport) framePacket(data []byte) []byte {
	// Meshtastic TCP framing: [0x94][0xC3][length_high][length_low] + protobuf data
	// Upper 16 bits: 0x94C3 (magic bytes)
	// Lower 16 bits: packet length (max 512 bytes)

	if len(data) > 512 {
		// This shouldn't happen with proper Meshtastic packets
		panic("packet too large for Meshtastic protocol")
	}

	framed := make([]byte, len(data)+4)
	framed[0] = 0x94                   // Magic byte 1
	framed[1] = 0xC3                   // Magic byte 2
	framed[2] = byte(len(data) >> 8)   // Length high byte
	framed[3] = byte(len(data) & 0xFF) // Length low byte
	copy(framed[4:], data)
	return framed
}

func (t *TCPTransport) ping(ctx context.Context) error {
	// Send keepalive ping if connection is idle
	// This would send a proper Meshtastic ping packet in real implementation
	return nil
}
