package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/notifications"
)

const (
	notificationTitleNodeDiscovered = "New node discovered"
)

// NotificationService listens to bus events and emits user-facing notifications.
type NotificationService struct {
	bus           bus.MessageBus
	chatStore     *domain.ChatStore
	nodeStore     *domain.NodeStore
	currentConfig func() config.AppConfig
	isForeground  func() bool
	sender        notifications.Sender
	logger        *slog.Logger

	connStatusMu     sync.Mutex
	lastConnState    connectors.ConnectionState
	lastConnStateSet bool
}

type messageMeta struct {
	From string `json:"from"`
}

func NewNotificationService(
	messageBus bus.MessageBus,
	chatStore *domain.ChatStore,
	nodeStore *domain.NodeStore,
	currentConfig func() config.AppConfig,
	isForeground func() bool,
	sender notifications.Sender,
	logger *slog.Logger,
) *NotificationService {
	if logger == nil {
		logger = slog.Default().With("component", "app.notifications")
	}

	return &NotificationService{
		bus:           messageBus,
		chatStore:     chatStore,
		nodeStore:     nodeStore,
		currentConfig: currentConfig,
		isForeground:  isForeground,
		sender:        sender,
		logger:        logger,
	}
}

func (s *NotificationService) Start(ctx context.Context) {
	if s == nil || s.bus == nil || s.sender == nil {
		return
	}

	textSub := s.bus.Subscribe(connectors.TopicTextMessage)
	nodeSub := s.bus.Subscribe(connectors.TopicNodeDiscovered)
	connSub := s.bus.Subscribe(connectors.TopicConnStatus)

	go func() {
		defer s.bus.Unsubscribe(textSub, connectors.TopicTextMessage)
		defer s.bus.Unsubscribe(nodeSub, connectors.TopicNodeDiscovered)
		defer s.bus.Unsubscribe(connSub, connectors.TopicConnStatus)

		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-textSub:
				if !ok {
					return
				}
				msg, ok := raw.(domain.ChatMessage)
				if !ok {
					continue
				}
				s.handleIncomingMessage(msg)
			case raw, ok := <-nodeSub:
				if !ok {
					return
				}
				event, ok := raw.(domain.NodeDiscovered)
				if !ok {
					continue
				}
				s.handleNodeDiscovered(event)
			case raw, ok := <-connSub:
				if !ok {
					return
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

func (s *NotificationService) handleIncomingMessage(msg domain.ChatMessage) {
	prefs := s.notificationPrefs()
	if msg.Direction != domain.MessageDirectionIn {
		return
	}
	if !s.shouldNotify(prefs, prefs.Events.IncomingMessage) {
		return
	}

	senderName := s.senderNameForMessage(msg)
	if senderName == "" {
		senderName = "unknown"
	}
	body := strings.TrimSpace(msg.Body)
	if body == "" {
		body = "(empty)"
	}

	titlePrefix := "#"
	titleSubject := s.chatTitle(msg.ChatKey)
	if chatTypeForNotification(msg.ChatKey) == domain.ChatTypeDM {
		titlePrefix = "@"
		titleSubject = senderName
	}
	if titleSubject == "" {
		titleSubject = strings.TrimSpace(msg.ChatKey)
	}
	if titleSubject == "" {
		titleSubject = "unknown"
	}

	s.send(notifications.Payload{
		Title:   titlePrefix + titleSubject,
		Content: fmt.Sprintf("%s: %s", senderName, body),
	})
}

func (s *NotificationService) handleNodeDiscovered(event domain.NodeDiscovered) {
	prefs := s.notificationPrefs()
	if !s.shouldNotify(prefs, prefs.Events.NodeDiscovered) {
		return
	}

	content := nodeDiscoveredContent(event)
	if content == "" {
		return
	}
	s.send(notifications.Payload{
		Title:   notificationTitleNodeDiscovered,
		Content: content,
	})
}

func (s *NotificationService) handleConnectionStatus(status connectors.ConnectionStatus) {
	prefs := s.notificationPrefs()
	if status.State == "" {
		return
	}

	s.connStatusMu.Lock()
	if s.lastConnStateSet && s.lastConnState == status.State {
		s.connStatusMu.Unlock()

		return
	}
	s.lastConnState = status.State
	s.lastConnStateSet = true
	s.connStatusMu.Unlock()

	if status.State != connectors.ConnectionStateConnected &&
		status.State != connectors.ConnectionStateDisconnected {
		return
	}
	if !s.shouldNotify(prefs, prefs.Events.ConnectionStatus) {
		return
	}

	transport := notificationTransportName(status.TransportName)
	if transport == "" {
		transport = "Unknown"
	}
	details := strings.TrimSpace(status.Target)
	if details == "" {
		details = "No connection details"
	}
	if status.State == connectors.ConnectionStateDisconnected {
		if errText := strings.TrimSpace(status.Err); errText != "" {
			details = fmt.Sprintf("%s (error: %s)", details, errText)
		}
	}

	s.send(notifications.Payload{
		Title:   fmt.Sprintf("%s - %s", transport, status.State),
		Content: details,
	})
}

func (s *NotificationService) shouldNotify(prefs config.NotificationConfig, kindEnabled bool) bool {
	if !kindEnabled {
		return false
	}
	if prefs.NotifyWhenFocused {
		return true
	}
	if s.isForeground == nil {
		return true
	}

	return !s.isForeground()
}

func (s *NotificationService) notificationPrefs() config.NotificationConfig {
	cfg := config.Default()
	if s.currentConfig != nil {
		cfg = s.currentConfig()
		cfg.FillMissingDefaults()
	}

	return cfg.UI.Notifications
}

func (s *NotificationService) senderNameForMessage(msg domain.ChatMessage) string {
	meta, ok := parseMessageMeta(msg.MetaJSON)
	if ok {
		if nodeID := normalizeNotificationNodeID(meta.From); nodeID != "" {
			return domain.NodeDisplayNameByID(s.nodeStore, nodeID)
		}
	}
	if nodeID := normalizeNotificationNodeID(domain.NodeIDFromDMChatKey(msg.ChatKey)); nodeID != "" {
		return domain.NodeDisplayNameByID(s.nodeStore, nodeID)
	}

	return ""
}

func (s *NotificationService) chatTitle(chatKey string) string {
	return domain.ChatTitleByKey(s.chatStore, chatKey)
}

func (s *NotificationService) send(notification notifications.Payload) {
	title := strings.TrimSpace(notification.Title)
	content := strings.TrimSpace(notification.Content)
	if title == "" && content == "" {
		return
	}
	s.logger.Debug("sending notification", "title", title)
	s.sender.Send(notifications.Payload{
		Title:   title,
		Content: content,
	})
}

func parseMessageMeta(raw string) (messageMeta, bool) {
	if strings.TrimSpace(raw) == "" {
		return messageMeta{}, false
	}
	var meta messageMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return messageMeta{}, false
	}

	return meta, true
}

func chatTypeForNotification(chatKey string) domain.ChatType {
	return domain.ChatTypeForKey(chatKey)
}

func normalizeNotificationNodeID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || strings.EqualFold(value, "unknown") || strings.EqualFold(value, "!ffffffff") {
		return ""
	}

	return value
}

func nodeDiscoveredContent(event domain.NodeDiscovered) string {
	nodeID := strings.TrimSpace(event.NodeID)
	if nodeID == "" {
		nodeID = strings.TrimSpace(event.Node.NodeID)
	}
	if nodeID == "" {
		nodeID = "unknown"
	}

	longName := strings.TrimSpace(event.Node.LongName)
	if longName == "" {
		return nodeID
	}
	shortName := strings.TrimSpace(event.Node.ShortName)
	if shortName == "" {
		return longName
	}

	return fmt.Sprintf("[%s] %s", shortName, longName)
}

func notificationTransportName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "ip":
		return "IP"
	case "serial":
		return "Serial"
	case "bluetooth":
		return "Bluetooth LE (unstable)"
	default:
		return strings.TrimSpace(name)
	}
}
