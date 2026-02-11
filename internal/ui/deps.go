package ui

import (
	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

type MessageSender interface {
	SendText(chatKey, text string) <-chan radio.SendResult
}

type DataDependencies struct {
	Config            config.AppConfig
	ChatStore         *domain.ChatStore
	NodeStore         *domain.NodeStore
	Bus               bus.MessageBus
	LastSelectedChat  string
	LocalNodeID       func() string
	CurrentConnStatus func() (connectors.ConnectionStatus, bool)
}

type ActionDependencies struct {
	Sender         MessageSender
	OnSave         func(cfg config.AppConfig) error
	OnChatSelected func(chatKey string)
	OnClearDB      func() error
	OnQuit         func()
}

type PlatformDependencies struct {
	BluetoothScanner      BluetoothScanner
	OpenBluetoothSettings func() error
}

type UIHooks struct {
	CurrentWindow           func() fyne.Window
	RunOnUI                 func(func())
	RunAsync                func(func())
	ShowBluetoothScanDialog func(window fyne.Window, devices []BluetoothScanDevice, onSelect func(BluetoothScanDevice))
	ShowErrorDialog         func(err error, window fyne.Window)
	ShowInfoDialog          func(title, message string, window fyne.Window)
}

type LaunchOptions struct {
	StartHidden bool
}

type RuntimeDependencies struct {
	Data     DataDependencies
	Actions  ActionDependencies
	Platform PlatformDependencies
	UIHooks  UIHooks
	Launch   LaunchOptions
}
