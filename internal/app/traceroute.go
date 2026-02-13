package app

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

var (
	tracerouteCooldownWindow = 30 * time.Second
	tracerouteRequestTimeout = 60 * time.Second
)

// TracerouteTarget identifies which node should be traced.
type TracerouteTarget struct {
	NodeID string
}

// TracerouteStarter provides UI-facing traceroute start operation.
type TracerouteStarter interface {
	StartTraceroute(ctx context.Context, target TracerouteTarget) (connectors.TracerouteUpdate, error)
}

type tracerouteSender interface {
	SendTraceroute(to uint32, channel uint32) (string, error)
}

// TracerouteCooldownError is returned when a new request is blocked by cooldown.
type TracerouteCooldownError struct {
	Remaining time.Duration
}

func (e *TracerouteCooldownError) Error() string {
	if e == nil {
		return "traceroute is in cooldown"
	}
	remaining := e.Remaining
	if remaining < 0 {
		remaining = 0
	}

	return fmt.Sprintf("traceroute is in cooldown for %s", remaining.Round(time.Second))
}

type pendingTraceroute struct {
	targetNodeID string
	startedAt    time.Time
	updatedAt    time.Time
	forwardRoute []string
	forwardSNR   []int32
	returnRoute  []string
	returnSNR    []int32
}

// TracerouteService handles request dispatch, cooldown, and progress publication.
type TracerouteService struct {
	bus        bus.MessageBus
	radio      tracerouteSender
	nodeStore  *domain.NodeStore
	connStatus func() (connectors.ConnectionStatus, bool)
	logger     *slog.Logger

	startMu sync.Mutex
	mu      sync.Mutex
	pending map[uint32]pendingTraceroute
	lastRun time.Time
}

func NewTracerouteService(
	messageBus bus.MessageBus,
	sender tracerouteSender,
	nodeStore *domain.NodeStore,
	connStatus func() (connectors.ConnectionStatus, bool),
	logger *slog.Logger,
) *TracerouteService {
	if logger == nil {
		logger = slog.Default().With("component", "app.traceroute")
	}

	return &TracerouteService{
		bus:        messageBus,
		radio:      sender,
		nodeStore:  nodeStore,
		connStatus: connStatus,
		logger:     logger,
		pending:    make(map[uint32]pendingTraceroute),
	}
}

func (s *TracerouteService) Start(ctx context.Context) {
	if s == nil || s.bus == nil {
		return
	}
	traceSub := s.bus.Subscribe(connectors.TopicTraceroute)
	statusSub := s.bus.Subscribe(connectors.TopicMessageStatus)
	connSub := s.bus.Subscribe(connectors.TopicConnStatus)

	go func() {
		defer s.bus.Unsubscribe(traceSub, connectors.TopicTraceroute)
		defer s.bus.Unsubscribe(statusSub, connectors.TopicMessageStatus)
		defer s.bus.Unsubscribe(connSub, connectors.TopicConnStatus)

		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-traceSub:
				if !ok {
					return
				}
				event, ok := raw.(connectors.TracerouteEvent)
				if !ok {
					continue
				}
				s.handleTracerouteEvent(event)
			case raw, ok := <-statusSub:
				if !ok {
					return
				}
				update, ok := raw.(domain.MessageStatusUpdate)
				if !ok {
					continue
				}
				s.handleMessageStatus(update)
			case raw, ok := <-connSub:
				if !ok {
					continue
				}
				status, ok := raw.(connectors.ConnectionStatus)
				if !ok {
					continue
				}
				s.handleConnectionStatus(status)
			}
		}
	}()
}

func (s *TracerouteService) StartTraceroute(ctx context.Context, target TracerouteTarget) (connectors.TracerouteUpdate, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return connectors.TracerouteUpdate{}, fmt.Errorf("traceroute service is not initialized")
	}
	if !s.isConnected() {
		return connectors.TracerouteUpdate{}, fmt.Errorf("device is not connected")
	}

	s.startMu.Lock()
	defer s.startMu.Unlock()

	now := time.Now()
	if remaining := s.cooldownRemaining(now); remaining > 0 {
		return connectors.TracerouteUpdate{}, &TracerouteCooldownError{Remaining: remaining}
	}

	nodeNum, err := parseNodeID(target.NodeID)
	if err != nil {
		return connectors.TracerouteUpdate{}, err
	}
	normalizedNodeID := formatNodeID(nodeNum)
	channel := s.resolveNodeChannel(normalizedNodeID)
	packetIDRaw, err := s.radio.SendTraceroute(nodeNum, channel)
	if err != nil {
		return connectors.TracerouteUpdate{}, err
	}
	requestID, err := parsePacketID(packetIDRaw)
	if err != nil {
		return connectors.TracerouteUpdate{}, err
	}

	startedAt := time.Now()
	pending := pendingTraceroute{
		targetNodeID: normalizedNodeID,
		startedAt:    startedAt,
		updatedAt:    startedAt,
	}

	s.mu.Lock()
	s.pending[requestID] = pending
	s.lastRun = startedAt
	s.mu.Unlock()

	update := toTracerouteUpdate(requestID, pending, connectors.TracerouteStatusStarted, "", time.Time{})
	s.bus.Publish(connectors.TopicTracerouteUpdate, update)
	s.logger.Info("started traceroute", "request_id", requestID, "target_node_id", normalizedNodeID, "channel", channel)

	go s.watchTimeout(ctx, requestID)

	return update, nil
}

func (s *TracerouteService) watchTimeout(ctx context.Context, requestID uint32) {
	timer := time.NewTimer(tracerouteRequestTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	now := time.Now()
	s.mu.Lock()
	pending, ok := s.pending[requestID]
	if !ok {
		s.mu.Unlock()

		return
	}
	delete(s.pending, requestID)
	pending.updatedAt = now
	s.mu.Unlock()

	update := toTracerouteUpdate(requestID, pending, connectors.TracerouteStatusTimedOut, "traceroute request timed out", now)
	s.bus.Publish(connectors.TopicTracerouteUpdate, update)
	s.logger.Warn("traceroute timed out", "request_id", requestID, "target_node_id", pending.targetNodeID)
}

func (s *TracerouteService) handleTracerouteEvent(event connectors.TracerouteEvent) {
	requestID := event.RequestID
	if requestID == 0 {
		return
	}

	now := time.Now()
	s.mu.Lock()
	pending, ok := s.pending[requestID]
	if !ok {
		s.mu.Unlock()

		return
	}
	if len(event.Route) > 0 {
		pending.forwardRoute = formatTracerouteNodeIDs(event.Route)
		pending.forwardSNR = append([]int32(nil), event.SnrTowards...)
	}
	if len(event.RouteBack) > 0 {
		pending.returnRoute = formatTracerouteNodeIDs(event.RouteBack)
		pending.returnSNR = append([]int32(nil), event.SnrBack...)
	}
	pending.updatedAt = now
	status := connectors.TracerouteStatusProgress
	completedAt := time.Time{}
	if event.IsComplete {
		status = connectors.TracerouteStatusCompleted
		completedAt = now
		delete(s.pending, requestID)
	} else {
		s.pending[requestID] = pending
	}
	s.mu.Unlock()

	update := toTracerouteUpdate(requestID, pending, status, "", completedAt)
	s.bus.Publish(connectors.TopicTracerouteUpdate, update)
	if status == connectors.TracerouteStatusCompleted {
		s.logger.Info("traceroute completed", "request_id", requestID, "target_node_id", pending.targetNodeID, "duration_ms", update.DurationMS)
	}
}

func (s *TracerouteService) handleMessageStatus(update domain.MessageStatusUpdate) {
	if update.Status != domain.MessageStatusFailed {
		return
	}
	requestID, err := strconv.ParseUint(strings.TrimSpace(update.DeviceMessageID), 10, 32)
	if err != nil {
		return
	}

	now := time.Now()
	s.mu.Lock()
	pending, ok := s.pending[uint32(requestID)]
	if !ok {
		s.mu.Unlock()

		return
	}
	delete(s.pending, uint32(requestID))
	pending.updatedAt = now
	s.mu.Unlock()

	reason := strings.TrimSpace(update.Reason)
	if reason == "" {
		reason = "device rejected traceroute request"
	}
	out := toTracerouteUpdate(uint32(requestID), pending, connectors.TracerouteStatusFailed, reason, now)
	s.bus.Publish(connectors.TopicTracerouteUpdate, out)
	s.logger.Warn("traceroute failed", "request_id", requestID, "target_node_id", pending.targetNodeID, "reason", reason)
}

func (s *TracerouteService) handleConnectionStatus(status connectors.ConnectionStatus) {
	if status.State == connectors.ConnectionStateConnected {
		return
	}

	now := time.Now()
	failed := make([]connectors.TracerouteUpdate, 0)
	s.mu.Lock()
	for requestID, pending := range s.pending {
		pending.updatedAt = now
		failed = append(failed, toTracerouteUpdate(
			requestID,
			pending,
			connectors.TracerouteStatusFailed,
			fmt.Sprintf("connection changed to %s", status.State),
			now,
		))
	}
	s.pending = make(map[uint32]pendingTraceroute)
	s.mu.Unlock()

	for _, update := range failed {
		s.bus.Publish(connectors.TopicTracerouteUpdate, update)
	}
}

func (s *TracerouteService) cooldownRemaining(now time.Time) time.Duration {
	s.mu.Lock()
	lastRun := s.lastRun
	s.mu.Unlock()
	if lastRun.IsZero() {
		return 0
	}
	remaining := tracerouteCooldownWindow - now.Sub(lastRun)
	if remaining < 0 {
		return 0
	}

	return remaining
}

func (s *TracerouteService) resolveNodeChannel(nodeID string) uint32 {
	if s.nodeStore == nil {
		return 0
	}
	node, ok := s.nodeStore.Get(nodeID)
	if !ok || node.Channel == nil {
		return 0
	}

	return *node.Channel
}

func (s *TracerouteService) isConnected() bool {
	if s.connStatus == nil {
		return false
	}
	status, known := s.connStatus()

	return known && status.State == connectors.ConnectionStateConnected
}

func toTracerouteUpdate(
	requestID uint32,
	pending pendingTraceroute,
	status connectors.TracerouteStatus,
	errMsg string,
	completedAt time.Time,
) connectors.TracerouteUpdate {
	update := connectors.TracerouteUpdate{
		RequestID:    requestID,
		TargetNodeID: pending.targetNodeID,
		StartedAt:    pending.startedAt,
		UpdatedAt:    pending.updatedAt,
		Status:       status,
		ForwardRoute: append([]string(nil), pending.forwardRoute...),
		ForwardSNR:   append([]int32(nil), pending.forwardSNR...),
		ReturnRoute:  append([]string(nil), pending.returnRoute...),
		ReturnSNR:    append([]int32(nil), pending.returnSNR...),
		Error:        strings.TrimSpace(errMsg),
		DurationMS:   pending.updatedAt.Sub(pending.startedAt).Milliseconds(),
	}
	if !completedAt.IsZero() {
		update.CompletedAt = completedAt
	}

	return update
}

func formatTracerouteNodeIDs(route []uint32) []string {
	out := make([]string, 0, len(route))
	for _, nodeNum := range route {
		out = append(out, formatNodeID(nodeNum))
	}

	return out
}

func formatNodeID(nodeNum uint32) string {
	if nodeNum == 0 {
		return "unknown"
	}

	return fmt.Sprintf("!%08x", nodeNum)
}
