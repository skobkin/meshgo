package domain

import "testing"

func TestShouldTransitionMessageStatus(t *testing.T) {
	tests := []struct {
		name    string
		current MessageStatus
		next    MessageStatus
		want    bool
	}{
		{name: "pending to sent", current: MessageStatusPending, next: MessageStatusSent, want: true},
		{name: "sent to acked", current: MessageStatusSent, next: MessageStatusAcked, want: true},
		{name: "failed to acked", current: MessageStatusFailed, next: MessageStatusAcked, want: true},
		{name: "acked to failed blocked", current: MessageStatusAcked, next: MessageStatusFailed, want: false},
		{name: "sent to pending blocked", current: MessageStatusSent, next: MessageStatusPending, want: false},
		{name: "sent to failed", current: MessageStatusSent, next: MessageStatusFailed, want: true},
	}

	for _, tc := range tests {
		if got := ShouldTransitionMessageStatus(tc.current, tc.next); got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestNodeDisplayName(t *testing.T) {
	tests := []struct {
		name string
		node Node
		want string
	}{
		{name: "long name preferred", node: Node{NodeID: "!11111111", LongName: "Long", ShortName: "Short"}, want: "Long"},
		{name: "short name fallback", node: Node{NodeID: "!11111111", ShortName: "Short"}, want: "Short"},
		{name: "node id fallback", node: Node{NodeID: "!11111111"}, want: "!11111111"},
		{name: "empty when all empty", node: Node{}, want: ""},
	}

	for _, tc := range tests {
		if got := NodeDisplayName(tc.node); got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestNodeDisplayNameByID(t *testing.T) {
	store := NewNodeStore()
	store.Upsert(Node{NodeID: "!11111111", LongName: "Long"})
	store.Upsert(Node{NodeID: "!22222222"})

	if got := NodeDisplayNameByID(store, "!11111111"); got != "Long" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := NodeDisplayNameByID(store, "!22222222"); got != "!22222222" {
		t.Fatalf("expected node id fallback, got %q", got)
	}
	if got := NodeDisplayNameByID(store, "!33333333"); got != "!33333333" {
		t.Fatalf("expected unresolved node id fallback, got %q", got)
	}
	if got := NodeDisplayNameByID(nil, "!44444444"); got != "!44444444" {
		t.Fatalf("expected nil store fallback, got %q", got)
	}
	if got := NodeDisplayNameByID(store, " "); got != "" {
		t.Fatalf("expected empty for empty node id, got %q", got)
	}
}
