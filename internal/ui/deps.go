package ui

import (
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

type Dependencies struct {
	Config           config.AppConfig
	ChatStore        *domain.ChatStore
	NodeStore        *domain.NodeStore
	Bus              bus.MessageBus
	LastSelectedChat string
	Sender           interface {
		SendText(chatKey, text string) <-chan radio.SendResult
	}
	LocalNodeID    func() string
	OnSave         func(cfg config.AppConfig) error
	OnChatSelected func(chatKey string)
	OnClearDB      func() error
	OnQuit         func()
}
