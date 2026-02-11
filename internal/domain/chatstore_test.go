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
