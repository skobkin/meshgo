package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"go.bug.st/serial"
)

const defaultSerialReadTimeout = 300 * time.Millisecond

type SerialTransport struct {
	portName string
	baudRate int

	mu      sync.Mutex
	port    serial.Port
	writeMu sync.Mutex
}

func NewSerialTransport(portName string, baudRate int) *SerialTransport {
	return &SerialTransport{
		portName: portName,
		baudRate: baudRate,
	}
}

func (t *SerialTransport) Name() string {
	return "serial"
}

func (t *SerialTransport) SetConfig(portName string, baudRate int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.portName = portName
	t.baudRate = baudRate
}

func (t *SerialTransport) PortName() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.portName
}

func (t *SerialTransport) StatusTarget() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.portName
}

func (t *SerialTransport) BaudRate() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.baudRate
}

func (t *SerialTransport) Connected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.port != nil
}

func (t *SerialTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	logger := transportLogger("serial", "port", t.portName, "baud", t.baudRate)

	if t.port != nil {
		logger.Debug("connect skipped: already connected")
		return nil
	}
	if err := ctx.Err(); err != nil {
		logger.Debug("connect canceled", "error", err)
		return err
	}
	if t.portName == "" {
		logger.Warn("connect failed: port is empty")
		return errors.New("serial port is empty")
	}
	if t.baudRate <= 0 {
		logger.Warn("connect failed: invalid baud rate", "baud", t.baudRate)
		return fmt.Errorf("invalid serial baud rate: %d", t.baudRate)
	}

	logger.Info("connecting")
	port, err := serial.Open(t.portName, &serial.Mode{BaudRate: t.baudRate})
	if err != nil {
		logger.Warn("open port failed", "error", err)
		return fmt.Errorf("open serial port %q: %w", t.portName, err)
	}
	if err := port.SetReadTimeout(defaultSerialReadTimeout); err != nil {
		_ = port.Close()
		logger.Warn("set read timeout failed", "timeout", defaultSerialReadTimeout, "error", err)
		return fmt.Errorf("set serial read timeout: %w", err)
	}
	t.port = port
	logger.Info("connected")

	return nil
}

func (t *SerialTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	logger := transportLogger("serial", "port", t.portName, "baud", t.baudRate)
	if t.port == nil {
		logger.Debug("close skipped: not connected")
		return nil
	}
	err := t.port.Close()
	t.port = nil
	if err != nil {
		logger.Warn("close failed", "error", err)
		return err
	}
	logger.Info("closed")
	return err
}

func (t *SerialTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	logger := transportLogger("serial")
	port, err := t.currentPort()
	if err != nil {
		logger.Debug("read frame failed: not connected", "error", err)
		return nil, err
	}

	payload, err := readFrame(func(buf []byte) error {
		return t.readFull(ctx, port, buf)
	})
	if err != nil {
		logger.Debug("read frame failed", "error", err)
		return nil, err
	}
	logger.Debug("read frame", "len", len(payload))
	return payload, nil
}

func (t *SerialTransport) WriteFrame(ctx context.Context, payload []byte) error {
	logger := transportLogger("serial")
	port, err := t.currentPort()
	if err != nil {
		logger.Debug("write frame failed: not connected", "error", err)
		return err
	}

	frame, err := encodeFrame(payload)
	if err != nil {
		logger.Warn("encode frame failed", "payload_len", len(payload), "error", err)
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := writeFull(ctx, port, frame); err != nil {
		logger.Warn("write frame failed", "payload_len", len(payload), "frame_len", len(frame), "error", err)
		return fmt.Errorf("write frame: %w", err)
	}
	logger.Debug("write frame", "payload_len", len(payload), "frame_len", len(frame))
	return nil
}

func (t *SerialTransport) currentPort() (serial.Port, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.port == nil {
		return nil, errors.New("transport is not connected")
	}
	return t.port, nil
}

func (t *SerialTransport) readFull(ctx context.Context, r io.Reader, buf []byte) error {
	if len(buf) == 0 {
		return nil
	}

	read := 0
	for read < len(buf) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := r.Read(buf[read:])
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		read += n
	}

	return nil
}

func writeFull(ctx context.Context, w io.Writer, buf []byte) error {
	written := 0
	for written < len(buf) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := w.Write(buf[written:])
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		written += n
	}
	return nil
}
