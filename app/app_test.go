package app

import (
	"context"
	"testing"
	"time"

	"meshgo/domain"
	"meshgo/radio"
	"meshgo/storage"
	"meshgo/transport"
)

type stubRadio struct {
	events chan radio.Event
	sent   []string
}

func newStubRadio() *stubRadio {
	return &stubRadio{events: make(chan radio.Event, 1)}
}

func (s *stubRadio) Start(ctx context.Context, t transport.Transport) error {
	<-ctx.Done()
	return ctx.Err()
}

func (s *stubRadio) Events() <-chan radio.Event { return s.events }
func (s *stubRadio) SendText(ctx context.Context, chatID string, toNode uint32, text string) error {
	s.sent = append(s.sent, text)
	return nil
}
func (s *stubRadio) SendExchangeUserInfo(ctx context.Context, node uint32) error {
	s.sent = append(s.sent, "userinfo")
	return nil
}
func (s *stubRadio) SendTraceroute(ctx context.Context, node uint32) error {
	s.sent = append(s.sent, "traceroute")
	return nil
}

type stubNotifier struct {
	count   int
	enabled bool
}

func (n *stubNotifier) NotifyNewMessage(chatID, title, body string, ts time.Time) error {
	if n.enabled {
		n.count++
	}
	return nil
}

func (n *stubNotifier) SetEnabled(e bool) { n.enabled = e }

type stubTray struct{ unread bool }

func (t *stubTray) SetUnread(u bool)                    { t.unread = u }
func (t *stubTray) OnShowHide(fn func())                {}
func (t *stubTray) OnToggleNotifications(fn func(bool)) {}
func (t *stubTray) OnExit(fn func())                    {}
func (t *stubTray) Run()                                {}

func TestAppStoresPacketsAndNotifies(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ms, err := storage.OpenMessageStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatal(err)
	}

	cs, err := storage.OpenChatStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		t.Fatal(err)
	}

	r := newStubRadio()
	n := &stubNotifier{enabled: true}
	tr := &stubTray{}
	a := New(r, ms, nil, cs, n, tr)
	go a.eventLoop(ctx)

	r.events <- radio.Event{Type: radio.EventPacket, Packet: []byte("hello")}
	time.Sleep(100 * time.Millisecond)

	msgs, err := ms.ListMessages(ctx, "default", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Text != "hello" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
	chats, err := cs.ListChats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || chats[0].ID != "default" {
		t.Fatalf("expected chat inserted, got %+v", chats)
	}
	if n.count != 1 {
		t.Fatalf("expected notifier call, got %d", n.count)
	}
	if !tr.unread {
		t.Fatalf("expected tray unread to be true")
	}

	if err := ms.SetRead(ctx, "default", time.Now()); err != nil {
		t.Fatal(err)
	}
	a.RefreshUnread(ctx)
	if tr.unread {
		t.Fatalf("expected tray unread to be false after marking read")
	}
}

func TestAppUpsertsNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns, err := storage.OpenNodeStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatal(err)
	}

	r := newStubRadio()
	a := New(r, nil, ns, nil, nil, nil)
	go a.eventLoop(ctx)

	r.events <- radio.Event{Type: radio.EventNode, Node: &domain.Node{ID: "n1", RSSI: -90, SNR: 9}}
	time.Sleep(100 * time.Millisecond)

	nodes, err := ns.ListNodes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Signal != domain.SignalGood {
		t.Fatalf("unexpected signal quality: %v", nodes[0].Signal)
	}
}

func TestAppSendTextPersists(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ms, err := storage.OpenMessageStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatal(err)
	}

	cs, err := storage.OpenChatStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		t.Fatal(err)
	}

	r := newStubRadio()
	a := New(r, ms, nil, cs, nil, nil)

	if err := a.SendText(ctx, "chat1", 0, "hello"); err != nil {
		t.Fatalf("SendText error: %v", err)
	}
	msgs, err := ms.ListMessages(ctx, "chat1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].Text != "hello" || msgs[0].IsUnread {
		t.Fatalf("unexpected message %+v", msgs)
	}
	chats, err := cs.ListChats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || chats[0].ID != "chat1" {
		t.Fatalf("expected chat record, got %+v", chats)
	}
	if len(r.sent) != 1 || r.sent[0] != "hello" {
		t.Fatalf("radio not called: %+v", r.sent)
	}
}

func TestAppMarkChatRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ms, err := storage.OpenMessageStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatal(err)
	}

	m := &domain.Message{ChatID: "chat1", Text: "hi", Timestamp: time.Now(), IsUnread: true}
	if err := ms.InsertMessage(ctx, m); err != nil {
		t.Fatal(err)
	}

	tr := &stubTray{}
	a := New(newStubRadio(), ms, nil, nil, nil, tr)

	if err := a.MarkChatRead(ctx, "chat1"); err != nil {
		t.Fatalf("MarkChatRead error: %v", err)
	}
	msgs, err := ms.ListMessages(ctx, "chat1", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].IsUnread {
		t.Fatalf("expected message marked read: %+v", msgs)
	}
	if tr.unread {
		t.Fatalf("expected tray unread cleared")
	}
}

func TestAppListHelpers(t *testing.T) {
	ctx := context.Background()

	ms, err := storage.OpenMessageStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ms.Close()
	if err := ms.Init(ctx); err != nil {
		t.Fatal(err)
	}
	m := &domain.Message{ChatID: "chat1", Text: "hello", Timestamp: time.Now(), IsUnread: true}
	if err := ms.InsertMessage(ctx, m); err != nil {
		t.Fatal(err)
	}

	cs, err := storage.OpenChatStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if err := cs.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := cs.UpsertChat(ctx, &domain.Chat{ID: "chat1", Title: "chat1"}); err != nil {
		t.Fatal(err)
	}

	ns, err := storage.OpenNodeStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ns.UpsertNode(ctx, &domain.Node{ID: "n1"}); err != nil {
		t.Fatal(err)
	}

	a := New(newStubRadio(), ms, ns, cs, nil, nil)

	msgs, err := a.ListMessages(ctx, "chat1", 10)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("ListMessages: %v %d", err, len(msgs))
	}

	chats, err := a.ListChats(ctx)
	if err != nil || len(chats) != 1 {
		t.Fatalf("ListChats: %v %d", err, len(chats))
	}
	if chats[0].UnreadCount != 1 {
		t.Fatalf("expected unread count 1, got %d", chats[0].UnreadCount)
	}

	nodes, err := a.ListNodes(ctx)
	if err != nil || len(nodes) != 1 {
		t.Fatalf("ListNodes: %v %d", err, len(nodes))
	}
}

func TestAppSendHelpers(t *testing.T) {
	ctx := context.Background()
	r := newStubRadio()
	a := New(r, nil, nil, nil, nil, nil)

	if err := a.SendExchangeUserInfo(ctx, 1); err != nil {
		t.Fatalf("SendExchangeUserInfo error: %v", err)
	}
	if err := a.SendTraceroute(ctx, 1); err != nil {
		t.Fatalf("SendTraceroute error: %v", err)
	}
	if len(r.sent) != 2 || r.sent[0] != "userinfo" || r.sent[1] != "traceroute" {
		t.Fatalf("unexpected sent slice: %+v", r.sent)
	}
}

func TestAppNodePreferenceHelpers(t *testing.T) {
	ctx := context.Background()
	ns, err := storage.OpenNodeStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer ns.Close()
	if err := ns.Init(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ns.UpsertNode(ctx, &domain.Node{ID: "n1"}); err != nil {
		t.Fatal(err)
	}
	a := New(newStubRadio(), nil, ns, nil, nil, nil)
	if err := a.SetNodeFavorite(ctx, "n1", true); err != nil {
		t.Fatalf("SetNodeFavorite: %v", err)
	}
	if err := a.SetNodeIgnored(ctx, "n1", true); err != nil {
		t.Fatalf("SetNodeIgnored: %v", err)
	}
	nodes, err := ns.ListNodes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !nodes[0].Favorite || !nodes[0].Ignored {
		t.Fatalf("node flags not set: %+v", nodes[0])
	}
}
