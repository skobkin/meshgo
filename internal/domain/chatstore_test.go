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
