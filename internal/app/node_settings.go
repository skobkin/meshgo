package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const (
	nodeSettingsOpTimeout = 10 * time.Second
	nodeSettingsChannel   = 0
	nodeSettingsReadRetry = 1
)

type adminSender interface {
	SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error)
}

// NodeSettingsService exchanges admin messages with a connected node.
type NodeSettingsService struct {
	bus            bus.MessageBus
	radio          adminSender
	connStatus     func() (connectors.ConnectionStatus, bool)
	logger         *slog.Logger
	saveInFlightMu sync.Mutex
	saveInFlight   bool
}

func NewNodeSettingsService(
	messageBus bus.MessageBus,
	sender adminSender,
	connStatus func() (connectors.ConnectionStatus, bool),
	logger *slog.Logger,
) *NodeSettingsService {
	if logger == nil {
		logger = slog.Default().With("component", "ui.node_settings")
	}

	return &NodeSettingsService{
		bus:        messageBus,
		radio:      sender,
		connStatus: connStatus,
		logger:     logger,
	}
}

func (s *NodeSettingsService) beginSave() (func(), error) {
	s.saveInFlightMu.Lock()
	if s.saveInFlight {
		s.saveInFlightMu.Unlock()

		return nil, fmt.Errorf("another settings save is already in progress")
	}
	s.saveInFlight = true
	s.saveInFlightMu.Unlock()

	return func() {
		s.saveInFlightMu.Lock()
		s.saveInFlight = false
		s.saveInFlightMu.Unlock()
	}, nil
}

// sendAdminAndWaitResponse is used for read requests where the device must return
// an admin payload (for example get_owner/get_config). It sends with
// wantResponse=true and correlates the response to the sent request ID.
func (s *NodeSettingsService) sendAdminAndWaitResponse(
	ctx context.Context,
	to uint32,
	action string,
	message *generated.AdminMessage,
) (*generated.AdminMessage, error) {
	adminSub := s.bus.Subscribe(connectors.TopicAdminMessage)
	defer s.bus.Unsubscribe(adminSub, connectors.TopicAdminMessage)
	statusSub := s.bus.Subscribe(connectors.TopicMessageStatus)
	defer s.bus.Unsubscribe(statusSub, connectors.TopicMessageStatus)
	connSub := s.bus.Subscribe(connectors.TopicConnStatus)
	defer s.bus.Unsubscribe(connSub, connectors.TopicConnStatus)

	requestID, err := s.sendAdmin(to, true, message)
	if err != nil {
		return nil, err
	}
	s.logger.Info("sent node settings admin request", "action", action, "request_id", requestID, "to", to)

	event, err := s.waitAdminResponse(ctx, adminSub, statusSub, connSub, requestID)
	if err != nil {
		return nil, err
	}
	s.logger.Info(
		"received node settings admin response",
		"action", action,
		"request_id", requestID,
		"response_request_id", event.RequestID,
		"response_reply_id", event.ReplyID,
		"from", event.From,
	)
	if event.Message == nil {
		return nil, fmt.Errorf("empty admin response")
	}

	return event.Message, nil
}

// sendAdminAndWaitStatus is used for write/update requests where delivery status
// (sent/acked/failed) is sufficient and no admin response payload is expected.
func (s *NodeSettingsService) sendAdminAndWaitStatus(ctx context.Context, to uint32, action string, message *generated.AdminMessage) error {
	sub := s.bus.Subscribe(connectors.TopicMessageStatus)
	defer s.bus.Unsubscribe(sub, connectors.TopicMessageStatus)
	connSub := s.bus.Subscribe(connectors.TopicConnStatus)
	defer s.bus.Unsubscribe(connSub, connectors.TopicConnStatus)

	requestID, err := s.sendAdmin(to, false, message)
	if err != nil {
		return err
	}
	s.logger.Info("sent node settings admin update", "action", action, "request_id", requestID, "to", to)

	if err := s.waitStatus(ctx, sub, connSub, requestID); err != nil {
		return err
	}
	s.logger.Info("received node settings delivery status", "action", action, "request_id", requestID)

	return nil
}

// sendAdmin is a low-level helper that sends one admin message and normalizes the
// returned packet ID string into uint32 so higher-level wait helpers can correlate events.
func (s *NodeSettingsService) sendAdmin(to uint32, wantResponse bool, message *generated.AdminMessage) (uint32, error) {
	packetIDRaw, err := s.radio.SendAdmin(to, nodeSettingsChannel, wantResponse, message)
	if err != nil {
		return 0, err
	}
	packetID, err := parsePacketID(packetIDRaw)
	if err != nil {
		return 0, err
	}

	return packetID, nil
}

// waitAdminResponse waits for the admin payload matching requestID while also
// watching message-status failures and connection drops for early explicit errors.
func (s *NodeSettingsService) waitAdminResponse(
	ctx context.Context,
	adminSub bus.Subscription,
	statusSub bus.Subscription,
	connSub bus.Subscription,
	requestID uint32,
) (*radio.AdminMessageEvent, error) {
	requestDeviceMessageID := strconv.FormatUint(uint64(requestID), 10)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait admin response %d: %w", requestID, ctx.Err())
		case raw, ok := <-statusSub:
			if !ok {
				return nil, fmt.Errorf("message status subscription closed")
			}
			update, ok := raw.(domain.MessageStatusUpdate)
			if !ok {
				continue
			}
			if strings.TrimSpace(update.DeviceMessageID) != requestDeviceMessageID {
				continue
			}
			if update.Status == domain.MessageStatusFailed {
				if strings.TrimSpace(update.Reason) == "" {
					return nil, fmt.Errorf("device rejected admin request")
				}

				return nil, fmt.Errorf("device rejected admin request: %s", strings.TrimSpace(update.Reason))
			}
		case raw, ok := <-adminSub:
			if !ok {
				return nil, fmt.Errorf("admin response subscription closed")
			}
			event, ok := raw.(radio.AdminMessageEvent)
			if !ok {
				continue
			}
			if !matchesAdminResponse(event, requestID) {
				continue
			}

			return &event, nil
		case raw, ok := <-connSub:
			if !ok {
				continue
			}
			status, ok := raw.(connectors.ConnectionStatus)
			if !ok {
				continue
			}
			if status.State != connectors.ConnectionStateConnected {
				return nil, fmt.Errorf("connection changed to %s while waiting admin response", status.State)
			}
		}
	}
}

// waitStatus waits for the status update matching requestID and treats failed
// statuses or connection drops as terminal errors for write/update operations.
func (s *NodeSettingsService) waitStatus(ctx context.Context, sub bus.Subscription, connSub bus.Subscription, requestID uint32) error {
	requestDeviceMessageID := strconv.FormatUint(uint64(requestID), 10)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait status for %d: %w", requestID, ctx.Err())
		case raw, ok := <-sub:
			if !ok {
				return fmt.Errorf("message status subscription closed")
			}
			update, ok := raw.(domain.MessageStatusUpdate)
			if !ok {
				continue
			}
			if strings.TrimSpace(update.DeviceMessageID) != requestDeviceMessageID {
				continue
			}
			if update.Status == domain.MessageStatusFailed {
				if strings.TrimSpace(update.Reason) == "" {
					return fmt.Errorf("device reported settings failure")
				}

				return fmt.Errorf("device reported settings failure: %s", strings.TrimSpace(update.Reason))
			}
			if update.Status == domain.MessageStatusAcked || update.Status == domain.MessageStatusSent {
				return nil
			}
		case raw, ok := <-connSub:
			if !ok {
				continue
			}
			status, ok := raw.(connectors.ConnectionStatus)
			if !ok {
				continue
			}
			if status.State != connectors.ConnectionStateConnected {
				return fmt.Errorf("connection changed to %s while waiting message status", status.State)
			}
		}
	}
}

func (s *NodeSettingsService) isConnected() bool {
	if s.connStatus == nil {
		return false
	}
	status, known := s.connStatus()

	return known && status.State == connectors.ConnectionStateConnected
}

func parseNodeID(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("node id is empty")
	}
	if strings.HasPrefix(raw, "!") {
		v, err := strconv.ParseUint(strings.TrimPrefix(raw, "!"), 16, 32)
		if err != nil {
			return 0, fmt.Errorf("parse node id %q: %w", raw, err)
		}

		return uint32(v), nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "0x") {
		v, err := strconv.ParseUint(raw, 0, 32)
		if err != nil {
			return 0, fmt.Errorf("parse node id %q: %w", raw, err)
		}

		return uint32(v), nil
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse node id %q: %w", raw, err)
	}

	return uint32(v), nil
}

func parsePacketID(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("empty packet id")
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse packet id %q: %w", raw, err)
	}

	return uint32(v), nil
}

func boolPtr(value bool) *bool {
	return &value
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}

	out := make([]byte, len(value))
	copy(out, value)

	return out
}

func cloneBytesList(values [][]byte) [][]byte {
	if len(values) == 0 {
		return nil
	}

	out := make([][]byte, 0, len(values))
	for _, value := range values {
		out = append(out, cloneBytes(value))
	}

	return out
}

func matchesAdminResponse(event radio.AdminMessageEvent, requestID uint32) bool {
	if event.ReplyID != 0 {
		return event.ReplyID == requestID
	}

	return event.RequestID == requestID
}

func isRetriableReadError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
