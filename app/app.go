package app

import (
	"context"
	"log/slog"
	"time"

	"meshgo/domain"
	"meshgo/notify"
	"meshgo/radio"
	"meshgo/storage"
	"meshgo/transport"
	"meshgo/tray"
)

// Radio abstracts the radio client used by the application.
type Radio interface {
	Start(ctx context.Context, t transport.Transport) error
	Events() <-chan radio.Event
	SendText(ctx context.Context, chatID string, toNode uint32, text string) error
	SendExchangeUserInfo(ctx context.Context, node uint32) error
	SendTraceroute(ctx context.Context, node uint32) error
}

// App wires together the transport, radio client and persistence layers.
type App struct {
	Radio    Radio
	Messages *storage.MessageStore
	Nodes    *storage.NodeStore
	Chats    *storage.ChatStore
	Channels *storage.ChannelStore
	Notifier notify.Notifier
	Tray     tray.Tray
	events   chan Event
}

// New creates an App using the provided radio client, stores and notifier.
func New(r Radio, ms *storage.MessageStore, ns *storage.NodeStore, cs *storage.ChatStore, chs *storage.ChannelStore, n notify.Notifier, tr tray.Tray) *App {
	return &App{Radio: r, Messages: ms, Nodes: ns, Chats: cs, Channels: chs, Notifier: n, Tray: tr, events: make(chan Event, 16)}
}

// Run starts the radio client with the given transport and processes events
// until the context is cancelled.
func (a *App) Run(ctx context.Context, t transport.Transport) error {
	slog.Info("app starting", "endpoint", t.Endpoint())
	go a.eventLoop(ctx)
	err := a.Radio.Start(ctx, t)
	if err != nil {
		slog.Error("radio stopped", "err", err)
	}
	return err
}

// SendText sends a text message via the radio client and persists it to the store.
func (a *App) SendText(ctx context.Context, chatID string, toNode uint32, text string) error {
	if err := a.Radio.SendText(ctx, chatID, toNode, text); err != nil {
		return err
	}
	if a.Messages != nil {
		m := &domain.Message{
			ChatID:    chatID,
			Text:      text,
			Timestamp: time.Now(),
		}
		if err := a.Messages.InsertMessage(ctx, m); err != nil {
			return err
		}
		if a.Chats != nil {
			_ = a.Chats.UpsertChat(ctx, &domain.Chat{ID: chatID, Title: chatID, LastMessageTS: m.Timestamp.Unix()})
		}
	}
	a.RefreshUnread(ctx)
	return nil
}

// SendExchangeUserInfo requests user information from the specified node.
func (a *App) SendExchangeUserInfo(ctx context.Context, node uint32) error {
	if a.Radio == nil {
		return nil
	}
	return a.Radio.SendExchangeUserInfo(ctx, node)
}

// SendTraceroute requests a traceroute to the specified node.
func (a *App) SendTraceroute(ctx context.Context, node uint32) error {
	if a.Radio == nil {
		return nil
	}
	return a.Radio.SendTraceroute(ctx, node)
}

func (a *App) eventLoop(ctx context.Context) {
	defer close(a.events)
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-a.Radio.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case radio.EventPacket:
				a.handlePacket(ctx, ev.Packet)
			case radio.EventNode:
				a.handleNode(ctx, ev.Node)
			case radio.EventConnecting:
				a.events <- Event{Type: EventConnecting}
			case radio.EventConnected:
				a.events <- Event{Type: EventConnected}
			case radio.EventDisconnected:
				a.events <- Event{Type: EventDisconnected, Err: ev.Err}
			case radio.EventRetrying:
				a.events <- Event{Type: EventRetrying, Delay: ev.Delay}
			}
		}
	}
}

func (a *App) handlePacket(ctx context.Context, pkt []byte) {
	if a.Messages == nil {
		return
	}
	m := &domain.Message{
		ChatID:    "default",
		Text:      string(pkt),
		Timestamp: time.Now(),
		IsUnread:  true,
	}
	if err := a.Messages.InsertMessage(ctx, m); err == nil {
		slog.Info("packet received", "chat", m.ChatID, "text", m.Text)
		if a.Chats != nil {
			_ = a.Chats.UpsertChat(ctx, &domain.Chat{ID: m.ChatID, Title: m.ChatID, LastMessageTS: m.Timestamp.Unix()})
		}
		if a.Notifier != nil {
			_ = a.Notifier.NotifyNewMessage(m.ChatID, "New Message", m.Text, m.Timestamp)
		}
		a.RefreshUnread(ctx)
		a.events <- Event{Type: EventMessage, Message: m}
	}
}

func (a *App) handleNode(ctx context.Context, n *domain.Node) {
	if a.Nodes == nil || n == nil {
		return
	}
	n.Signal = domain.ComputeSignalQuality(n.RSSI, n.SNR)
	_ = a.Nodes.UpsertNode(ctx, n)
	slog.Info("node event", "id", n.ID, "short", n.ShortName, "signal", n.Signal)
	a.events <- Event{Type: EventNode, Node: n}
}

// RefreshUnread updates the tray unread state based on the message store.
func (a *App) RefreshUnread(ctx context.Context) {
	if a.Messages == nil || a.Tray == nil {
		return
	}
	if count, err := a.Messages.UnreadCount(ctx); err == nil {
		a.Tray.SetUnread(count > 0)
	}
}

// MarkChatRead marks all messages in the specified chat as read up to now and
// refreshes the unread indicator.
func (a *App) MarkChatRead(ctx context.Context, chatID string) error {
	if a.Messages == nil {
		return nil
	}
	if err := a.Messages.SetRead(ctx, chatID, time.Now()); err != nil {
		return err
	}
	a.RefreshUnread(ctx)
	return nil
}

// ListMessages returns recent messages for the chat.
func (a *App) ListMessages(ctx context.Context, chatID string, limit int) ([]*domain.Message, error) {
	if a.Messages == nil {
		return nil, nil
	}
	return a.Messages.ListMessages(ctx, chatID, limit)
}

// ListChats returns all chats from the store.
func (a *App) ListChats(ctx context.Context) ([]*domain.Chat, error) {
	if a.Chats == nil {
		return nil, nil
	}
	chats, err := a.Chats.ListChats(ctx)
	if err != nil {
		return nil, err
	}
	if a.Messages != nil {
		if counts, err := a.Messages.UnreadCountByChat(ctx); err == nil {
			for _, c := range chats {
				c.UnreadCount = counts[c.ID]
			}
		}
	}
	return chats, nil
}

// ListChannels returns all channels from the store.
func (a *App) ListChannels(ctx context.Context) ([]*domain.Channel, error) {
	if a.Channels == nil {
		return nil, nil
	}
	return a.Channels.ListChannels(ctx)
}

// ListNodes returns all nodes from the store.
func (a *App) ListNodes(ctx context.Context) ([]*domain.Node, error) {
	if a.Nodes == nil {
		return nil, nil
	}
	return a.Nodes.ListNodes(ctx)
}

// SetNodeFavorite marks a node as favorite or not.
func (a *App) SetNodeFavorite(ctx context.Context, id string, fav bool) error {
	if a.Nodes == nil {
		return nil
	}
	return a.Nodes.SetFavorite(ctx, id, fav)
}

// SetNodeIgnored marks a node as ignored or not.
func (a *App) SetNodeIgnored(ctx context.Context, id string, ignored bool) error {
	if a.Nodes == nil {
		return nil
	}
	return a.Nodes.SetIgnored(ctx, id, ignored)
}

// RemoveNode deletes the node with the given ID from the store.
func (a *App) RemoveNode(ctx context.Context, id string) error {
	if a.Nodes == nil {
		return nil
	}
	return a.Nodes.RemoveNode(ctx, id)
}
