package ui

import "github.com/skobkin/meshgo/internal/radio"

type MessageSender interface {
	SendText(chatKey, text string) <-chan radio.SendResult
}
