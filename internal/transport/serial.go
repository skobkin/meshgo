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
	if t.port != nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.portName == "" {
		return errors.New("serial port is empty")
	}
	if t.baudRate <= 0 {
		return fmt.Errorf("invalid serial baud rate: %d", t.baudRate)
	}

	port, err := serial.Open(t.portName, &serial.Mode{BaudRate: t.baudRate})
	if err != nil {
		return fmt.Errorf("open serial port %q: %w", t.portName, err)
	}
	if err := port.SetReadTimeout(defaultSerialReadTimeout); err != nil {
		_ = port.Close()
		return fmt.Errorf("set serial read timeout: %w", err)
	}
	t.port = port

	return nil
}

func (t *SerialTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.port == nil {
		return nil
	}
	err := t.port.Close()
	t.port = nil
	return err
}

func (t *SerialTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	port, err := t.currentPort()
	if err != nil {
		return nil, err
	}

	payload, err := readFrame(func(buf []byte) error {
		return t.readFull(ctx, port, buf)
	})
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (t *SerialTransport) WriteFrame(ctx context.Context, payload []byte) error {
	port, err := t.currentPort()
	if err != nil {
		return err
	}

	frame, err := encodeFrame(payload)
	if err != nil {
		return err
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := writeFull(ctx, port, frame); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
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
