package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
		nil,
	)
	if line != "!1234abcd: hello" {
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
		nil,
	)
	if line != "Alice: hello" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTextLine_OutgoingUsesSender(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Body: "ping"},
		messageMeta{},
		false,
		nil,
		nil,
	)
	if line != "you: ping" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTextLine_OutgoingUsesResolvedSenderName(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Body: "ping"},
		messageMeta{From: "!1234abcd"},
		true,
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}
			return "Local Node"
		},
		nil,
	)
	if line != "Local Node: ping" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTextParts_IncomingWithSender(t *testing.T) {
	sender, body, hasSender := messageTextParts(
		domain.ChatMessage{Direction: domain.MessageDirectionIn, Body: "hello"},
		messageMeta{From: "!1234abcd"},
		true,
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}
			return "Alice"
		},
		nil,
	)
	if sender != "Alice" || body != "hello" || !hasSender {
		t.Fatalf("unexpected parts: sender=%q body=%q hasSender=%v", sender, body, hasSender)
	}
}

func TestMessageTextSegments_SenderIsBold(t *testing.T) {
	segs := messageTextSegments(
		domain.ChatMessage{Direction: domain.MessageDirectionIn, Body: "hello"},
		messageMeta{From: "!1234abcd"},
		true,
		func(_ string) string { return "Alice" },
		nil,
	)
	if len(segs) != 2 {
		t.Fatalf("unexpected segment count: %d", len(segs))
	}
	sender, ok := segs[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("sender segment type mismatch: %T", segs[0])
	}
	if sender.Text != "Alice" {
		t.Fatalf("unexpected sender text: %q", sender.Text)
	}
	if !sender.Style.TextStyle.Bold {
		t.Fatalf("sender segment should be bold")
	}
}

func TestMessageTextLine_OutgoingUsesLocalNodeResolver(t *testing.T) {
	line := messageTextLine(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Body: "ping"},
		messageMeta{},
		false,
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}
			return "Local Node"
		},
		func() string { return "!1234abcd" },
	)
	if line != "Local Node: ping" {
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
	if line != "⓪ ▂▄▆█ RSSI: -67 SNR: 4.25" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaSegments_DirectIncomingSignalBarsAndValueColors(t *testing.T) {
	rssi := -125
	snr := -14.0
	segs := messageMetaSegments(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0), RxRSSI: &rssi, RxSNR: &snr},
		true,
	)

	line := richTextSegmentsText(segs)
	if line != "⓪ ▂▄▆  RSSI: -125 SNR: -14.00" {
		t.Fatalf("unexpected line: %q", line)
	}

	bars := findTextSegmentByContent(t, segs, "▂▄▆ ")
	if bars.Style.ColorName != theme.ColorNameWarning {
		t.Fatalf("unexpected bars color: %q", bars.Style.ColorName)
	}

	rssiValue := findTextSegmentByContent(t, segs, "-125")
	if rssiValue.Style.ColorName != theme.ColorNameWarning {
		t.Fatalf("unexpected RSSI color: %q", rssiValue.Style.ColorName)
	}

	snrValue := findTextSegmentByContent(t, segs, "-14.00")
	if snrValue.Style.ColorName != theme.ColorNameWarning {
		t.Fatalf("unexpected SNR color: %q", snrValue.Style.ColorName)
	}
}

func TestMessageMetaSegments_UnknownSignalOmitsBars(t *testing.T) {
	rssi := -67
	segs := messageMetaSegments(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0), RxRSSI: &rssi},
		true,
	)

	line := richTextSegmentsText(segs)
	if line != "⓪ RSSI: -67" {
		t.Fatalf("unexpected line: %q", line)
	}
	if strings.Contains(line, "▂") || strings.Contains(line, "▄") {
		t.Fatalf("signal bars should be omitted for unknown signal quality: %q", line)
	}
}

func TestMessageMetaLine_MQTTShowsHopsOnly(t *testing.T) {
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(2), ViaMQTT: true},
		true,
	)
	if line != "②" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageTransportBadge(t *testing.T) {
	tests := []struct {
		name    string
		meta    messageMeta
		hasMeta bool
		want    string
		hint    string
	}{
		{name: "no meta", meta: messageMeta{}, hasMeta: false, want: "", hint: ""},
		{name: "via mqtt", meta: messageMeta{ViaMQTT: true}, hasMeta: true, want: "☁", hint: "via MQTT"},
		{name: "transport mqtt", meta: messageMeta{Transport: "TRANSPORT_MQTT"}, hasMeta: true, want: "☁", hint: "via MQTT"},
		{name: "not mqtt", meta: messageMeta{Transport: "TRANSPORT_TCP"}, hasMeta: true, want: "", hint: ""},
	}

	for _, tc := range tests {
		got, hint := messageTransportBadge(tc.meta, tc.hasMeta)
		if got != tc.want {
			t.Fatalf("%s: expected badge %q, got %q", tc.name, tc.want, got)
		}
		if hint != tc.hint {
			t.Fatalf("%s: expected tooltip %q, got %q", tc.name, tc.hint, hint)
		}
	}
}

func TestMessageMetaLine_UnknownHopsGracefulFallback(t *testing.T) {
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{},
		false,
	)
	if line != "?" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestHopBadge(t *testing.T) {
	tests := []struct {
		name string
		hops int
		want string
	}{
		{name: "unknown", hops: -1, want: "?"},
		{name: "zero", hops: 0, want: "⓪"},
		{name: "max_meshtastic", hops: 7, want: "⑦"},
		{name: "fallback", hops: 8, want: "h8"},
	}

	for _, tc := range tests {
		if got := hopBadge(tc.hops); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
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

func TestMessageStatusLine_Outgoing(t *testing.T) {
	tests := []struct {
		name   string
		status domain.MessageStatus
		want   string
	}{
		{name: "pending", status: domain.MessageStatusPending, want: "Pending"},
		{name: "sent", status: domain.MessageStatusSent, want: "Sent"},
		{name: "acked", status: domain.MessageStatusAcked, want: "Acked"},
		{name: "failed", status: domain.MessageStatusFailed, want: "Failed"},
	}

	for _, tc := range tests {
		got := messageStatusLine(domain.ChatMessage{
			Direction: domain.MessageDirectionOut,
			Status:    tc.status,
		})
		if got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestMessageStatusLine_IncomingHidden(t *testing.T) {
	got := messageStatusLine(domain.ChatMessage{
		Direction: domain.MessageDirectionIn,
		Status:    domain.MessageStatusAcked,
	})
	if got != "" {
		t.Fatalf("expected empty status for incoming message, got %q", got)
	}
}

func ptrInt(v int) *int {
	return &v
}

func richTextSegmentsText(segs []widget.RichTextSegment) string {
	var b strings.Builder
	for _, seg := range segs {
		text, ok := seg.(*widget.TextSegment)
		if !ok {
			continue
		}
		b.WriteString(text.Text)
	}
	return b.String()
}

func findTextSegmentByContent(t *testing.T, segs []widget.RichTextSegment, content string) *widget.TextSegment {
	t.Helper()
	for _, seg := range segs {
		text, ok := seg.(*widget.TextSegment)
		if !ok {
			continue
		}
		if text.Text == content {
			return text
		}
	}
	t.Fatalf("segment with text %q not found", content)
	return nil
}
