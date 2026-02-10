package domain

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

type ChatStore struct {
	mu       sync.RWMutex
	chats    map[string]Chat
	messages map[string][]ChatMessage
	changes  chan struct{}
}

func NewChatStore() *ChatStore {
	return &ChatStore{
		chats:    make(map[string]Chat),
		messages: make(map[string][]ChatMessage),
		changes:  make(chan struct{}, 1),
	}
}

func (s *ChatStore) Load(chats []Chat, messages map[string][]ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, chat := range chats {
		s.chats[chat.Key] = chat
	}
	for key, msgs := range messages {
		cloned := make([]ChatMessage, len(msgs))
		copy(cloned, msgs)
		s.messages[key] = cloned
	}
	s.notify()
}

func (s *ChatStore) Start(ctx context.Context, b bus.MessageBus) {
	textSub := b.Subscribe(connectors.TopicTextMessage)
	statusSub := b.Subscribe(connectors.TopicMessageStatus)
	channelsSub := b.Subscribe(connectors.TopicChannels)

	go func() {
		defer b.Unsubscribe(textSub, connectors.TopicTextMessage)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-textSub:
				if !ok {
					return
				}
				chatMsg, ok := msg.(ChatMessage)
				if !ok {
					continue
				}
				s.AppendMessage(chatMsg)
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(statusSub, connectors.TopicMessageStatus)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-statusSub:
				if !ok {
					return
				}
				update, ok := msg.(MessageStatusUpdate)
				if !ok {
					continue
				}
				s.UpdateMessageStatusByDeviceID(update.DeviceMessageID, update.Status)
			}
		}
	}()

	go func() {
		defer b.Unsubscribe(channelsSub, connectors.TopicChannels)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-channelsSub:
				if !ok {
					return
				}
				channels, ok := msg.(ChannelList)
				if !ok {
					continue
				}
				now := time.Now()
				for _, ch := range channels.Items {
					key := ChatKeyForChannel(ch.Index)
					title := strings.TrimSpace(ch.Title)
					if title == "" {
						title = key
					}
					s.UpsertChat(Chat{Key: key, Title: title, Type: ChatTypeChannel, UpdatedAt: now})
				}
			}
		}
	}()
}

func (s *ChatStore) UpsertChat(chat Chat) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.chats[chat.Key]
	if ok {
		if !chat.LastSentByMeAt.After(existing.LastSentByMeAt) {
			chat.LastSentByMeAt = existing.LastSentByMeAt
		}
		if existing.UpdatedAt.After(chat.UpdatedAt) {
			chat.UpdatedAt = existing.UpdatedAt
		}
	}
	if chat.UpdatedAt.IsZero() {
		chat.UpdatedAt = time.Now()
	}
	s.chats[chat.Key] = chat
	s.notify()
}

func (s *ChatStore) AppendMessage(msg ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg.At.IsZero() {
		msg.At = time.Now()
	}
	if msg.DeviceMessageID != "" {
		msgs := s.messages[msg.ChatKey]
		for i := range msgs {
			if msgs[i].DeviceMessageID != msg.DeviceMessageID {
				continue
			}
			changed := false
			if msgs[i].Body == "" && msg.Body != "" {
				msgs[i].Body = msg.Body
				changed = true
			}
			if msgs[i].MetaJSON == "" && msg.MetaJSON != "" {
				msgs[i].MetaJSON = msg.MetaJSON
				changed = true
			}
			if !msg.At.IsZero() && msgs[i].At.Before(msg.At) {
				msgs[i].At = msg.At
				changed = true
			}
			if ShouldTransitionMessageStatus(msgs[i].Status, msg.Status) {
				msgs[i].Status = msg.Status
				changed = true
			}
			if changed {
				s.messages[msg.ChatKey] = msgs
				chat, ok := s.chats[msg.ChatKey]
				if ok && msg.At.After(chat.UpdatedAt) {
					chat.UpdatedAt = msg.At
					s.chats[msg.ChatKey] = chat
				}
				s.notify()
			}
			return
		}
	}
	s.messages[msg.ChatKey] = append(s.messages[msg.ChatKey], msg)

	chat, ok := s.chats[msg.ChatKey]
	if !ok {
		chat = Chat{Key: msg.ChatKey, Type: chatTypeForKey(msg.ChatKey), Title: msg.ChatKey}
	}
	if msg.Direction == MessageDirectionOut {
		chat.LastSentByMeAt = msg.At
	}
	chat.UpdatedAt = msg.At
	s.chats[msg.ChatKey] = chat
	s.notify()
}

func (s *ChatStore) UpdateMessageStatusByDeviceID(deviceMessageID string, status MessageStatus) {
	deviceMessageID = strings.TrimSpace(deviceMessageID)
	if deviceMessageID == "" || status == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	for chatKey, messages := range s.messages {
		for i := range messages {
			if messages[i].DeviceMessageID != deviceMessageID {
				continue
			}
			if !ShouldTransitionMessageStatus(messages[i].Status, status) {
				continue
			}
			messages[i].Status = status
			changed = true
		}
		s.messages[chatKey] = messages
	}
	if changed {
		s.notify()
	}
}

func (s *ChatStore) ChatListSorted() []Chat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Chat, 0, len(s.chats))
	for _, chat := range s.chats {
		out = append(out, chat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastSentByMeAt.Equal(out[j].LastSentByMeAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].LastSentByMeAt.After(out[j].LastSentByMeAt)
	})
	return out
}

func (s *ChatStore) Messages(chatKey string) []ChatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[chatKey]
	cloned := make([]ChatMessage, len(msgs))
	copy(cloned, msgs)
	sort.Slice(cloned, func(i, j int) bool {
		return cloned[i].At.Before(cloned[j].At)
	})
	return cloned
}

func (s *ChatStore) Changes() <-chan struct{} {
	return s.changes
}

func (s *ChatStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chats = make(map[string]Chat)
	s.messages = make(map[string][]ChatMessage)
	s.notify()
}

func (s *ChatStore) notify() {
	select {
	case s.changes <- struct{}{}:
	default:
	}
}

func chatTypeForKey(key string) ChatType {
	if strings.HasPrefix(key, "dm:") {
		return ChatTypeDM
	}
	return ChatTypeChannel
}
