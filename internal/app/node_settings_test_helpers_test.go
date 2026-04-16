package app

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type stubAdminSender struct {
	send func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error)
}

func (s stubAdminSender) SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
	return s.send(to, channel, wantResponse, payload)
}

func newTestNodeSettingsService(t *testing.T, sender stubAdminSender, connected bool) (*NodeSettingsService, bus.MessageBus) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	t.Cleanup(func() {
		messageBus.Close()
	})

	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (busmsg.ConnectionStatus, bool) {
			if connected {
				return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
			}

			return busmsg.ConnectionStatus{}, false
		},
		logger,
	)

	return service, messageBus
}

func publishAdminReply(messageBus bus.MessageBus, to, replyID uint32, message *generated.AdminMessage) {
	messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
		From:      to,
		RequestID: 777,
		ReplyID:   replyID,
		Message:   message,
	})
}

func publishSentStatus(messageBus bus.MessageBus, packetID uint32) {
	messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
		DeviceMessageID: stringFromUint32(packetID),
		Status:          domain.MessageStatusSent,
	})
}

func stringFromUint32(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

func mustHexNodeTarget() NodeSettingsTarget {
	return NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true}
}

func mustLocalNodeTarget() NodeSettingsTarget {
	return NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}
}

func contextWithTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	return context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
}
