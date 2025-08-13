package radio

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"meshgo/transport"
)

// ReconnectConfig defines exponential backoff parameters for auto-reconnect.
type ReconnectConfig struct {
	InitialMillis int
	MaxMillis     int
	Multiplier    float64
	Jitter        float64
}

// Client manages a transport connection and emits events about its state.
type Client struct {
	cfg    ReconnectConfig
	events chan Event
	tMu    sync.RWMutex
	t      transport.Transport
}

// New creates a Client with the provided reconnect configuration.
func New(cfg ReconnectConfig) *Client {
	if cfg.InitialMillis <= 0 {
		cfg.InitialMillis = 1000
	}
	if cfg.MaxMillis <= 0 {
		cfg.MaxMillis = 60000
	}
	if cfg.Multiplier <= 1 {
		cfg.Multiplier = 1.6
	}
	if cfg.Jitter < 0 {
		cfg.Jitter = 0.2
	}
	return &Client{cfg: cfg, events: make(chan Event, 16)}
}

// Events returns a channel that receives connection state events.
func (c *Client) Events() <-chan Event { return c.events }

// Start begins the connect/read loop for the given transport and blocks until the
// context is cancelled.
func (c *Client) Start(ctx context.Context, t transport.Transport) error {
	backoff := time.Duration(c.cfg.InitialMillis) * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		c.events <- Event{Type: EventConnecting}
		if err := t.Connect(ctx); err != nil {
			c.events <- Event{Type: EventDisconnected, Err: err}
			delay := jitterDuration(backoff, c.cfg.Jitter)
			c.events <- Event{Type: EventRetrying, Delay: delay}
			if wait(ctx, delay) {
				return ctx.Err()
			}
			backoff = nextBackoff(backoff, c.cfg)
			continue
		}
		c.events <- Event{Type: EventConnected}
		c.setTransport(t)
		start := time.Now()
		if err := c.readLoop(ctx, t); err != nil {
			c.events <- Event{Type: EventDisconnected, Err: err}
		}
		c.setTransport(nil)
		_ = t.Close()
		if time.Since(start) >= time.Minute {
			backoff = time.Duration(c.cfg.InitialMillis) * time.Millisecond
		} else {
			backoff = nextBackoff(backoff, c.cfg)
		}
		delay := jitterDuration(backoff, c.cfg.Jitter)
		c.events <- Event{Type: EventRetrying, Delay: delay}
		if wait(ctx, delay) {
			return ctx.Err()
		}
	}
}

func (c *Client) readLoop(ctx context.Context, t transport.Transport) error {
	for {
		pkt, err := t.ReadPacket(ctx)
		if err != nil {
			return err
		}
		c.events <- Event{Type: EventPacket, Packet: pkt}
	}
}

// SendText writes the provided text to the active transport connection.
func (c *Client) SendText(ctx context.Context, chatID string, toNode uint32, text string) error {
	c.tMu.RLock()
	t := c.t
	c.tMu.RUnlock()
	if t == nil {
		return errors.New("not connected")
	}
	return t.WritePacket(ctx, []byte(text))
}

func (c *Client) setTransport(t transport.Transport) {
	c.tMu.Lock()
	c.t = t
	c.tMu.Unlock()
}

func nextBackoff(current time.Duration, cfg ReconnectConfig) time.Duration {
	next := time.Duration(float64(current) * cfg.Multiplier)
	max := time.Duration(cfg.MaxMillis) * time.Millisecond
	if next > max {
		return max
	}
	return next
}

func jitterDuration(base time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return base
	}
	r := (rand.Float64()*2 - 1) * jitter
	return base + time.Duration(r*float64(base))
}

func wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
		return false
	}
}
