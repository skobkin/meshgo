package core

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

type ReconnectManager struct {
	settings  *ReconnectSettings
	transport Transport
	logger    *slog.Logger

	state   ConnectionState
	stateMu sync.RWMutex

	currentDelay    time.Duration
	connectionStart time.Time
	lastConnected   time.Time
	retryCount      int

	cancelFunc context.CancelFunc
	events     chan Event
}

func NewReconnectManager(settings *ReconnectSettings, transport Transport, logger *slog.Logger) *ReconnectManager {
	return &ReconnectManager{
		settings:     settings,
		transport:    transport,
		logger:       logger,
		state:        StateDisconnected,
		events:       make(chan Event, 10),
		currentDelay: time.Duration(settings.InitialMillis) * time.Millisecond,
	}
}

func (rm *ReconnectManager) Start(ctx context.Context) {
	rm.logger.Info("Starting reconnect manager")

	ctx, cancel := context.WithCancel(ctx)
	rm.cancelFunc = cancel

	go rm.connectionLoop(ctx)
}

func (rm *ReconnectManager) Stop() {
	rm.logger.Info("Stopping reconnect manager")

	if rm.cancelFunc != nil {
		rm.cancelFunc()
	}

	rm.setState(StateDisconnected)

	if rm.transport.IsConnected() {
		rm.transport.Close()
	}
}

func (rm *ReconnectManager) ConnectNow() {
	// Trigger immediate connection attempt
	rm.currentDelay = time.Duration(rm.settings.InitialMillis) * time.Millisecond
	rm.retryCount = 0

	rm.logger.Info("Immediate connection requested")
}

func (rm *ReconnectManager) Events() <-chan Event {
	return rm.events
}

func (rm *ReconnectManager) State() ConnectionState {
	rm.stateMu.RLock()
	defer rm.stateMu.RUnlock()
	return rm.state
}

func (rm *ReconnectManager) IsConnected() bool {
	return rm.State() == StateConnected
}

func (rm *ReconnectManager) connectionLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			rm.logger.Info("Connection loop cancelled")
			return
		default:
		}

		if !rm.transport.IsConnected() {
			rm.attemptConnection(ctx)
		}

		if rm.transport.IsConnected() {
			rm.monitorConnection(ctx)
		} else {
			rm.waitForRetry(ctx)
		}
	}
}

func (rm *ReconnectManager) attemptConnection(ctx context.Context) {
	rm.setState(StateConnecting)
	rm.connectionStart = time.Now()

	rm.logger.Info("Attempting connection",
		"endpoint", rm.transport.Endpoint(),
		"retry", rm.retryCount)

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := rm.transport.Connect(connectCtx)
	if err != nil {
		rm.logger.Warn("Connection failed", "error", err, "retry", rm.retryCount)
		rm.setState(StateRetrying)
		rm.retryCount++
		rm.increaseDelay()
		return
	}

	// Connection successful
	rm.setState(StateConnected)
	rm.lastConnected = time.Now()
	rm.resetDelay()

	connectionDuration := time.Since(rm.connectionStart)
	rm.logger.Info("Connected successfully",
		"endpoint", rm.transport.Endpoint(),
		"duration", connectionDuration)
}

func (rm *ReconnectManager) monitorConnection(ctx context.Context) {
	// Connection is active, monitor for failures
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for rm.transport.IsConnected() {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Could add keepalive ping here
			continue
		}
	}

	// Connection lost
	connectionDuration := time.Since(rm.lastConnected)
	rm.logger.Warn("Connection lost",
		"endpoint", rm.transport.Endpoint(),
		"duration", connectionDuration)

	rm.setState(StateRetrying)

	// Reset delay if connection was stable for more than 60 seconds
	if connectionDuration > 60*time.Second {
		rm.resetDelay()
		rm.retryCount = 0
		rm.logger.Debug("Connection was stable, reset backoff")
	}
}

func (rm *ReconnectManager) waitForRetry(ctx context.Context) {
	if rm.retryCount == 0 {
		return // First attempt, no wait
	}

	delay := rm.getJitteredDelay()
	rm.logger.Info("Waiting before retry",
		"delay", delay,
		"retry", rm.retryCount)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}

func (rm *ReconnectManager) setState(state ConnectionState) {
	rm.stateMu.Lock()
	oldState := rm.state
	rm.state = state
	rm.stateMu.Unlock()

	if oldState != state {
		rm.logger.Debug("Connection state changed",
			"from", oldState.String(),
			"to", state.String())

		// Emit state change event
		event := Event{
			Type: EventConnectionStateChanged,
			Data: ConnectionStateData{
				State:    state,
				Endpoint: rm.transport.Endpoint(),
				Error:    nil,
			},
		}

		select {
		case rm.events <- event:
		default:
			rm.logger.Warn("Event queue full, dropping state change event")
		}
	}
}

func (rm *ReconnectManager) increaseDelay() {
	maxDelay := time.Duration(rm.settings.MaxMillis) * time.Millisecond
	rm.currentDelay = time.Duration(float64(rm.currentDelay) * rm.settings.Multiplier)

	if rm.currentDelay > maxDelay {
		rm.currentDelay = maxDelay
	}

	rm.logger.Debug("Increased retry delay",
		"delay", rm.currentDelay,
		"retry", rm.retryCount)
}

func (rm *ReconnectManager) resetDelay() {
	rm.currentDelay = time.Duration(rm.settings.InitialMillis) * time.Millisecond
	rm.retryCount = 0
	rm.logger.Debug("Reset retry delay", "delay", rm.currentDelay)
}

func (rm *ReconnectManager) getJitteredDelay() time.Duration {
	jitter := rm.settings.Jitter
	if jitter <= 0 {
		return rm.currentDelay
	}

	// Add ±jitter% randomness
	multiplier := 1.0 + (rand.Float64()-0.5)*2*jitter
	jitteredDelay := time.Duration(float64(rm.currentDelay) * multiplier)

	// Ensure minimum delay
	minDelay := time.Duration(rm.settings.InitialMillis) * time.Millisecond
	if jitteredDelay < minDelay {
		jitteredDelay = minDelay
	}

	return jitteredDelay
}

type ConnectionStateData struct {
	State    ConnectionState
	Endpoint string
	Error    error
}
