package radio

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"github.com/skobkin/meshgo/internal/transport"
)

// SendResult is the async outcome of a user message send request.
type SendResult struct {
	Message domain.ChatMessage
	Err     error
}

type sendRequest struct {
	chatKey string
	text    string
	result  chan SendResult
}

type ackTrackState struct {
	targetNodeNum uint32
}

// Service runs transport I/O, codec translation, and bus publication loops.
type Service struct {
	logger    *slog.Logger
	transport transport.Transport
	codec     Codec
	bus       bus.MessageBus
	outbox    chan sendRequest

	ackTrackMu sync.Mutex
	ackTrack   map[string]ackTrackState
}

type localNodeIDCodec interface {
	LocalNodeID() string
}

func NewService(logger *slog.Logger, b bus.MessageBus, tr transport.Transport, codec Codec) *Service {
	return &Service{
		logger:    logger,
		transport: tr,
		codec:     codec,
		bus:       b,
		outbox:    make(chan sendRequest, 128),
		ackTrack:  make(map[string]ackTrackState),
	}
}

func (s *Service) Start(ctx context.Context) {
	go s.runOutbox(ctx)
	go s.runConnector(ctx)
}

func (s *Service) SendText(chatKey, text string) <-chan SendResult {
	resCh := make(chan SendResult, 1)
	chatKey = strings.TrimSpace(chatKey)
	if chatKey == "" {
		resCh <- SendResult{Err: errors.New("chat key is required")}
		close(resCh)

		return resCh
	}
	if utf8.RuneCountInString(text) == 0 {
		resCh <- SendResult{Err: errors.New("message body is empty")}
		close(resCh)

		return resCh
	}
	if len([]byte(text)) > 200 {
		resCh <- SendResult{Err: fmt.Errorf("message body exceeds 200 bytes: %d", len([]byte(text)))}
		close(resCh)

		return resCh
	}

	s.outbox <- sendRequest{chatKey: chatKey, text: text, result: resCh}

	return resCh
}

func (s *Service) LocalNodeID() string {
	codec, ok := s.codec.(localNodeIDCodec)
	if !ok {
		return ""
	}

	return strings.TrimSpace(codec.LocalNodeID())
}

func (s *Service) runConnector(ctx context.Context) {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		s.publishConnStatus(connectors.ConnectionStateConnecting, nil)
		if err := s.transport.Connect(ctx); err != nil {
			s.publishConnStatus(connectors.ConnectionStateReconnecting, err)
			s.logger.Error("transport connect failed", "error", err)
			if !sleepWithContext(ctx, backoff) {
				return
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}

			continue
		}

		backoff = time.Second
		s.publishConnStatus(connectors.ConnectionStateConnected, nil)
		if err := s.sendWantConfig(ctx); err != nil {
			s.logger.Warn("want_config send failed", "error", err)
		}

		keepAliveCtx, cancelKeepAlive := context.WithCancel(ctx)
		go s.runKeepAlive(keepAliveCtx)
		err := s.runReader(ctx)
		cancelKeepAlive()
		_ = s.transport.Close()
		s.publishConnStatus(connectors.ConnectionStateReconnecting, err)

		if !sleepWithContext(ctx, backoff) {
			return
		}
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (s *Service) runReader(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		payload, err := s.transport.ReadFrame(readCtx)
		cancel()
		if err != nil {
			return err
		}

		s.bus.Publish(connectors.TopicRawFrameIn, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(payload)), Len: len(payload)})
		decoded, err := s.codec.DecodeFromRadio(payload)
		if err != nil {
			s.logger.Warn("decode fromradio failed", "error", err)

			continue
		}
		s.bus.Publish(connectors.TopicRadioFrom, decoded)

		if decoded.NodeUpdate != nil {
			s.bus.Publish(connectors.TopicNodeInfo, *decoded.NodeUpdate)
		}
		if decoded.Channels != nil {
			s.bus.Publish(connectors.TopicChannels, *decoded.Channels)
		}
		if decoded.ConfigSnapshot != nil {
			s.bus.Publish(connectors.TopicConfigSnapshot, *decoded.ConfigSnapshot)
		}
		if decoded.TextMessage != nil {
			s.bus.Publish(connectors.TopicTextMessage, *decoded.TextMessage)
		}
		if decoded.AdminMessage != nil {
			s.bus.Publish(connectors.TopicAdminMessage, *decoded.AdminMessage)
		}
		if decoded.Traceroute != nil {
			s.bus.Publish(connectors.TopicTraceroute, *decoded.Traceroute)
		}
		if decoded.MessageStatus != nil {
			status := s.normalizeMessageStatus(*decoded.MessageStatus)
			s.bus.Publish(connectors.TopicMessageStatus, status)
		}
	}
}

func (s *Service) runKeepAlive(ctx context.Context) {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload, err := s.codec.EncodeHeartbeat()
			if err != nil {
				s.logger.Debug("encode heartbeat failed", "error", err)

				continue
			}
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = s.transport.WriteFrame(writeCtx, payload)
			cancel()
			if err != nil {
				s.logger.Debug("heartbeat write failed", "error", err)

				continue
			}
			s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(payload)), Len: len(payload)})
		}
	}
}

func (s *Service) runOutbox(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.outbox:
			res := s.handleSend(ctx, req)
			req.result <- res
			close(req.result)
		}
	}
}

func (s *Service) handleSend(ctx context.Context, req sendRequest) SendResult {
	encoded, err := s.codec.EncodeText(req.chatKey, req.text)
	if err != nil {
		return SendResult{Err: fmt.Errorf("encode outgoing message: %w", err)}
	}
	writeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	err = s.transport.WriteFrame(writeCtx, encoded.Payload)
	cancel()
	if err != nil {
		return SendResult{Err: fmt.Errorf("send outgoing frame: %w", err)}
	}

	now := time.Now()
	initialStatus := domain.MessageStatusPending
	if encoded.WantAck {
		s.markAckTracked(encoded.DeviceMessageID, encoded.TargetNodeNum)
	}
	msg := domain.ChatMessage{
		DeviceMessageID: encoded.DeviceMessageID,
		ChatKey:         req.chatKey,
		Direction:       domain.MessageDirectionOut,
		Body:            req.text,
		Status:          initialStatus,
		At:              now,
		MetaJSON:        outgoingMessageMetaJSON(s.LocalNodeID()),
	}

	s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(encoded.Payload)), Len: len(encoded.Payload)})
	s.bus.Publish(connectors.TopicTextMessage, msg)

	return SendResult{Message: msg}
}

func (s *Service) sendWantConfig(ctx context.Context) error {
	payload, err := s.codec.EncodeWantConfig()
	if err != nil {
		return err
	}
	writeCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	if err := s.transport.WriteFrame(writeCtx, payload); err != nil {
		return err
	}
	s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(payload)), Len: len(payload)})

	return nil
}

func (s *Service) SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
	if payload == nil {
		return "", fmt.Errorf("admin payload is required")
	}
	encoded, err := s.codec.EncodeAdmin(to, channel, wantResponse, payload)
	if err != nil {
		return "", fmt.Errorf("encode admin payload: %w", err)
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	err = s.transport.WriteFrame(writeCtx, encoded.Payload)
	cancel()
	if err != nil {
		return "", fmt.Errorf("send admin frame: %w", err)
	}
	s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(encoded.Payload)), Len: len(encoded.Payload)})

	return encoded.DeviceMessageID, nil
}

func (s *Service) SendTraceroute(to uint32, channel uint32) (string, error) {
	encoded, err := s.codec.EncodeTraceroute(to, channel)
	if err != nil {
		return "", fmt.Errorf("encode traceroute packet: %w", err)
	}
	writeCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	err = s.transport.WriteFrame(writeCtx, encoded.Payload)
	cancel()
	if err != nil {
		return "", fmt.Errorf("send traceroute frame: %w", err)
	}
	s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(encoded.Payload)), Len: len(encoded.Payload)})

	return encoded.DeviceMessageID, nil
}

func (s *Service) publishConnStatus(state connectors.ConnectionState, err error) {
	status := connectors.ConnectionStatus{
		State:         state,
		TransportName: s.transport.Name(),
		Timestamp:     time.Now(),
	}
	if provider, ok := s.transport.(transport.StatusTargetResolver); ok {
		status.Target = strings.TrimSpace(provider.StatusTarget())
	}
	if err != nil {
		status.Err = err.Error()
	}
	s.bus.Publish(connectors.TopicConnStatus, status)
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func outgoingMessageMetaJSON(localNodeID string) string {
	localNodeID = strings.TrimSpace(localNodeID)
	if localNodeID == "" {
		return ""
	}
	raw, err := json.Marshal(map[string]any{
		"from": localNodeID,
	})
	if err != nil {
		return ""
	}

	return string(raw)
}

func (s *Service) markAckTracked(deviceMessageID string, targetNodeNum uint32) {
	deviceMessageID = strings.TrimSpace(deviceMessageID)
	if deviceMessageID == "" {
		return
	}
	s.ackTrackMu.Lock()
	s.ackTrack[deviceMessageID] = ackTrackState{targetNodeNum: targetNodeNum}
	s.ackTrackMu.Unlock()
}

func (s *Service) clearAckTracked(deviceMessageID string) {
	deviceMessageID = strings.TrimSpace(deviceMessageID)
	if deviceMessageID == "" {
		return
	}
	s.ackTrackMu.Lock()
	delete(s.ackTrack, deviceMessageID)
	s.ackTrackMu.Unlock()
}

func (s *Service) ackTrackStateFor(deviceMessageID string) (ackTrackState, bool) {
	deviceMessageID = strings.TrimSpace(deviceMessageID)
	if deviceMessageID == "" {
		return ackTrackState{}, false
	}
	s.ackTrackMu.Lock()
	state, ok := s.ackTrack[deviceMessageID]
	s.ackTrackMu.Unlock()

	return state, ok
}

func (s *Service) normalizeMessageStatus(update domain.MessageStatusUpdate) domain.MessageStatusUpdate {
	switch update.Status {
	case domain.MessageStatusAcked:
		state, tracked := s.ackTrackStateFor(update.DeviceMessageID)
		if !tracked {
			return update
		}
		if state.targetNodeNum == broadcastNodeNum {
			update.Status = domain.MessageStatusSent
			s.clearAckTracked(update.DeviceMessageID)

			return update
		}
		if update.FromNodeNum != 0 && update.FromNodeNum != state.targetNodeNum {
			update.Status = domain.MessageStatusSent

			return update
		}
		s.clearAckTracked(update.DeviceMessageID)
	case domain.MessageStatusFailed:
		s.clearAckTracked(update.DeviceMessageID)
	}

	return update
}
