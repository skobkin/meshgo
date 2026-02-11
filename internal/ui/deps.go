package ui

import (
	"fyne.io/fyne/v2"

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
	BluetoothScanner BluetoothScanner
	Sender           interface {
		SendText(chatKey, text string) <-chan radio.SendResult
	}
	LocalNodeID             func() string
	OnSave                  func(cfg config.AppConfig) error
	OnChatSelected          func(chatKey string)
	OnClearDB               func() error
	OpenBluetoothSettings   func() error
	CurrentWindow           func() fyne.Window
	RunOnUI                 func(func())
	RunAsync                func(func())
	ShowBluetoothScanDialog func(window fyne.Window, devices []BluetoothScanDevice, onSelect func(BluetoothScanDevice))
	ShowErrorDialog         func(err error, window fyne.Window)
	ShowInfoDialog          func(title, message string, window fyne.Window)
	OnQuit                  func()
}
