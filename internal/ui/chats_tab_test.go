package ui

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

type sendTextFunc func(chatKey string, text string, opts radio.TextSendOptions) <-chan radio.SendResult

func (f sendTextFunc) SendText(chatKey string, text string, opts radio.TextSendOptions) <-chan radio.SendResult {
	return f(chatKey, text, opts)
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

func TestChatPreviewLine_IgnoresReactionMessages(t *testing.T) {
	preview := chatPreviewLine(
		[]domain.ChatMessage{
			{
				DeviceMessageID:        "100",
				Direction:              domain.MessageDirectionIn,
				Body:                   "hello",
				MetaJSON:               `{"from":"!1234abcd"}`,
				ReplyToDeviceMessageID: "",
				Emoji:                  0,
			},
			{
				DeviceMessageID:        "101",
				Direction:              domain.MessageDirectionIn,
				Body:                   "👍",
				ReplyToDeviceMessageID: "100",
				Emoji:                  1,
				MetaJSON:               `{"from":"!1234abcd"}`,
			},
		},
		func(_ string) string { return "Alice" },
	)
	if preview != "Alice: hello" {
		t.Fatalf("unexpected preview: %q", preview)
	}
}

func TestBuildChatMessageView_GroupsReactionsByTargetAndEmoji(t *testing.T) {
	view := buildChatMessageView(
		[]domain.ChatMessage{
			{
				DeviceMessageID: "200",
				Direction:       domain.MessageDirectionIn,
				Body:            "base",
				MetaJSON:        `{"from":"!aaaa0001"}`,
			},
			{
				DeviceMessageID:        "201",
				Direction:              domain.MessageDirectionIn,
				Body:                   "👍",
				ReplyToDeviceMessageID: "200",
				Emoji:                  1,
				MetaJSON:               `{"from":"!bbbb0002"}`,
			},
			{
				DeviceMessageID:        "202",
				Direction:              domain.MessageDirectionIn,
				Body:                   "👍",
				ReplyToDeviceMessageID: "200",
				Emoji:                  1,
				MetaJSON:               `{"from":"!bbbb0002"}`,
			},
			{
				DeviceMessageID:        "203",
				Direction:              domain.MessageDirectionIn,
				Body:                   "🔥",
				ReplyToDeviceMessageID: "200",
				Emoji:                  1,
				MetaJSON:               `{"from":"!cccc0003"}`,
			},
		},
		func(nodeID string) string {
			switch nodeID {
			case "!bbbb0002":
				return "Bob"
			case "!cccc0003":
				return "Carol"
			default:
				return nodeID
			}
		},
		nil,
	)

	if len(view.Timeline) != 1 {
		t.Fatalf("expected one timeline message, got %d", len(view.Timeline))
	}
	reactions := view.ReactionsByTargetDeviceID["200"]
	if len(reactions) != 2 {
		t.Fatalf("expected two reaction chips, got %d", len(reactions))
	}
	if reactions[0].Emoji != "👍" || len(reactions[0].Senders) != 1 || reactions[0].Senders[0] != "Bob" {
		t.Fatalf("unexpected first reaction chip: %+v", reactions[0])
	}
	if reactions[1].Emoji != "🔥" || len(reactions[1].Senders) != 1 || reactions[1].Senders[0] != "Carol" {
		t.Fatalf("unexpected second reaction chip: %+v", reactions[1])
	}
}

func TestReactionChipSegments_EmojiUsesHeadingSizeAndCount(t *testing.T) {
	segments := reactionChipSegments(reactionChip{
		Emoji:   "👍",
		Senders: []string{"Alice", "Bob"},
	})
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}
	emoji, ok := segments[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("expected text segment for emoji, got %T", segments[0])
	}
	if emoji.Text != "👍" {
		t.Fatalf("unexpected emoji segment text: %q", emoji.Text)
	}
	if emoji.Style.SizeName != theme.SizeNameHeadingText {
		t.Fatalf("expected heading text size for emoji, got %q", emoji.Style.SizeName)
	}
	count, ok := segments[2].(*widget.TextSegment)
	if !ok {
		t.Fatalf("expected text segment for count, got %T", segments[2])
	}
	if count.Text != "2" {
		t.Fatalf("unexpected reaction count: %q", count.Text)
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

func TestMessageMetaChunksWithContext_HopTooltipIncludesRelayAndMQTT(t *testing.T) {
	chunks := messageMetaChunksWithContext(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(3), RelayNode: ptrUint32(0xcd), ViaMQTT: true},
		true,
		nil,
		func(relayNode uint32) string {
			if relayNode != 0xcd {
				t.Fatalf("unexpected relay node value: %x", relayNode)
			}

			return "Relay Alpha"
		},
	)
	if len(chunks) == 0 {
		t.Fatalf("expected hop chunk")
	}

	tooltip := richTextSegmentsText(chunks[0].Tooltip)
	want := "Hops: 3\nReceived from: Relay Alpha (last relay node)\nMQTT involved"
	if tooltip != want {
		t.Fatalf("unexpected hop tooltip:\nwant: %q\ngot:  %q", want, tooltip)
	}
}

func TestMessageMetaChunksWithContext_HopTooltipResolvesRelayFromSenderNode(t *testing.T) {
	chunks := messageMetaChunksWithContext(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(2), From: "!1234abcd", RelayNode: ptrUint32(0xcd)},
		true,
		func(nodeID string) string {
			if nodeID != "!1234abcd" {
				t.Fatalf("unexpected node id: %q", nodeID)
			}

			return "Alice"
		},
		nil,
	)
	if len(chunks) == 0 {
		t.Fatalf("expected hop chunk")
	}

	tooltip := richTextSegmentsText(chunks[0].Tooltip)
	want := "Hops: 2\nReceived from: Alice (last relay node)"
	if tooltip != want {
		t.Fatalf("unexpected hop tooltip:\nwant: %q\ngot:  %q", want, tooltip)
	}
}

func TestMessageMetaChunksWithContext_HopTooltipFallsBackToRelayByte(t *testing.T) {
	chunks := messageMetaChunksWithContext(
		domain.ChatMessage{Direction: domain.MessageDirectionIn},
		messageMeta{Hops: ptrInt(1), RelayNode: ptrUint32(0x7f)},
		true,
		nil,
		nil,
	)
	if len(chunks) == 0 {
		t.Fatalf("expected hop chunk")
	}

	tooltip := richTextSegmentsText(chunks[0].Tooltip)
	want := "Hops: 1\nReceived from: 0x7f (last relay node)"
	if tooltip != want {
		t.Fatalf("unexpected hop tooltip:\nwant: %q\ngot:  %q", want, tooltip)
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
		{name: "pending", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusPending}, want: "◷", hint: messageStatusPendingTooltipText},
		{name: "sent channel", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusSent, ChatKey: "channel:0"}, want: "✓", hint: messageStatusSentChannelTooltipText},
		{name: "sent dm", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusSent, ChatKey: "dm:!abcd1234"}, want: "✓", hint: messageStatusSentDMTooltipText},
		{name: "acked channel", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusAcked, ChatKey: "channel:0"}, want: "✓✓", hint: messageStatusAckedChannelTooltipText},
		{name: "acked dm", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusAcked, ChatKey: "dm:!abcd1234"}, want: "✓✓", hint: messageStatusAckedDMTooltipText},
		{name: "failed", message: domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusFailed, StatusReason: "NO_ROUTE"}, want: "⚠", hint: messageStatusFailedTooltipText + "\nReason: NO_ROUTE."},
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

func TestMessageStatusTooltipContent_UsesPrebuiltContent(t *testing.T) {
	cache := newMessageStatusTooltipCache()

	if got := messageStatusTooltipContent(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusPending},
		cache,
	); got != cache.pending {
		t.Fatalf("pending should reuse prebuilt tooltip object")
	}

	if got := messageStatusTooltipContent(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusSent, ChatKey: "channel:0"},
		cache,
	); got != cache.sentChannel {
		t.Fatalf("channel sent should reuse prebuilt channel-sent tooltip object")
	}

	if got := messageStatusTooltipContent(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusSent, ChatKey: "dm:!abcd1234"},
		cache,
	); got != cache.sentDM {
		t.Fatalf("dm sent should reuse prebuilt dm-sent tooltip object")
	}

	if got := messageStatusTooltipContent(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusAcked, ChatKey: "dm:!abcd1234"},
		cache,
	); got != cache.ackedDM {
		t.Fatalf("dm ack should reuse prebuilt dm tooltip object")
	}

	if got := messageStatusTooltipContent(
		domain.ChatMessage{Direction: domain.MessageDirectionOut, Status: domain.MessageStatusFailed, StatusReason: "NO_ROUTE"},
		cache,
	); got != nil {
		t.Fatalf("failed with reason should use dynamic tooltip text")
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

func TestShouldUpdateMessageItemHeight(t *testing.T) {
	tests := []struct {
		name       string
		hasPrev    bool
		prevHeight float32
		prevWidth  float32
		rowHeight  float32
		rowWidth   float32
		want       bool
	}{
		{
			name:      "first measurement",
			hasPrev:   false,
			rowHeight: 64,
			rowWidth:  420,
			want:      true,
		},
		{
			name:       "height grew",
			hasPrev:    true,
			prevHeight: 64,
			prevWidth:  420,
			rowHeight:  90,
			rowWidth:   420,
			want:       true,
		},
		{
			name:       "height shrank after wider layout",
			hasPrev:    true,
			prevHeight: 120,
			prevWidth:  300,
			rowHeight:  76,
			rowWidth:   420,
			want:       true,
		},
		{
			name:       "height shrank without wider layout",
			hasPrev:    true,
			prevHeight: 120,
			prevWidth:  420,
			rowHeight:  76,
			rowWidth:   420,
			want:       false,
		},
		{
			name:       "wider layout but same height",
			hasPrev:    true,
			prevHeight: 76,
			prevWidth:  300,
			rowHeight:  76,
			rowWidth:   420,
			want:       false,
		},
		{
			name:       "tiny jitter ignored",
			hasPrev:    true,
			prevHeight: 76,
			prevWidth:  420,
			rowHeight:  76.2,
			rowWidth:   420,
			want:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldUpdateMessageItemHeight(
				tc.hasPrev,
				tc.prevHeight,
				tc.prevWidth,
				tc.rowHeight,
				tc.rowWidth,
			)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestChatsTabSendFailureShowsStatusAndKeepsEntryText(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	store := domain.NewChatStore()
	store.Load(
		[]domain.Chat{{Key: "ch:general", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: time.Now()}},
		map[string][]domain.ChatMessage{},
	)

	tab := newChatsTab(
		nil,
		store,
		sendTextFunc(func(_ string, _ string, _ radio.TextSendOptions) <-chan radio.SendResult {
			result := make(chan radio.SendResult, 1)
			result <- radio.SendResult{Err: errors.New("send failed")}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		nil,
		"ch:general",
		nil,
		nil,
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

	var got string
	fyne.DoAndWait(func() {
		got = entry.Text
	})
	if got != "hello from test" {
		t.Fatalf("entry text should stay unchanged after send failure, got %q", got)
	}
}

func TestChatsTabSendSuccessClearsPreviousFailureStatus(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	store := domain.NewChatStore()
	store.Load(
		[]domain.Chat{{Key: "ch:general", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: time.Now()}},
		map[string][]domain.ChatMessage{},
	)

	sendAttempt := 0
	tab := newChatsTab(
		nil,
		store,
		sendTextFunc(func(_ string, _ string, _ radio.TextSendOptions) <-chan radio.SendResult {
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
		nil,
		"ch:general",
		nil,
		nil,
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

		return statusLabel != nil &&
			strings.TrimSpace(statusLabel.Text) == "Send failed: send failed" &&
			!sendButton.Disabled() &&
			!entry.Disabled()
	})

	fyne.DoAndWait(func() {
		entry.SetText("second")
	})
	fynetest.Tap(sendButton)

	waitForCondition(t, func() bool {
		return statusLabel.Text == "" && entry.Text == ""
	})
}

func TestChatsTabMessageRichTextWrapsLongSingleLine(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	store := domain.NewChatStore()
	store.Load(
		[]domain.Chat{{Key: "ch:general", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: time.Now()}},
		map[string][]domain.ChatMessage{
			"ch:general": {
				{
					ChatKey:   "ch:general",
					Direction: domain.MessageDirectionOut,
					Body:      "wrap-token " + strings.Repeat("longword ", 48),
					At:        time.Now(),
					Status:    domain.MessageStatusSent,
				},
			},
		},
	)

	tab := newChatsTab(
		nil,
		store,
		sendTextFunc(func(_ string, _ string, _ radio.TextSendOptions) <-chan radio.SendResult {
			result := make(chan radio.SendResult, 1)
			result <- radio.SendResult{}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		nil,
		"ch:general",
		nil,
		nil,
		nil,
	)
	_ = fynetest.NewTempWindow(t, tab)

	waitForCondition(t, func() bool {
		return findRichTextBySubstringAndWrapping(tab, "wrap-token", fyne.TextWrapWord) != nil
	})
}

func TestChatsTabOpenRequestSelectsExistingChat(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	store := domain.NewChatStore()
	base := time.Now()
	store.Load(
		[]domain.Chat{
			{Key: "channel:0", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: base.Add(1 * time.Hour)},
			{Key: "dm:!0000002a", Title: "dm:!0000002a", Type: domain.ChatTypeDM, UpdatedAt: base},
		},
		map[string][]domain.ChatMessage{},
	)

	openRequests := make(chan string, 1)
	var selectedKeysMu sync.Mutex
	selectedKeys := make([]string, 0, 2)
	tab := newChatsTab(
		nil,
		store,
		sendTextFunc(func(_ string, _ string, _ radio.TextSendOptions) <-chan radio.SendResult {
			result := make(chan radio.SendResult, 1)
			result <- radio.SendResult{}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		nil,
		"channel:0",
		openRequests,
		func(chatKey string) {
			selectedKeysMu.Lock()
			selectedKeys = append(selectedKeys, chatKey)
			selectedKeysMu.Unlock()
		},
		nil,
	)
	_ = fynetest.NewTempWindow(t, tab)

	openRequests <- "dm:!0000002a"

	waitForCondition(t, func() bool {
		selectedKeysMu.Lock()
		defer selectedKeysMu.Unlock()
		if len(selectedKeys) == 0 {
			return false
		}

		return selectedKeys[len(selectedKeys)-1] == "dm:!0000002a"
	})
}

func TestChatListContextMenuDeleteDisabledForChannel(t *testing.T) {
	menu := newChatListContextMenu(domain.Chat{Key: "channel:0", Title: "General", Type: domain.ChatTypeChannel}, nil)
	if len(menu.Items) != 1 {
		t.Fatalf("expected one menu item, got %d", len(menu.Items))
	}
	if !menu.Items[0].Disabled {
		t.Fatalf("expected delete action to be disabled for channel chat")
	}
}

func TestChatListContextMenuDeleteEnabledForDM(t *testing.T) {
	menu := newChatListContextMenu(domain.Chat{Key: "dm:!12345678", Title: "Alice", Type: domain.ChatTypeDM}, nil)
	if len(menu.Items) != 1 {
		t.Fatalf("expected one menu item, got %d", len(menu.Items))
	}
	if menu.Items[0].Disabled {
		t.Fatalf("expected delete action to be enabled for dm chat")
	}
}

func TestChatsTabStoreDeleteSelectedDMClearsSelectionAndDisablesComposer(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	store := domain.NewChatStore()
	base := time.Now()
	store.Load(
		[]domain.Chat{
			{Key: "channel:0", Title: "General", Type: domain.ChatTypeChannel, UpdatedAt: base.Add(1 * time.Hour)},
			{Key: "dm:!0000002a", Title: "dm:!0000002a", Type: domain.ChatTypeDM, UpdatedAt: base},
		},
		map[string][]domain.ChatMessage{
			"dm:!0000002a": {
				{ChatKey: "dm:!0000002a", Body: "hello", Direction: domain.MessageDirectionIn, Status: domain.MessageStatusSent, At: base},
			},
		},
	)

	tab := newChatsTab(
		nil,
		store,
		sendTextFunc(func(_ string, _ string, _ radio.TextSendOptions) <-chan radio.SendResult {
			result := make(chan radio.SendResult, 1)
			result <- radio.SendResult{}
			close(result)

			return result
		}),
		nil,
		nil,
		nil,
		nil,
		"dm:!0000002a",
		nil,
		nil,
		nil,
	)
	_ = fynetest.NewTempWindow(t, tab)

	fyne.DoAndWait(func() {
		store.DeleteChat("dm:!0000002a")
	})

	entry := mustFindEntryByPlaceholder(t, tab, "Type message (max 200 bytes)")
	sendButton := mustFindButtonByText(t, tab, "Send")
	waitForCondition(t, func() bool {
		title := findLabelByPrefix(tab, "No chat selected")

		return title != nil && entry.Disabled() && sendButton.Disabled()
	})
}

func ptrInt(v int) *int {
	return &v
}

func ptrUint32(v uint32) *uint32 {
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

func findRichTextBySubstringAndWrapping(root fyne.CanvasObject, substring string, wrapping fyne.TextWrap) *widget.RichText {
	for _, object := range fynetest.LaidOutObjects(root) {
		richText, ok := object.(*widget.RichText)
		if !ok {
			continue
		}
		if strings.Contains(richTextSegmentsText(richText.Segments), substring) && richText.Wrapping == wrapping {
			return richText
		}
	}

	return nil
}
