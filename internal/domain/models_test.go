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
