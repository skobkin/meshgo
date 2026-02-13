package domain

import "testing"

func TestChatTypeForKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want ChatType
	}{
		{name: "dm key", key: "dm:!11111111", want: ChatTypeDM},
		{name: "channel key", key: "channel:0", want: ChatTypeChannel},
		{name: "unknown key defaults to channel", key: "custom", want: ChatTypeChannel},
	}

	for _, tt := range tests {
		if got := ChatTypeForKey(tt.key); got != tt.want {
			t.Fatalf("%s: expected %v, got %v", tt.name, tt.want, got)
		}
	}
}

func TestNodeIDFromDMChatKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "dm key", key: "dm:!11111111", want: "!11111111"},
		{name: "dm key with spaces", key: "  dm:!22222222  ", want: "!22222222"},
		{name: "channel key", key: "channel:0", want: ""},
		{name: "empty", key: "", want: ""},
	}

	for _, tt := range tests {
		if got := NodeIDFromDMChatKey(tt.key); got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}
