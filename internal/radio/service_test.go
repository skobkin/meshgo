package radio

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
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
