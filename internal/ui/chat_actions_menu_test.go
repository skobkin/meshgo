package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestCanReactToMessage(t *testing.T) {
	cases := []struct {
		name    string
		message domain.ChatMessage
		want    bool
	}{
		{
			name: "incoming ack-required message",
			message: domain.ChatMessage{
				DeviceMessageID: "abc123",
				ChatKey:         "channel:0",
				Direction:       domain.MessageDirectionIn,
			},
			want: true,
		},
		{
			name: "outgoing sent message",
			message: domain.ChatMessage{
				DeviceMessageID: "abc123",
				ChatKey:         "channel:0",
				Direction:       domain.MessageDirectionOut,
				Status:          domain.MessageStatusSent,
			},
			want: true,
		},
		{
			name: "outgoing still pending",
			message: domain.ChatMessage{
				DeviceMessageID: "abc123",
				ChatKey:         "channel:0",
				Direction:       domain.MessageDirectionOut,
				Status:          domain.MessageStatusPending,
			},
			want: false,
		},
		{
			name: "no device message id",
			message: domain.ChatMessage{
				ChatKey:   "channel:0",
				Direction: domain.MessageDirectionIn,
			},
			want: false,
		},
		{
			name: "reaction message itself",
			message: domain.ChatMessage{
				DeviceMessageID:        "abc123",
				ReplyToDeviceMessageID: "target456",
				Emoji:                  1,
				ChatKey:                "channel:0",
				Direction:              domain.MessageDirectionIn,
			},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canReactToMessage(tc.message); got != tc.want {
				t.Fatalf("canReactToMessage = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewChatMessageContextMenu_Reactions(t *testing.T) {
	cases := []struct {
		name              string
		message           domain.ChatMessage
		wantReplyDisabled bool
		wantReactDisabled bool
	}{
		{
			name: "reactable message has both items enabled",
			message: domain.ChatMessage{
				DeviceMessageID: "abc",
				ChatKey:         "channel:0",
				Body:            "hi",
				Direction:       domain.MessageDirectionIn,
			},
			wantReplyDisabled: false,
			wantReactDisabled: false,
		},
		{
			name: "reaction message disables Add reaction only",
			message: domain.ChatMessage{
				DeviceMessageID:        "abc",
				ReplyToDeviceMessageID: "target",
				Emoji:                  1,
				ChatKey:                "channel:0",
				Body:                   "❤️",
				Direction:              domain.MessageDirectionIn,
			},
			wantReplyDisabled: true,
			wantReactDisabled: true,
		},
		{
			name: "outgoing pending disables Add reaction only",
			message: domain.ChatMessage{
				DeviceMessageID: "abc",
				ChatKey:         "channel:0",
				Body:            "draft",
				Direction:       domain.MessageDirectionOut,
				Status:          domain.MessageStatusPending,
			},
			wantReplyDisabled: false,
			wantReactDisabled: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotActions []ChatAction
			menu := newChatMessageContextMenu(tc.message, func(_ domain.ChatMessage, action ChatAction) {
				gotActions = append(gotActions, action)
			})

			if got, want := len(menu.Items), 2; got != want {
				t.Fatalf("expected %d menu items, got %d", want, got)
			}

			reply := menu.Items[0]
			react := menu.Items[1]
			if reply.Label != "Reply" {
				t.Fatalf("first item: expected label %q, got %q", "Reply", reply.Label)
			}
			if react.Label != "Add reaction" {
				t.Fatalf("second item: expected label %q, got %q", "Add reaction", react.Label)
			}
			if reply.Disabled != tc.wantReplyDisabled {
				t.Fatalf("Reply.Disabled = %v, want %v", reply.Disabled, tc.wantReplyDisabled)
			}
			if react.Disabled != tc.wantReactDisabled {
				t.Fatalf("Add reaction.Disabled = %v, want %v", react.Disabled, tc.wantReactDisabled)
			}

			// Click simulation is only valid for enabled items; the
			// disabled-state cases are covered by the Disabled flag
			// assertions above.
			if !reply.Disabled {
				reply.Action()
				if len(gotActions) != 1 || gotActions[0] != ChatActionReply {
					t.Fatalf("expected Reply action to fire, got %v", gotActions)
				}
			}
			if !react.Disabled {
				react.Action()
				if len(gotActions) == 0 || gotActions[len(gotActions)-1] != ChatActionReact {
					t.Fatalf("expected Add reaction action to fire, got %v", gotActions)
				}
			}
		})
	}
}

func TestNewChatMessageContextMenu_TitleTruncation(t *testing.T) {
	// Body longer than the truncation cap should still produce a
	// finite-length title with the ellipsis suffix, so the menu never
	// overflows the screen on a long quoted reply.
	long := make([]byte, chatMenuTitleMaxLen*3)
	for i := range long {
		long[i] = 'a'
	}
	menu := newChatMessageContextMenu(domain.ChatMessage{
		DeviceMessageID: "abc",
		ChatKey:         "channel:0",
		Body:            string(long),
		Direction:       domain.MessageDirectionIn,
	}, nil)
	title := menu.Label
	if got := len([]rune(title)); got > chatMenuTitleMaxLen {
		t.Fatalf("expected menu title <= %d runes, got %d (%q)", chatMenuTitleMaxLen, got, title)
	}
	if title[len(title)-len("…"):] != "…" {
		t.Fatalf("expected menu title to end with ellipsis, got %q", title)
	}

	// Empty body falls back to the literal "Message" title.
	menu2 := newChatMessageContextMenu(domain.ChatMessage{
		DeviceMessageID: "abc",
		ChatKey:         "channel:0",
		Direction:       domain.MessageDirectionIn,
	}, nil)
	if menu2.Label != "Message" {
		t.Fatalf("expected fallback title %q, got %q", "Message", menu2.Label)
	}
}
