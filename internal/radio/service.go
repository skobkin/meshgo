package radio

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/transport"
)

type SendResult struct {
	Message domain.ChatMessage
	Err     error
}

type sendRequest struct {
	chatKey string
	text    string
	result  chan SendResult
}

type Service struct {
	logger    *slog.Logger
	transport transport.Transport
	codec     Codec
	bus       bus.MessageBus
	outbox    chan sendRequest
}

func NewService(logger *slog.Logger, b bus.MessageBus, tr transport.Transport, codec Codec) *Service {
	return &Service{
		logger:    logger,
		transport: tr,
		codec:     codec,
		bus:       b,
		outbox:    make(chan sendRequest, 128),
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
	payload, err := s.codec.EncodeText(req.chatKey, req.text)
	if err != nil {
		return SendResult{Err: fmt.Errorf("encode outgoing message: %w", err)}
	}
	writeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	err = s.transport.WriteFrame(writeCtx, payload)
	cancel()
	if err != nil {
		return SendResult{Err: fmt.Errorf("send outgoing frame: %w", err)}
	}

	now := time.Now()
	msg := domain.ChatMessage{
		ChatKey:   req.chatKey,
		Direction: domain.MessageDirectionOut,
		Body:      req.text,
		Status:    domain.MessageStatusSent,
		At:        now,
	}

	s.bus.Publish(connectors.TopicRawFrameOut, connectors.RawFrame{Hex: strings.ToUpper(hex.EncodeToString(payload)), Len: len(payload)})
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

func (s *Service) publishConnStatus(state connectors.ConnectionState, err error) {
	status := connectors.ConnStatus{
		State:         state,
		TransportName: s.transport.Name(),
		Timestamp:     time.Now(),
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
