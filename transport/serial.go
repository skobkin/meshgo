package transport

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.bug.st/serial"
)

// SerialTransport connects to a Meshtastic device over a serial port.
type SerialTransport struct {
	name string
	baud int
	port serial.Port
}

// NewSerial creates a serial transport for the given port name using the
// default Meshtastic baud rate of 115200.
func NewSerial(name string) *SerialTransport {
	return &SerialTransport{name: name, baud: 115200}
}

// Connect opens the serial port with the configured baud rate.
func (s *SerialTransport) Connect(ctx context.Context) error {
	slog.Info("serial open", "port", s.name, "baud", s.baud)
	mode := &serial.Mode{BaudRate: s.baud}
	p, err := serial.Open(s.name, mode)
	if err != nil {
		slog.Error("serial open failed", "port", s.name, "err", err)
		return err
	}
	s.port = p
	slog.Info("serial opened", "port", s.name)
	return nil
}

// Close closes the serial port if it is open.
func (s *SerialTransport) Close() error {
	if s.port != nil {
		slog.Info("serial close", "port", s.name)
		err := s.port.Close()
		s.port = nil
		if err != nil {
			slog.Error("serial close error", "port", s.name, "err", err)
		}
		return err
	}
	return nil
}

// ReadPacket reads up to 1024 bytes from the serial port.
func (s *SerialTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	if s.port == nil {
		return nil, errors.New("serial port not open")
	}
	if d, ok := ctx.Deadline(); ok {
		_ = s.port.SetReadTimeout(time.Until(d))
	}
	buf := make([]byte, 1024)
	n, err := s.port.Read(buf)
	if err != nil {
		slog.Error("serial read", "err", err)
		return nil, err
	}
	return buf[:n], nil
}

// WritePacket writes the given bytes to the serial port.
func (s *SerialTransport) WritePacket(ctx context.Context, b []byte) error {
	if s.port == nil {
		return errors.New("serial port not open")
	}
	if d, ok := ctx.Deadline(); ok {
		if wt, ok := s.port.(interface{ SetWriteTimeout(time.Duration) error }); ok {
			_ = wt.SetWriteTimeout(time.Until(d))
		}
	}
	_, err := s.port.Write(b)
	if err != nil {
		slog.Error("serial write", "err", err)
	}
	return err
}

// IsConnected reports whether the transport has an open serial port.
func (s *SerialTransport) IsConnected() bool { return s.port != nil }

// Endpoint returns the configured port name.
func (s *SerialTransport) Endpoint() string { return s.name }
