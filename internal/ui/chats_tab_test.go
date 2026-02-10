package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestMessageTextLine_IncomingShowsSender(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn, Body: "hello"},
		messageMeta{From: "!1234abcd"},
		true,
		nil,
	)
	if line != "< !1234abcd: hello" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTextLine_IncomingPrefersResolvedSenderName(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn, Body: "hello"},
		messageMeta{From: "!1234abcd"},
		true,
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}
			return "Alice"
		},
	)
	if line != "< Alice: hello" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTextLine_OutgoingUsesArrow(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Body: "ping"},
		messageMeta{},
		false,
		nil,
	)
	if line != "> ping" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaLine_DirectIncomingShowsRSSIAndSNR(t *testing.T) {
	rssi := -67
	snr := 4.25
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0), RxRSSI: &rssi, RxSNR: &snr},
		true,
	)
	if line != "hops: 0 | RSSI: -67 | SNR: 4.25" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaLine_MQTTShowsMarker(t *testing.T) {
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(2), ViaMQTT: true},
		true,
	)
	if line != "hops: 2 | [MQTT]" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaLine_UnknownHopsGracefulFallback(t *testing.T) {
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{},
		false,
	)
	if line != "hops: ?" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func ptrInt(v int) *int {
	return &v
}
