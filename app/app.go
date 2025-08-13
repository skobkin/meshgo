package app

import (
	"context"
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
}

// App wires together the transport, radio client and persistence layers.
type App struct {
	Radio    Radio
	Messages *storage.MessageStore
	Nodes    *storage.NodeStore
	Chats    *storage.ChatStore
	Notifier notify.Notifier
	Tray     tray.Tray
}

// New creates an App using the provided radio client, message store and notifier.
func New(r Radio, ms *storage.MessageStore, ns *storage.NodeStore, cs *storage.ChatStore, n notify.Notifier, tr tray.Tray) *App {
	return &App{Radio: r, Messages: ms, Nodes: ns, Chats: cs, Notifier: n, Tray: tr}
}

// Run starts the radio client with the given transport and processes events
// until the context is cancelled.
func (a *App) Run(ctx context.Context, t transport.Transport) error {
	go a.eventLoop(ctx)
	return a.Radio.Start(ctx, t)
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

func (a *App) eventLoop(ctx context.Context) {
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
		if a.Chats != nil {
			_ = a.Chats.UpsertChat(ctx, &domain.Chat{ID: m.ChatID, Title: m.ChatID, LastMessageTS: m.Timestamp.Unix()})
		}
		if a.Notifier != nil {
			_ = a.Notifier.NotifyNewMessage(m.ChatID, "New Message", m.Text, m.Timestamp)
		}
		a.RefreshUnread(ctx)
	}
}

func (a *App) handleNode(ctx context.Context, n *domain.Node) {
	if a.Nodes == nil || n == nil {
		return
	}
	n.Signal = domain.ComputeSignalQuality(n.RSSI, n.SNR)
	_ = a.Nodes.UpsertNode(ctx, n)
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
