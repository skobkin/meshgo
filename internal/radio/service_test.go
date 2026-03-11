package radio

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNormalizeMessageStatus_BroadcastAckBecomesSent(t *testing.T) {
	svc := &Service{ackTrack: make(map[string]ackTrackState)}
	svc.markAckTracked("101", broadcastNodeNum)

	got := svc.normalizeMessageStatus(domain.MessageStatusUpdate{
		DeviceMessageID: "101",
		Status:          domain.MessageStatusAcked,
		FromNodeNum:     0x0000beef,
	})

	if got.Status != domain.MessageStatusSent {
		t.Fatalf("expected sent status, got %v", got.Status)
	}
	if _, ok := svc.ackTrackStateFor("101"); ok {
		t.Fatalf("expected broadcast message tracking to be cleared")
	}
}

func TestNormalizeMessageStatus_DMRelayAckBecomesSentAndKeepsTracking(t *testing.T) {
	svc := &Service{ackTrack: make(map[string]ackTrackState)}
	svc.markAckTracked("202", 0x0000cafe)

	got := svc.normalizeMessageStatus(domain.MessageStatusUpdate{
		DeviceMessageID: "202",
		Status:          domain.MessageStatusAcked,
		FromNodeNum:     0x0000beef,
	})

	if got.Status != domain.MessageStatusSent {
		t.Fatalf("expected sent status, got %v", got.Status)
	}
	if _, ok := svc.ackTrackStateFor("202"); !ok {
		t.Fatalf("expected dm tracking to remain until destination ack")
	}
}

func TestNormalizeMessageStatus_DMDestinationAckStaysAckedAndClearsTracking(t *testing.T) {
	svc := &Service{ackTrack: make(map[string]ackTrackState)}
	svc.markAckTracked("303", 0x0000cafe)

	got := svc.normalizeMessageStatus(domain.MessageStatusUpdate{
		DeviceMessageID: "303",
		Status:          domain.MessageStatusAcked,
		FromNodeNum:     0x0000cafe,
	})

	if got.Status != domain.MessageStatusAcked {
		t.Fatalf("expected acked status, got %v", got.Status)
	}
	if _, ok := svc.ackTrackStateFor("303"); ok {
		t.Fatalf("expected dm tracking to be cleared on destination ack")
	}
}

func TestNormalizeMessageStatus_FailedClearsTracking(t *testing.T) {
	svc := &Service{ackTrack: make(map[string]ackTrackState)}
	svc.markAckTracked("404", 0x0000cafe)

	got := svc.normalizeMessageStatus(domain.MessageStatusUpdate{
		DeviceMessageID: "404",
		Status:          domain.MessageStatusFailed,
		Reason:          "NO_ROUTE",
	})

	if got.Status != domain.MessageStatusFailed {
		t.Fatalf("expected failed status, got %v", got.Status)
	}
	if _, ok := svc.ackTrackStateFor("404"); ok {
		t.Fatalf("expected tracking to be cleared on failure")
	}
}

func TestRunReader_IgnoresIdleReadTimeout(t *testing.T) {
	messageBus := bus.New(slog.New(slog.NewTextHandler(io.Discard, nil)))
	rawSub := messageBus.Subscribe(bus.TopicRawFrameIn)
	defer messageBus.Unsubscribe(rawSub, bus.TopicRawFrameIn)

	transport := &stubTransport{
		readFrames: []stubReadFrameResult{
			{err: errors.Join(errors.New("read frame header byte 1"), context.DeadlineExceeded)},
			{payload: []byte{0x01, 0x02, 0x03}},
		},
	}
	codec := &stubCodec{}
	svc := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), messageBus, transport, codec)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- svc.runReader(ctx)
	}()

	select {
	case raw := <-rawSub:
		frame, ok := raw.(busmsg.RawFrame)
		if !ok {
			t.Fatalf("expected raw frame event, got %T", raw)
		}
		if frame.Hex != "010203" {
			t.Fatalf("expected hex payload 010203, got %q", frame.Hex)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for raw frame publish")
	}

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for reader shutdown")
	}

	if got := transport.ReadCalls(); got < 2 {
		t.Fatalf("expected reader to retry after idle timeout, got %d read calls", got)
	}
	if got := codec.DecodeCalls(); got != 1 {
		t.Fatalf("expected one decode after timeout retry, got %d", got)
	}
}

type stubTransport struct {
	mu         sync.Mutex
	readFrames []stubReadFrameResult
	readCalls  int
}

type stubReadFrameResult struct {
	payload []byte
	err     error
}

func (t *stubTransport) Name() string { return "stub" }

func (t *stubTransport) Connect(context.Context) error { return nil }

func (t *stubTransport) Close() error { return nil }

func (t *stubTransport) ReadFrame(ctx context.Context) ([]byte, error) {
	t.mu.Lock()
	t.readCalls++
	callIdx := t.readCalls - 1
	if callIdx < len(t.readFrames) {
		result := t.readFrames[callIdx]
		t.mu.Unlock()

		return result.payload, result.err
	}
	t.mu.Unlock()

	<-ctx.Done()

	return nil, ctx.Err()
}

func (t *stubTransport) WriteFrame(context.Context, []byte) error { return nil }

func (t *stubTransport) ReadCalls() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.readCalls
}

type stubCodec struct {
	mu          sync.Mutex
	decodeCalls int
}

func (c *stubCodec) EncodeWantConfig() ([]byte, error) { return nil, nil }

func (c *stubCodec) EncodeHeartbeat() ([]byte, error) { return nil, nil }

func (c *stubCodec) EncodeText(string, string, TextSendOptions) (EncodedText, error) {
	return EncodedText{}, nil
}

func (c *stubCodec) EncodeAdmin(uint32, uint32, bool, *generated.AdminMessage) (EncodedAdmin, error) {
	return EncodedAdmin{}, nil
}

func (c *stubCodec) EncodeTraceroute(uint32, uint32) (EncodedTraceroute, error) {
	return EncodedTraceroute{}, nil
}

func (c *stubCodec) DecodeFromRadio([]byte) (DecodedFrame, error) {
	c.mu.Lock()
	c.decodeCalls++
	c.mu.Unlock()

	return DecodedFrame{}, nil
}

func (c *stubCodec) DecodeCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.decodeCalls
}
