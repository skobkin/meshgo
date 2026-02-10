package ui

import (
	"strings"
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestChatTypeLabel(t *testing.T) {
	if v := chatTypeLabel(domain.Chat{Type: domain.ChatTypeChannel}); v != "Channel" {
		t.Fatalf("unexpected channel label: %q", v)
	}
	if v := chatTypeLabel(domain.Chat{Type: domain.ChatTypeDM}); v != "DM" {
		t.Fatalf("unexpected dm label: %q", v)
	}
}

func TestChatPreviewLine_Empty(t *testing.T) {
	if got := chatPreviewLine(nil, nil); got != "No messages yet" {
		t.Fatalf("unexpected preview: %q", got)
	}
}

func TestChatPreviewLine_Outgoing(t *testing.T) {
	preview := chatPreviewLine(
		[]domain.ChatMessage{
			{Direction: domain.MessageDirectionOut, Body: "  hello there  "},
		},
		nil,
	)
	if preview != "you: hello there" {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestChatPreviewLine_IncomingResolvedSender(t *testing.T) {
	preview := chatPreviewLine(
		[]domain.ChatMessage{
			{
				Direction: domain.MessageDirectionIn,
				Body:      "status update",
				MetaJSON:  `{"from":"!1234abcd"}`,
			},
		},
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}
			return "Alice"
		},
	)
	if preview != "Alice: status update" {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestChatPreviewLine_TruncatesLong(t *testing.T) {
	preview := chatPreviewLine(
		[]domain.ChatMessage{
			{Direction: domain.MessageDirectionOut, Body: strings.Repeat("x", 200)},
		},
		nil,
	)
	if len([]rune(preview)) != 72 {
		t.Fatalf("unexpected preview length: %d (%q)", len([]rune(preview)), preview)
	}
	if !strings.HasSuffix(preview, "...") {
		t.Fatalf("expected truncated suffix, got %q", preview)
	}
}

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

func TestMessageTextParts_IncomingWithSender(t *testing.T) {
	prefix, sender, body, hasSender := messageTextParts(
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
	if prefix != "<" || sender != "Alice" || body != "hello" || !hasSender {
		t.Fatalf("unexpected parts: prefix=%q sender=%q body=%q hasSender=%v", prefix, sender, body, hasSender)
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

func TestChatIndexByKey(t *testing.T) {
	chats := []domain.Chat{
		{Key: "dm:alice"},
		{Key: "ch:1"},
		{Key: "ch:2"},
	}

	if got := chatIndexByKey(chats, "ch:1"); got != 1 {
		t.Fatalf("unexpected index for ch:1: %d", got)
	}
	if got := chatIndexByKey(chats, "missing"); got != -1 {
		t.Fatalf("unexpected index for missing key: %d", got)
	}
	if got := chatIndexByKey(chats, ""); got != -1 {
		t.Fatalf("unexpected index for empty key: %d", got)
	}
}

func ptrInt(v int) *int {
	return &v
}
