package app

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/notifications"
)

func TestNotificationServiceIncomingDMMessage(t *testing.T) {
	messageBus := newTestMessageBus(t)
	chatStore := domain.NewChatStore()
	nodeStore := domain.NewNodeStore()
	nodeStore.Upsert(domain.Node{
		NodeID:    "!12345678",
		LongName:  "Alice",
		ShortName: "ALC",
	})
	cfg := config.Default()
	foreground := false
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		chatStore,
		nodeStore,
		func() config.AppConfig { return cfg },
		func() bool { return foreground },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicTextMessage, domain.ChatMessage{
		ChatKey:         domain.ChatKeyForDM("!12345678"),
		Direction:       domain.MessageDirectionIn,
		Body:            "Hello there",
		MetaJSON:        `{"from":"!12345678"}`,
		DeviceMessageID: "1",
	})

	gotNotifications := sender.waitForCount(t, 1)
	if got := gotNotifications[0].Title; got != "@Alice" {
		t.Fatalf("expected title @Alice, got %q", got)
	}
	if got := gotNotifications[0].Content; got != "Alice: Hello there" {
		t.Fatalf("expected content %q, got %q", "Alice: Hello there", got)
	}
}

func TestNotificationServiceIncomingChannelMessage(t *testing.T) {
	messageBus := newTestMessageBus(t)
	chatStore := domain.NewChatStore()
	chatStore.UpsertChat(domain.Chat{
		Key:       domain.ChatKeyForChannel(0),
		Title:     "General",
		Type:      domain.ChatTypeChannel,
		UpdatedAt: time.Now(),
	})
	nodeStore := domain.NewNodeStore()
	nodeStore.Upsert(domain.Node{
		NodeID:    "!87654321",
		ShortName: "B0B",
	})
	cfg := config.Default()
	foreground := false
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		chatStore,
		nodeStore,
		func() config.AppConfig { return cfg },
		func() bool { return foreground },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicTextMessage, domain.ChatMessage{
		ChatKey:   domain.ChatKeyForChannel(0),
		Direction: domain.MessageDirectionIn,
		Body:      "Hi channel",
		MetaJSON:  `{"from":"!87654321"}`,
	})

	gotNotifications := sender.waitForCount(t, 1)
	if got := gotNotifications[0].Title; got != "#General" {
		t.Fatalf("expected title #General, got %q", got)
	}
	if got := gotNotifications[0].Content; got != "B0B: Hi channel" {
		t.Fatalf("expected content %q, got %q", "B0B: Hi channel", got)
	}
}

func TestNotificationServiceSkipsOutgoingMessages(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig { return cfg },
		func() bool { return false },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicTextMessage, domain.ChatMessage{
		ChatKey:   domain.ChatKeyForChannel(0),
		Direction: domain.MessageDirectionOut,
		Body:      "outgoing",
	})

	sender.assertCount(t, 0)
}

func TestNotificationServiceNodeDiscoveredFormatting(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig { return cfg },
		func() bool { return false },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicNodeDiscovered, domain.NodeDiscovered{
		NodeID: "!00000001",
		Node: domain.Node{
			NodeID:    "!00000001",
			ShortName: "ABCD",
			LongName:  "Alpha Node",
		},
	})
	messageBus.Publish(connectors.TopicNodeDiscovered, domain.NodeDiscovered{
		NodeID: "!00000002",
		Node: domain.Node{
			NodeID: "!00000002",
		},
	})

	gotNotifications := sender.waitForCount(t, 2)
	if got := gotNotifications[0].Title; got != notificationTitleNodeDiscovered {
		t.Fatalf("expected title %q, got %q", notificationTitleNodeDiscovered, got)
	}
	if got := gotNotifications[0].Content; got != "[ABCD] Alpha Node" {
		t.Fatalf("expected content %q, got %q", "[ABCD] Alpha Node", got)
	}
	if got := gotNotifications[1].Content; got != "!00000002" {
		t.Fatalf("expected node id fallback, got %q", got)
	}
}

func TestNotificationServiceConnectionStatusFilteringAndFormatting(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig { return cfg },
		func() bool { return false },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
		Target:        "192.168.0.156:4403",
	})
	gotNotifications := sender.waitForCount(t, 1)
	if got := gotNotifications[0].Title; got != "IP - connected" {
		t.Fatalf("expected connected title, got %q", got)
	}

	// Duplicate consecutive state must be ignored.
	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
		Target:        "192.168.0.156:4403",
	})
	sender.assertCount(t, 1)

	// Reconnecting itself should not notify.
	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateReconnecting,
		TransportName: "ip",
		Target:        "192.168.0.156:4403",
	})
	sender.assertCount(t, 1)

	// Connected again after a different state should notify.
	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
		Target:        "192.168.0.156:4403",
	})
	gotNotifications = sender.waitForCount(t, 2)
	if got := gotNotifications[1].Title; got != "IP - connected" {
		t.Fatalf("expected reconnection title, got %q", got)
	}

	messageBus.Publish(connectors.TopicConnStatus, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: "serial",
		Target:        "/dev/ttyACM0",
		Err:           "read timeout",
	})
	gotNotifications = sender.waitForCount(t, 3)
	if got := gotNotifications[2].Title; got != "Serial - disconnected" {
		t.Fatalf("expected disconnected title, got %q", got)
	}
	if got := gotNotifications[2].Content; got != "/dev/ttyACM0 (error: read timeout)" {
		t.Fatalf("expected disconnected content with error, got %q", got)
	}
}

func TestNotificationServiceForegroundAndPerTypeSettings(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	var cfgMu sync.RWMutex
	foreground := true
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig {
			cfgMu.RLock()
			defer cfgMu.RUnlock()

			return cfg
		},
		func() bool { return foreground },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	message := domain.ChatMessage{
		ChatKey:   domain.ChatKeyForDM("!12345678"),
		Direction: domain.MessageDirectionIn,
		Body:      "hello",
		MetaJSON:  `{"from":"!12345678"}`,
	}

	// Focused app + notify_when_focused=false -> suppressed.
	messageBus.Publish(connectors.TopicTextMessage, message)
	sender.assertCount(t, 0)

	cfgMu.Lock()
	cfg.UI.Notifications.NotifyWhenFocused = true
	cfgMu.Unlock()
	messageBus.Publish(connectors.TopicTextMessage, message)
	sender.waitForCount(t, 1)

	cfgMu.Lock()
	cfg.UI.Notifications.Events.IncomingMessage = false
	cfgMu.Unlock()
	messageBus.Publish(connectors.TopicTextMessage, message)
	sender.assertCount(t, 1)
}

func TestNotificationServiceUpdateAvailableOnLaterSnapshot(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig { return cfg },
		func() bool { return false },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: false,
		Latest: ReleaseInfo{
			Version: "1.1.0",
		},
	})
	sender.assertCount(t, 0)

	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.1.0",
		},
	})
	gotNotifications := sender.waitForCount(t, 1)
	if got := gotNotifications[0].Title; got != "Update available: 1.1.0" {
		t.Fatalf("expected update title, got %q", got)
	}
}

func TestNotificationServiceUpdateAvailableDedupesAndNotifiesNewer(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig { return cfg },
		func() bool { return false },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.1.0",
		},
	})
	sender.waitForCount(t, 1)

	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.1.0",
		},
	})
	sender.assertCount(t, 1)

	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.2.0",
		},
	})
	gotNotifications := sender.waitForCount(t, 2)
	if got := gotNotifications[1].Title; got != "Update available: 1.2.0" {
		t.Fatalf("expected newer update notification, got %q", got)
	}
}

func TestNotificationServiceUpdateAvailableRespectsSettingsAndForeground(t *testing.T) {
	messageBus := newTestMessageBus(t)
	cfg := config.Default()
	var cfgMu sync.RWMutex
	foreground := true
	sender := newCollectingNotificationSender()
	service := NewNotificationService(
		messageBus,
		domain.NewChatStore(),
		domain.NewNodeStore(),
		func() config.AppConfig {
			cfgMu.RLock()
			defer cfgMu.RUnlock()

			return cfg
		},
		func() bool { return foreground },
		sender,
		nil,
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service.Start(ctx)

	snapshot := UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.1.0",
		},
	}
	messageBus.Publish(connectors.TopicUpdateSnapshot, snapshot)
	sender.assertCount(t, 0)

	cfgMu.Lock()
	cfg.UI.Notifications.NotifyWhenFocused = true
	cfg.UI.Notifications.Events.UpdateAvailable = false
	cfgMu.Unlock()
	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.2.0",
		},
	})
	sender.assertCount(t, 0)

	cfgMu.Lock()
	cfg.UI.Notifications.Events.UpdateAvailable = true
	cfgMu.Unlock()
	messageBus.Publish(connectors.TopicUpdateSnapshot, UpdateSnapshot{
		CurrentVersion:  "1.0.0",
		UpdateAvailable: true,
		Latest: ReleaseInfo{
			Version: "1.3.0",
		},
	})
	gotNotifications := sender.waitForCount(t, 1)
	if got := gotNotifications[0].Title; got != "Update available: 1.3.0" {
		t.Fatalf("expected update title after enabling, got %q", got)
	}
}

func newTestMessageBus(t *testing.T) *bus.PubSubBus {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	t.Cleanup(func() {
		messageBus.Close()
	})

	return messageBus
}

type collectingNotificationSender struct {
	mu            sync.Mutex
	notifications []notifications.Payload
	changes       chan struct{}
}

func newCollectingNotificationSender() *collectingNotificationSender {
	return &collectingNotificationSender{
		changes: make(chan struct{}, 1),
	}
}

func (s *collectingNotificationSender) Send(notification notifications.Payload) {
	s.mu.Lock()
	s.notifications = append(s.notifications, notification)
	s.mu.Unlock()

	select {
	case s.changes <- struct{}{}:
	default:
	}
}

func (s *collectingNotificationSender) snapshot() []notifications.Payload {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]notifications.Payload, len(s.notifications))
	copy(out, s.notifications)

	return out
}

func (s *collectingNotificationSender) waitForCount(t *testing.T, expected int) []notifications.Payload {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current := s.snapshot()
		if len(current) >= expected {
			return current
		}
		select {
		case <-s.changes:
		case <-time.After(10 * time.Millisecond):
		}
	}

	t.Fatalf("timed out waiting for %d notifications", expected)

	return nil
}

func (s *collectingNotificationSender) assertCount(t *testing.T, expected int) {
	t.Helper()

	time.Sleep(100 * time.Millisecond)
	current := s.snapshot()
	if len(current) != expected {
		t.Fatalf("expected %d notifications, got %d", expected, len(current))
	}
}
