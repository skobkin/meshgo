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

// NodeSettingsTarget identifies which node should be read/modified.
// IsLocal is reserved for future remote editing support.
type NodeSettingsTarget struct {
	NodeID  string
	IsLocal bool
}

// NodeUserSettings contains editable owner/user settings.
type NodeUserSettings struct {
	NodeID          string
	LongName        string
	ShortName       string
	HamLicensed     bool
	IsUnmessageable bool
}

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

func (s *NodeSettingsService) LoadUserSettings(ctx context.Context, target NodeSettingsTarget) (NodeUserSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeUserSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeUserSettings{}, parseErr
	}
	s.logger.Info("requesting node user settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_owner", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetOwnerRequest{GetOwnerRequest: true},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node user settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeUserSettings{}, err
		}
		s.logger.Warn(
			"requesting node user settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}
	user := resp.GetGetOwnerResponse()
	if user == nil {
		s.logger.Warn("requesting node user settings returned empty response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeUserSettings{}, fmt.Errorf("owner response is empty")
	}
	s.logger.Info("received node user settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeUserSettings{
		NodeID:          strings.TrimSpace(target.NodeID),
		LongName:        strings.TrimSpace(user.GetLongName()),
		ShortName:       strings.TrimSpace(user.GetShortName()),
		HamLicensed:     user.GetIsLicensed(),
		IsUnmessageable: user.GetIsUnmessagable(),
	}, nil
}

func (s *NodeSettingsService) SaveUserSettings(ctx context.Context, target NodeSettingsTarget, settings NodeUserSettings) error {
	if s == nil || s.bus == nil || s.radio == nil {
		return fmt.Errorf("node settings service is not initialized")
	}
	if !s.isConnected() {
		return fmt.Errorf("device is not connected")
	}

	nodeNum, err := parseNodeID(target.NodeID)
	if err != nil {
		return err
	}
	s.logger.Info("saving node user settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	release, err := s.beginSave()
	if err != nil {
		return err
	}
	defer release()

	saveCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "begin_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_BeginEditSettings{BeginEditSettings: true},
	}); err != nil {
		s.logger.Warn("begin edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("begin edit settings: %w", err)
	}

	admin := &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_SetOwner{
			SetOwner: &generated.User{
				Id:             strings.TrimSpace(target.NodeID),
				LongName:       strings.TrimSpace(settings.LongName),
				ShortName:      strings.TrimSpace(settings.ShortName),
				IsLicensed:     settings.HamLicensed,
				IsUnmessagable: boolPtr(settings.IsUnmessageable),
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_owner", admin); err != nil {
		s.logger.Warn("set owner failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set owner: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node user settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
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

func matchesAdminResponse(event radio.AdminMessageEvent, requestID uint32) bool {
	if event.ReplyID != 0 {
		return event.ReplyID == requestID
	}

	return event.RequestID == requestID
}

func isRetriableReadError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
