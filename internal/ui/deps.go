package ui

import (
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/transport"
)

type Dependencies struct {
	Config    config.AppConfig
	ChatStore *domain.ChatStore
	NodeStore *domain.NodeStore
	Bus       bus.MessageBus
	Sender    interface {
		SendText(chatKey, text string) <-chan radio.SendResult
	}
	IPTransport *transport.IPTransport
	OnSave      func(cfg config.AppConfig) error
	OnQuit      func()
}
