package ui

import (
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
	if v := chatTypeLabel(domain.Chat{Key: "dm:!1234abcd", Type: domain.ChatTypeChannel}); v != "DM" {
		t.Fatalf("unexpected key-derived dm label: %q", v)
	}
}

func TestChatDisplayTitle_ChannelUsesChatTitle(t *testing.T) {
	chat := domain.Chat{Key: "channel:0", Title: "General", Type: domain.ChatTypeChannel}
	if got := chatDisplayTitle(chat, nil); got != "General" {
		t.Fatalf("unexpected channel title: %q", got)
	}
}

func TestChatDisplayTitle_DMUsesResolvedNodeName(t *testing.T) {
	chat := domain.Chat{Key: "dm:!1234abcd", Title: "dm:!1234abcd", Type: domain.ChatTypeDM}
	got := chatDisplayTitle(chat, func(nodeID string) string {
		if nodeID != "!1234abcd" {
			t.Fatalf("unexpected node id: %q", nodeID)
		}

		return "Alice"
	})
	if got != "Alice" {
		t.Fatalf("unexpected dm title: %q", got)
	}
}

func TestChatDisplayTitle_DMFallsBackToNodeID(t *testing.T) {
	chat := domain.Chat{Key: "dm:!1234abcd", Title: "dm:!1234abcd", Type: domain.ChatTypeDM}
	if got := chatDisplayTitle(chat, nil); got != "!1234abcd" {
		t.Fatalf("unexpected dm fallback title: %q", got)
	}
}

func TestChatDisplayTitle_DMFallsBackToCustomTitle(t *testing.T) {
	chat := domain.Chat{Key: "dm:!1234abcd", Title: "Alice", Type: domain.ChatTypeDM}
	if got := chatDisplayTitle(chat, func(string) string { return "" }); got != "Alice" {
		t.Fatalf("unexpected dm custom fallback title: %q", got)
	}
}

func TestChatDisplayTitle_DMFallsBackToCustomTitleWhenResolverReturnsNodeID(t *testing.T) {
	chat := domain.Chat{Key: "dm:!1234abcd", Title: "Alice", Type: domain.ChatTypeDM}
	if got := chatDisplayTitle(chat, func(string) string { return "!1234abcd" }); got != "Alice" {
		t.Fatalf("unexpected dm custom fallback title: %q", got)
	}
}

func TestChatDisplayTitle_DMDetectedByKeyWhenTypeIsChannel(t *testing.T) {
	chat := domain.Chat{Key: "dm:!1234abcd", Title: "dm:!1234abcd", Type: domain.ChatTypeChannel}
	got := chatDisplayTitle(chat, func(nodeID string) string {
		if nodeID != "!1234abcd" {
			t.Fatalf("unexpected node id: %q", nodeID)
		}

		return "Alice"
	})
	if got != "Alice" {
		t.Fatalf("unexpected dm title: %q", got)
	}
}
