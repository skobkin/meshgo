package bus

import (
	"log/slog"
	"reflect"

	"github.com/cskr/pubsub"
)

type Subscription chan any

type MessageBus interface {
	Publish(topic string, msg any)
	Subscribe(topic string) Subscription
	Unsubscribe(ch Subscription, topics ...string)
	Close()
}

type PubSubBus struct {
	ps     *pubsub.PubSub
	logger *slog.Logger
}

func New(logger *slog.Logger) *PubSubBus {
	return &PubSubBus{
		ps:     pubsub.New(128),
		logger: logger,
	}
}

func (b *PubSubBus) Publish(topic string, msg any) {
	b.logger.Debug("publish", "topic", topic, "payload_type", payloadType(msg))
	b.ps.Pub(msg, topic)
}

func (b *PubSubBus) Subscribe(topic string) Subscription {
	ch := b.ps.Sub(topic)
	b.logger.Debug("subscribe", "topic", topic)
	return ch
}

func (b *PubSubBus) Unsubscribe(ch Subscription, topics ...string) {
	if len(topics) == 0 {
		b.ps.Unsub(ch)
		b.logger.Debug("unsubscribe", "mode", "all")
		return
	}
	b.ps.Unsub(ch, topics...)
	b.logger.Debug("unsubscribe", "topics", topics)
}

func (b *PubSubBus) Close() {
	b.ps.Shutdown()
}

func payloadType(v any) string {
	if v == nil {
		return "<nil>"
	}
	return reflect.TypeOf(v).String()
}
