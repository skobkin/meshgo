package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

type sendTextFunc func(chatKey string, text string) <-chan radio.SendResult

func (f sendTextFunc) SendText(chatKey string, text string) <-chan radio.SendResult {
	return f(chatKey, text)
}

func TestChatTypeLabel(t *testing.T) {
	if v := chatTypeLabel(domain.Chat{Type: domain.ChatTypeChannel}); v != "Channel" {
		t.Fatalf("unexpected channel label: %q", v)
	}
	if v := chatTypeLabel(domain.Chat{Type: domain.ChatTypeDM}); v != "DM" {
		t.Fatalf("unexpected dm label: %q", v)
	}
}

func TestChatUnreadMarker(t *testing.T) {
	if got := chatUnreadMarker(true); got != "●" {
		t.Fatalf("unexpected unread marker: %q", got)
	}
	if got := chatUnreadMarker(false); got != " " {
		t.Fatalf("unexpected read marker: %q", got)
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

func TestMessageMetaLine_DirectIncomingShowsSignalBarsOnly(t *testing.T) {
	rssi := -67
	snr := 4.25
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0), RxRSSI: &rssi, RxSNR: &snr},
		true,
	)
	if line != "▂▅█" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaSegments_DirectIncomingSignalBarsColor(t *testing.T) {
	rssi := -125
	snr := -14.0
	segs := messageMetaSegments(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0), RxRSSI: &rssi, RxSNR: &snr},
		true,
	)

	line := richTextSegmentsText(segs)
	if line != "▂▅ " {
		t.Fatalf("unexpected line: %q", line)
	}

	bars := findTextSegmentByContent(t, segs, "▂▅ ")
	if bars.Style.ColorName != theme.ColorNameWarning {
		t.Fatalf("unexpected bars color: %q", bars.Style.ColorName)
	}
}

func TestSignalTooltipSegments_UsesValueColors(t *testing.T) {
	rssi := -125
	snr := -14.0
	segs := signalTooltipSegments(messageMeta{RxRSSI: &rssi, RxSNR: &snr})

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
	if line != "" {
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
		message domain.ChatMessage
		meta    messageMeta
		hasMeta bool
		want    string
		hint    string
	}{
		{name: "incoming no meta", message: domain.ChatMessage{Direction: domain.MessageDirectionIn}, meta: messageMeta{}, hasMeta: false, want: "📡", hint: "via Radio"},
		{name: "incoming via mqtt", message: domain.ChatMessage{Direction: domain.MessageDirectionIn}, meta: messageMeta{ViaMQTT: true}, hasMeta: true, want: "☁", hint: "via MQTT"},
		{name: "incoming transport mqtt", message: domain.ChatMessage{Direction: domain.MessageDirectionIn}, meta: messageMeta{Transport: "TRANSPORT_MQTT"}, hasMeta: true, want: "☁", hint: "via MQTT"},
		{name: "incoming not mqtt", message: domain.ChatMessage{Direction: domain.MessageDirectionIn}, meta: messageMeta{Transport: "TRANSPORT_TCP"}, hasMeta: true, want: "📡", hint: "via Radio"},
		{name: "outgoing hidden", message: domain.ChatMessage{Direction: domain.MessageDirectionOut}, meta: messageMeta{ViaMQTT: true}, hasMeta: true, want: "", hint: ""},
	}

	for _, tc := range tests {
		got, hint := messageTransportBadge(tc.message, tc.meta, tc.hasMeta)
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
	if line != "" {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestMessageMetaLine_DirectHopsHidden(t *testing.T) {
	line := messageMetaLine(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(0)},
		true,
	)
	if line != "" {
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

func TestLatestIncomingAt(t *testing.T) {
	base := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	messages := []domain.ChatMessage{
		{Direction: domain.MessageDirectionOut, At: base.Add(1 * time.Minute)},
		{Direction: domain.MessageDirectionIn, At: base.Add(2 * time.Minute)},
		{Direction: domain.MessageDirectionIn, At: base.Add(5 * time.Minute)},
	}

	got := latestIncomingAt(messages)
	want := base.Add(5 * time.Minute)
	if !got.Equal(want) {
		t.Fatalf("unexpected latest incoming time: got %v want %v", got, want)
	}
}

func TestChatUnreadByKeyAndMarkRead(t *testing.T) {
	base := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	chats := []domain.Chat{
		{Key: "ch:1", Title: "One", Type: domain.ChatTypeChannel},
		{Key: "ch:2", Title: "Two", Type: domain.ChatTypeChannel},
	}
	store := domain.NewChatStore()
	store.Load(chats, map[string][]domain.ChatMessage{
		"ch:1": {
			{ChatKey: "ch:1", Direction: domain.MessageDirectionIn, Body: "hello", At: base},
		},
		"ch:2": {
			{ChatKey: "ch:2", Direction: domain.MessageDirectionOut, Body: "out", At: base},
		},
	})

	read := initialReadIncomingByChat(store, chats)
	unread := chatUnreadByKey(store, chats, read)
	if unread["ch:1"] {
		t.Fatalf("chat ch:1 should be read initially")
	}
	if unread["ch:2"] {
		t.Fatalf("chat ch:2 should be read initially")
	}

	store.AppendMessage(domain.ChatMessage{
		ChatKey:   "ch:1",
		Direction: domain.MessageDirectionIn,
		Body:      "new",
		At:        base.Add(10 * time.Minute),
		MetaJSON:  `{"from":"!abcd1234"}`,
	})

	unread = chatUnreadByKey(store, chats, read)
	if !unread["ch:1"] {
		t.Fatalf("chat ch:1 should be unread after new incoming message")
	}

	markChatRead(store, read, "ch:1")
	unread = chatUnreadByKey(store, chats, read)
	if unread["ch:1"] {
		t.Fatalf("chat ch:1 should be read after markChatRead")
	}
}

func TestMessageStatusBadge_Outgoing(t *testing.T) {
	tests := []struct {
		name    string
		message domain.ChatMessage
		want    string
		hint    string
	}{
		{name: "pending", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusPending}, want: "◷", hint: "Pending"},
		{name: "sent", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusSent}, want: "✓", hint: "Sent"},
		{name: "acked", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusAcked}, want: "✓✓", hint: "Acked"},
		{name: "failed", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusFailed, StatusReason: "NO_ROUTE"}, want: "⚠", hint: "Failed: NO_ROUTE"},
	}
	for _, tc := range tests {
		got, hint := messageStatusBadge(tc.message)
		if got != tc.want {
			t.Fatalf("%s: expected badge %q, got %q", tc.name, tc.want, got)
		}
		if hint != tc.hint {
			t.Fatalf("%s: expected tooltip %q, got %q", tc.name, tc.hint, hint)
		}
	}
}

func TestMessageStatusBadge_IncomingHidden(t *testing.T) {
	got, hint := messageStatusBadge(domain.ChatMessage{
		Direction: domain.MessageDirectionIn,
		Status:    domain.MessageStatusAcked,
	})
	if got != "" || hint != "" {
		t.Fatalf("unexpected incoming badge (%q, %q)", got, hint)
	}
}

func TestMessageTimeLabel(t *testing.T) {
	at := time.Date(2026, 2, 11, 8, 7, 0, 0, time.Local)
	if got := messageTimeLabel(at); got != "08:07" {
		t.Fatalf("unexpected time label: %q", got)
	}
	if got := messageTimeLabel(time.Time{}); got != "" {
		t.Fatalf("unexpected zero time label: %q", got)
	}
}

func TestChatsTabSendFailureShowsStatusAndKeepsEntryText(t *testing.T) {
	store := domain.NewChatStore()
	store.Load(
		[]domain.Chat{{Key: "ch:general", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: time.Now()}},
		map[string][]domain.ChatMessage{},
	)

	tab := newChatsTab(
		store,
		sendTextFunc(func(_ string, _ string) <-chan radio.SendResult {
			result := make(chan radio.SendResult, 1)
			result <- radio.SendResult{Err: errors.New("send failed")}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		"ch:general",
		nil,
	)
	_ = fynetest.NewTempWindow(t, tab)

	entry := mustFindEntryByPlaceholder(t, tab, "Type message (max 200 bytes)")
	sendButton := mustFindButtonByText(t, tab, "Send")
	entry.SetText("hello from test")

	fynetest.Tap(sendButton)

	waitForCondition(t, func() bool {
		label := findLabelByPrefix(tab, "Send failed: ")

		return label != nil && strings.TrimSpace(label.Text) == "Send failed: send failed"
	})

	if got := entry.Text; got != "hello from test" {
		t.Fatalf("entry text should stay unchanged after send failure, got %q", got)
	}
}

func TestChatsTabSendSuccessClearsPreviousFailureStatus(t *testing.T) {
	store := domain.NewChatStore()
	store.Load(
		[]domain.Chat{{Key: "ch:general", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: time.Now()}},
		map[string][]domain.ChatMessage{},
	)

	sendAttempt := 0
	tab := newChatsTab(
		store,
		sendTextFunc(func(_ string, _ string) <-chan radio.SendResult {
			sendAttempt++
			result := make(chan radio.SendResult, 1)
			if sendAttempt == 1 {
				result <- radio.SendResult{Err: errors.New("send failed")}
			} else {
				result <- radio.SendResult{}
			}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		"ch:general",
		nil,
	)
	_ = fynetest.NewTempWindow(t, tab)

	entry := mustFindEntryByPlaceholder(t, tab, "Type message (max 200 bytes)")
	sendButton := mustFindButtonByText(t, tab, "Send")
	entry.SetText("first")
	fynetest.Tap(sendButton)

	var statusLabel *widget.Label
	waitForCondition(t, func() bool {
		statusLabel = findLabelByPrefix(tab, "Send failed: ")

		return statusLabel != nil
	})

	entry.SetText("second")
	fynetest.Tap(sendButton)

	waitForCondition(t, func() bool {
		return statusLabel.Text == "" && entry.Text == ""
	})
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

func findLabelByPrefix(root fyne.CanvasObject, prefix string) *widget.Label {
	for _, object := range fynetest.LaidOutObjects(root) {
		label, ok := object.(*widget.Label)
		if !ok {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(label.Text), prefix) {
			return label
		}
	}

	return nil
}
