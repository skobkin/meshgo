package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/transport"
)

type ConnectionTransport struct {
	mu sync.RWMutex

	cfg       config.ConnectionConfig
	transport transport.Transport
}

func NewConnectionTransport(cfg config.ConnectionConfig) (*ConnectionTransport, error) {
	tr, err := newTransportForConnection(cfg)
	if err != nil {
		return nil, err
	}

	return &ConnectionTransport{
		cfg:       cfg,
		transport: tr,
	}, nil
}

func (t *ConnectionTransport) Apply(cfg config.ConnectionConfig) error {
	next, err := newTransportForConnection(cfg)
	if err != nil {
		return err
	}

	t.mu.Lock()
	current := t.transport
	t.transport = next
	t.cfg = cfg
	t.mu.Unlock()

	if current != nil {
		_ = current.Close()
	}

	return nil
}

func (t *ConnectionTransport) Name() string {
	tr := t.current()
	if tr == nil {
		return "unknown"
	}
	return tr.Name()
}

func (t *ConnectionTransport) StatusTarget() string {
	t.mu.RLock()
	tr := t.transport
	cfg := t.cfg
	t.mu.RUnlock()

	if provider, ok := tr.(transport.StatusTargetProvider); ok {
		target := strings.TrimSpace(provider.StatusTarget())
		if target != "" {
			return target
		}
	}

	return ConnectionTarget(cfg)
}

func (t *ConnectionTransport) Connect(ctx context.Context) error {
	tr := t.current()
	if tr == nil {
		return fmt.Errorf("transport is not configured")
	}
	return tr.Connect(ctx)
}

func (t *ConnectionTransport) Close() error {
	tr := t.current()
	if tr == nil {
		return nil
	}
	return tr.Close()
}

func (t *ConnectionTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	tr := t.current()
	if tr == nil {
		return nil, fmt.Errorf("transport is not configured")
	}
	return tr.ReadFrame(ctx)
}

func (t *ConnectionTransport) WriteFrame(ctx context.Context, payload []byte) error {
	tr := t.current()
	if tr == nil {
		return fmt.Errorf("transport is not configured")
	}
	return tr.WriteFrame(ctx, payload)
}

func (t *ConnectionTransport) current() transport.Transport {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.transport
}

func (t *ConnectionTransport) Config() config.ConnectionConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cfg
}

func NewTransportForConnection(cfg config.ConnectionConfig) (transport.Transport, error) {
	return newTransportForConnection(cfg)
}

func newTransportForConnection(cfg config.ConnectionConfig) (transport.Transport, error) {
	switch cfg.Connector {
	case config.ConnectorIP:
		return transport.NewIPTransport(cfg.Host, DefaultIPPort), nil
	case config.ConnectorSerial:
		return transport.NewSerialTransport(cfg.SerialPort, cfg.SerialBaud), nil
	case config.ConnectorBluetooth:
		return transport.NewBluetoothTransport(cfg.BluetoothAddress, cfg.BluetoothAdapter), nil
	default:
		return nil, fmt.Errorf("unknown connector: %q", cfg.Connector)
	}
}
