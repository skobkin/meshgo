package projections

import (
	"context"
	"strconv"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

// WriteQueue serializes persistence writes from async domain events.
type WriteQueue interface {
	Enqueue(name string, fn func(context.Context) error)
}

// HistoryLimitsProvider returns current node history caps.
type HistoryLimitsProvider interface {
	PositionHistoryLimit() int
	TelemetryHistoryLimit() int
	IdentityHistoryLimit() int
}

func StartPersistenceProjection(
	ctx context.Context,
	b bus.MessageBus,
	queue WriteQueue,
	coreRepo domain.NodeCoreRepository,
	positionRepo domain.NodePositionRepository,
	telemetryRepo domain.NodeTelemetryRepository,
	historyLimits HistoryLimitsProvider,
	chatRepo domain.ChatRepository,
	msgRepo domain.MessageRepository,
	tracerouteRepo domain.TracerouteRepository,
) {
	coreSub := b.Subscribe(bus.TopicNodeCore)
	positionSub := b.Subscribe(bus.TopicNodePosition)
	telemetrySub := b.Subscribe(bus.TopicNodeTelemetry)
	channelSub := b.Subscribe(bus.TopicChannels)
	textSub := b.Subscribe(bus.TopicTextMessage)
	statusSub := b.Subscribe(bus.TopicMessageStatus)
	var tracerouteSub bus.Subscription
	if tracerouteRepo != nil {
		tracerouteSub = b.Subscribe(bus.TopicTracerouteUpdate)
	}

	go func() {
		defer b.Unsubscribe(coreSub, bus.TopicNodeCore)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-coreSub:
				if !ok {
					return
				}
				update, ok := raw.(domain.NodeCoreUpdate)
				if !ok {
					continue
				}
				copyUpdate := update
				queue.Enqueue("upsert_node_core", func(writeCtx context.Context) error {
					limit := 0
					if historyLimits != nil {
						limit = historyLimits.IdentityHistoryLimit()
					}

					return coreRepo.Upsert(writeCtx, copyUpdate, limit)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(positionSub, bus.TopicNodePosition)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-positionSub:
				if !ok {
					return
				}
				update, ok := raw.(domain.NodePositionUpdate)
				if !ok {
					continue
				}
				copyUpdate := update
				queue.Enqueue("upsert_node_position", func(writeCtx context.Context) error {
					limit := 0
					if historyLimits != nil {
						limit = historyLimits.PositionHistoryLimit()
					}

					return positionRepo.Upsert(writeCtx, copyUpdate, limit)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(telemetrySub, bus.TopicNodeTelemetry)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-telemetrySub:
				if !ok {
					return
				}
				update, ok := raw.(domain.NodeTelemetryUpdate)
				if !ok {
					continue
				}
				copyUpdate := update
				queue.Enqueue("upsert_node_telemetry", func(writeCtx context.Context) error {
					limit := 0
					if historyLimits != nil {
						limit = historyLimits.TelemetryHistoryLimit()
					}

					return telemetryRepo.Upsert(writeCtx, copyUpdate, limit)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(channelSub, bus.TopicChannels)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-channelSub:
				if !ok {
					return
				}
				channels, ok := raw.(domain.ChannelList)
				if !ok {
					continue
				}
				for _, ch := range channels.Items {
					chat := domain.Chat{
						Key:   domain.ChatKeyForChannel(ch.Index),
						Title: ch.Title,
						Type:  domain.ChatTypeChannel,
					}
					queue.Enqueue("upsert_channel_chat", func(writeCtx context.Context) error {
						return chatRepo.Upsert(writeCtx, chat)
					})
				}
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(textSub, bus.TopicTextMessage)
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
				copyMsg := msg
				queue.Enqueue("insert_message", func(writeCtx context.Context) error {
					_, err := msgRepo.Insert(writeCtx, copyMsg)
					if err != nil {
						return err
					}
					chat := domain.Chat{
						Key:       copyMsg.ChatKey,
						Type:      domain.ChatTypeForKey(copyMsg.ChatKey),
						Title:     copyMsg.ChatKey,
						UpdatedAt: copyMsg.At,
					}
					if copyMsg.Direction == domain.MessageDirectionOut {
						chat.LastSentByMeAt = copyMsg.At
					}

					return chatRepo.Upsert(writeCtx, chat)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(statusSub, bus.TopicMessageStatus)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-statusSub:
				if !ok {
					return
				}
				update, ok := raw.(domain.MessageStatusUpdate)
				if !ok {
					continue
				}
				copyUpdate := update
				queue.Enqueue("update_message_status", func(writeCtx context.Context) error {
					return msgRepo.UpdateStatusByDeviceMessageID(writeCtx, copyUpdate.DeviceMessageID, copyUpdate.Status)
				})
			}
		}
	}()

	if tracerouteRepo != nil {
		go func() {
			defer b.Unsubscribe(tracerouteSub, bus.TopicTracerouteUpdate)
			for {
				select {
				case <-ctx.Done():
					return
				case raw, ok := <-tracerouteSub:
					if !ok {
						return
					}
					update, ok := raw.(busmsg.TracerouteUpdate)
					if !ok {
						continue
					}
					rec := domain.TracerouteRecord{
						RequestID:    stringFromUint32(update.RequestID),
						TargetNodeID: update.TargetNodeID,
						StartedAt:    update.StartedAt,
						UpdatedAt:    update.UpdatedAt,
						CompletedAt:  update.CompletedAt,
						Status:       update.Status,
						ForwardRoute: append([]string(nil), update.ForwardRoute...),
						ForwardSNR:   append([]int32(nil), update.ForwardSNR...),
						ReturnRoute:  append([]string(nil), update.ReturnRoute...),
						ReturnSNR:    append([]int32(nil), update.ReturnSNR...),
						ErrorText:    update.Error,
						DurationMS:   update.DurationMS,
					}
					queue.Enqueue("upsert_traceroute", func(writeCtx context.Context) error {
						return tracerouteRepo.Upsert(writeCtx, rec)
					})
				}
			}
		}()
	}
}

func stringFromUint32(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
