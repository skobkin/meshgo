package domain

import (
	"context"
	"strconv"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

// WriteQueue serializes persistence writes from async domain events.
type WriteQueue interface {
	Enqueue(name string, fn func(context.Context) error)
}

func StartPersistenceProjection(
	ctx context.Context,
	b bus.MessageBus,
	queue WriteQueue,
	nodeRepo NodeRepository,
	chatRepo ChatRepository,
	msgRepo MessageRepository,
	tracerouteRepo TracerouteRepository,
) {
	nodeSub := b.Subscribe(connectors.TopicNodeInfo)
	channelSub := b.Subscribe(connectors.TopicChannels)
	textSub := b.Subscribe(connectors.TopicTextMessage)
	statusSub := b.Subscribe(connectors.TopicMessageStatus)
	var tracerouteSub bus.Subscription
	if tracerouteRepo != nil {
		tracerouteSub = b.Subscribe(connectors.TopicTracerouteUpdate)
	}

	go func() {
		defer b.Unsubscribe(nodeSub, connectors.TopicNodeInfo)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-nodeSub:
				if !ok {
					return
				}
				update, ok := raw.(NodeUpdate)
				if !ok {
					continue
				}
				n := update.Node
				queue.Enqueue("upsert_node", func(writeCtx context.Context) error {
					return nodeRepo.Upsert(writeCtx, n)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(channelSub, connectors.TopicChannels)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-channelSub:
				if !ok {
					return
				}
				channels, ok := raw.(ChannelList)
				if !ok {
					continue
				}
				for _, ch := range channels.Items {
					chat := Chat{Key: ChatKeyForChannel(ch.Index), Title: ch.Title, Type: ChatTypeChannel}
					queue.Enqueue("upsert_channel_chat", func(writeCtx context.Context) error {
						return chatRepo.Upsert(writeCtx, chat)
					})
				}
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(textSub, connectors.TopicTextMessage)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-textSub:
				if !ok {
					return
				}
				msg, ok := raw.(ChatMessage)
				if !ok {
					continue
				}
				copyMsg := msg
				queue.Enqueue("insert_message", func(writeCtx context.Context) error {
					_, err := msgRepo.Insert(writeCtx, copyMsg)
					if err != nil {
						return err
					}
					chat := Chat{Key: copyMsg.ChatKey, Type: ChatTypeForKey(copyMsg.ChatKey), Title: copyMsg.ChatKey, UpdatedAt: copyMsg.At}
					if copyMsg.Direction == MessageDirectionOut {
						chat.LastSentByMeAt = copyMsg.At
					}

					return chatRepo.Upsert(writeCtx, chat)
				})
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(statusSub, connectors.TopicMessageStatus)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-statusSub:
				if !ok {
					return
				}
				update, ok := raw.(MessageStatusUpdate)
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
			defer b.Unsubscribe(tracerouteSub, connectors.TopicTracerouteUpdate)
			for {
				select {
				case <-ctx.Done():
					return
				case raw, ok := <-tracerouteSub:
					if !ok {
						return
					}
					update, ok := raw.(connectors.TracerouteUpdate)
					if !ok {
						continue
					}
					rec := TracerouteRecord{
						RequestID:    stringFromUint32(update.RequestID),
						TargetNodeID: update.TargetNodeID,
						StartedAt:    update.StartedAt,
						UpdatedAt:    update.UpdatedAt,
						CompletedAt:  update.CompletedAt,
						Status:       TracerouteStatus(update.Status),
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
