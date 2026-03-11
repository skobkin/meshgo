package domain

import "testing"

func TestChatStore_AppendMessage_DedupesByDeviceMessageID(t *testing.T) {
	store := NewChatStore()

	store.AppendMessage(ChatMessage{
		ChatKey:         "dm:!1234abcd",
		DeviceMessageID: "100",
		Direction:       MessageDirectionOut,
		Body:            "hello",
		Status:          MessageStatusPending,
	})
	store.AppendMessage(ChatMessage{
		ChatKey:         "dm:!1234abcd",
		DeviceMessageID: "100",
		Direction:       MessageDirectionOut,
		Body:            "hello",
		Status:          MessageStatusPending,
	})

	msgs := store.Messages("dm:!1234abcd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after dedupe, got %d", len(msgs))
	}
	if msgs[0].Status != MessageStatusPending {
		t.Fatalf("expected status pending, got %v", msgs[0].Status)
	}
}

func TestChatStore_AppendMessage_DedupeMergesReplyAndEmoji(t *testing.T) {
	store := NewChatStore()

	store.AppendMessage(ChatMessage{
		ChatKey:         "dm:!1234abcd",
		DeviceMessageID: "100",
		Direction:       MessageDirectionIn,
		Body:            "👍",
		Status:          MessageStatusSent,
	})
	store.AppendMessage(ChatMessage{
		ChatKey:                "dm:!1234abcd",
		DeviceMessageID:        "100",
		ReplyToDeviceMessageID: "99",
		Emoji:                  1,
		Direction:              MessageDirectionIn,
		Body:                   "👍",
		Status:                 MessageStatusSent,
	})

	msgs := store.Messages("dm:!1234abcd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after dedupe, got %d", len(msgs))
	}
	if msgs[0].ReplyToDeviceMessageID != "99" {
		t.Fatalf("expected reply id to be merged, got %q", msgs[0].ReplyToDeviceMessageID)
	}
	if msgs[0].Emoji != 1 {
		t.Fatalf("expected emoji to be merged, got %d", msgs[0].Emoji)
	}
}

func TestChatStore_UpdateMessageStatusByDeviceID_SetsFailedReason(t *testing.T) {
	store := NewChatStore()
	store.AppendMessage(ChatMessage{
		ChatKey:         "dm:!1234abcd",
		DeviceMessageID: "100",
		Direction:       MessageDirectionOut,
		Body:            "hello",
		Status:          MessageStatusPending,
	})

	store.UpdateMessageStatusByDeviceID("100", MessageStatusFailed, "NO_ROUTE")

	msgs := store.Messages("dm:!1234abcd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Status != MessageStatusFailed {
		t.Fatalf("expected failed status, got %v", msgs[0].Status)
	}
	if msgs[0].StatusReason != "NO_ROUTE" {
		t.Fatalf("expected failed reason to be set, got %q", msgs[0].StatusReason)
	}
}

func TestChatStore_UpdateMessageStatusByDeviceID_ClearsReasonOnAck(t *testing.T) {
	store := NewChatStore()
	store.AppendMessage(ChatMessage{
		ChatKey:         "dm:!1234abcd",
		DeviceMessageID: "100",
		Direction:       MessageDirectionOut,
		Body:            "hello",
		Status:          MessageStatusFailed,
		StatusReason:    "NO_ROUTE",
	})

	store.UpdateMessageStatusByDeviceID("100", MessageStatusAcked, "")

	msgs := store.Messages("dm:!1234abcd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Status != MessageStatusAcked {
		t.Fatalf("expected acked status, got %v", msgs[0].Status)
	}
	if msgs[0].StatusReason != "" {
		t.Fatalf("expected reason to be cleared, got %q", msgs[0].StatusReason)
	}
}

func TestChatTitleByKey(t *testing.T) {
	store := NewChatStore()
	store.UpsertChat(Chat{Key: "channel:0", Title: "General"})
	store.UpsertChat(Chat{Key: "channel:1", Title: ""})

	if got := ChatTitleByKey(store, "channel:0"); got != "General" {
		t.Fatalf("expected explicit title, got %q", got)
	}
	if got := ChatTitleByKey(store, "channel:1"); got != "channel:1" {
		t.Fatalf("expected chat key fallback, got %q", got)
	}
	if got := ChatTitleByKey(store, "channel:2"); got != "channel:2" {
		t.Fatalf("expected unknown chat key fallback, got %q", got)
	}
	if got := ChatTitleByKey(nil, "channel:3"); got != "channel:3" {
		t.Fatalf("expected nil store fallback, got %q", got)
	}
}

func TestChatStoreDeleteChat_RemovesChatAndMessages(t *testing.T) {
	store := NewChatStore()
	store.UpsertChat(Chat{Key: "dm:!1234abcd", Title: "Alice", Type: ChatTypeDM})
	store.UpsertChat(Chat{Key: "channel:0", Title: "General", Type: ChatTypeChannel})
	store.AppendMessage(ChatMessage{ChatKey: "dm:!1234abcd", Body: "hello", Direction: MessageDirectionIn})
	store.AppendMessage(ChatMessage{ChatKey: "channel:0", Body: "world", Direction: MessageDirectionIn})

	store.DeleteChat("dm:!1234abcd")

	if _, ok := store.ChatByKey("dm:!1234abcd"); ok {
		t.Fatalf("expected dm chat to be removed")
	}
	if got := len(store.Messages("dm:!1234abcd")); got != 0 {
		t.Fatalf("expected dm messages to be removed, got %d", got)
	}
	if _, ok := store.ChatByKey("channel:0"); !ok {
		t.Fatalf("expected other chat to remain")
	}
	if got := len(store.Messages("channel:0")); got != 1 {
		t.Fatalf("expected other chat messages to remain, got %d", got)
	}
}

func TestChatStoreDeleteChat_BlankOrMissingKeyIsNoop(t *testing.T) {
	store := NewChatStore()
	store.UpsertChat(Chat{Key: "channel:0", Title: "General", Type: ChatTypeChannel})
	store.AppendMessage(ChatMessage{ChatKey: "channel:0", Body: "hello", Direction: MessageDirectionIn})

	store.DeleteChat("")
	store.DeleteChat("missing")

	if _, ok := store.ChatByKey("channel:0"); !ok {
		t.Fatalf("expected chat to remain")
	}
	if got := len(store.Messages("channel:0")); got != 1 {
		t.Fatalf("expected messages to remain, got %d", got)
	}
}
