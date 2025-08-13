package radio

import (
	"context"
	"io"
	"testing"
	"time"
)

func TestJitterDurationRange(t *testing.T) {
	base := 100 * time.Millisecond
	jitter := 0.2
	for i := 0; i < 100; i++ {
		d := jitterDuration(base, jitter)
		if d < 80*time.Millisecond || d > 120*time.Millisecond {
			t.Fatalf("jitter out of range: %v", d)
		}
	}
}

func TestReadLoopEmitsPacketEvent(t *testing.T) {
	c := New(ReconnectConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := &stubTransport{packet: []byte("hi")}
	go func() { _ = c.readLoop(ctx, st) }()

	select {
	case ev := <-c.Events():
		if ev.Type != EventPacket || string(ev.Packet) != "hi" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("no event received")
	}
}

type stubTransport struct {
	packet  []byte
	read    bool
	written []byte
}

func (s *stubTransport) Connect(ctx context.Context) error { return nil }
func (s *stubTransport) Close() error                      { return nil }
func (s *stubTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	if s.read {
		return nil, io.EOF
	}
	s.read = true
	return s.packet, nil
}
func (s *stubTransport) WritePacket(ctx context.Context, b []byte) error {
	s.written = append([]byte(nil), b...)
	return nil
}
func (s *stubTransport) IsConnected() bool { return true }
func (s *stubTransport) Endpoint() string  { return "" }

func TestSendTextWritesPacket(t *testing.T) {
	c := New(ReconnectConfig{})
	st := &stubTransport{}
	c.t = st
	if err := c.SendText(context.Background(), "c", 0, "msg"); err != nil {
		t.Fatalf("SendText error: %v", err)
	}
	if string(st.written) != "msg" {
		t.Fatalf("expected 'msg', got %q", st.written)
	}
}
