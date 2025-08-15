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
	const maxPacketSize = 512 // Meshtastic max packet size for BLE compatibility
	
	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Find sync bytes - read until we find the magic 0x94C3 sequence
	for {
		// Read first byte
		firstByte := make([]byte, 1)
		n, err := conn.Read(firstByte)
		if err != nil {
			return nil, fmt.Errorf("failed to read first sync byte: %w", err)
		}
		if n < 1 {
			return nil, fmt.Errorf("incomplete sync byte read")
		}

		if firstByte[0] != 0x94 {
			continue // Keep looking for magic byte
		}

		// Read second byte
		secondByte := make([]byte, 1)
		n, err = conn.Read(secondByte)
		if err != nil {
			return nil, fmt.Errorf("failed to read second sync byte: %w", err)
		}
		if n < 1 {
			return nil, fmt.Errorf("incomplete second sync byte read")
		}

		if secondByte[0] != 0xC3 {
			continue // Keep looking for complete magic sequence
		}

		// Found sync bytes, read length
		lengthBytes := make([]byte, 2)
		n, err = conn.Read(lengthBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read length bytes: %w", err)
		}
		if n < 2 {
			return nil, fmt.Errorf("incomplete length read, got %d bytes", n)
		}

		// Extract packet size from length bytes (big-endian)
		packetSize := int(lengthBytes[0])<<8 | int(lengthBytes[1])
		
		// Debug logging
		fmt.Printf("TCP: Found sync - magic:94C3 size:%d\n", packetSize)
		
		if packetSize <= 0 || packetSize > maxPacketSize {
			fmt.Printf("TCP: Invalid packet size %d, continuing sync search\n", packetSize)
			continue // Invalid size, keep looking
		}

		// Read packet data
		buffer := make([]byte, packetSize)
		bytesRead := 0
		for bytesRead < packetSize {
			n, err := conn.Read(buffer[bytesRead:])
			if err != nil {
				return nil, fmt.Errorf("failed to read packet data: %w", err)
			}
			bytesRead += n
		}

		// Debug logging
		fmt.Printf("TCP: Read %d bytes of packet data: %02x...\n", len(buffer), buffer[:min(16, len(buffer))])

		return buffer, nil
	}
}

func min(a, b int) int {
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
	framed[0] = 0x94 // Magic byte 1
	framed[1] = 0xC3 // Magic byte 2
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