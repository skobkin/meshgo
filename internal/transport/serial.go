package transport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
)

type SerialTransport struct {
	port     string
	baudRate int
	conn     serial.Port
	mu       sync.RWMutex
}

func NewSerialTransport(port string, baudRate int) *SerialTransport {
	if baudRate <= 0 {
		baudRate = 115200 // Default Meshtastic baud rate
	}
	return &SerialTransport{
		port:     port,
		baudRate: baudRate,
	}
}

func (s *SerialTransport) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		return nil // Already connected
	}

	mode := &serial.Mode{
		BaudRate: s.baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	conn, err := serial.Open(s.port, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %w", s.port, err)
	}

	s.conn = conn
	return nil
}

func (s *SerialTransport) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return nil
	}

	err := s.conn.Close()
	s.conn = nil
	return err
}

func (s *SerialTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return nil, errors.New("not connected")
	}

	done := make(chan struct{})
	var result []byte
	var err error

	go func() {
		defer close(done)
		// Read packet with Meshtastic framing
		result, err = s.readFramedPacket(conn)
	}()

	select {
	case <-done:
		return result, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *SerialTransport) WritePacket(ctx context.Context, data []byte) error {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()

	if conn == nil {
		return errors.New("not connected")
	}

	done := make(chan error, 1)

	go func() {
		framed := s.framePacket(data)
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

func (s *SerialTransport) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn != nil
}

func (s *SerialTransport) Endpoint() string {
	return fmt.Sprintf("serial://%s@%d", s.port, s.baudRate)
}

func (s *SerialTransport) readFramedPacket(conn serial.Port) ([]byte, error) {
	const maxPacketSize = 1024
	buffer := make([]byte, maxPacketSize)

	// Set read timeout
	if err := conn.SetReadTimeout(5 * time.Second); err != nil {
		return nil, fmt.Errorf("failed to set read timeout: %w", err)
	}

	// Simple implementation - in real app would need proper Meshtastic framing
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	if n == 0 {
		return nil, errors.New("no data received")
	}

	return buffer[:n], nil
}

func (s *SerialTransport) framePacket(data []byte) []byte {
	// Simple implementation - in real app would need proper Meshtastic framing
	// Meshtastic uses packet delimiters and size headers
	framed := make([]byte, len(data)+4)
	framed[0] = 0x94                   // Start delimiter
	framed[1] = 0xC3                   // Start delimiter
	framed[2] = byte(len(data) >> 8)   // Size high byte
	framed[3] = byte(len(data) & 0xFF) // Size low byte
	copy(framed[4:], data)
	return framed
}

func DetectSerialPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to list serial ports: %w", err)
	}

	var meshtasticPorts []string
	for _, port := range ports {
		// Add basic filtering for common Meshtastic devices
		// In real implementation, would check VID/PID
		if isLikelyMeshtasticPort(port) {
			meshtasticPorts = append(meshtasticPorts, port)
		}
	}

	if len(meshtasticPorts) == 0 {
		return ports, nil // Return all ports if no specific ones found
	}

	return meshtasticPorts, nil
}

func isLikelyMeshtasticPort(port string) bool {
	// Basic heuristic - in real app would check device descriptors
	// Common patterns for ESP32-based Meshtastic devices
	commonPatterns := []string{
		"ttyUSB", "ttyACM", "COM", "cu.usbserial", "cu.wchusbserial",
	}

	for _, pattern := range commonPatterns {
		if len(port) >= len(pattern) &&
			port[len(port)-len(pattern):] == pattern {
			return true
		}
	}
	return false
}
