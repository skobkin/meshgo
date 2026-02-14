package ui

import (
	"context"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

// MessageSender sends user text messages through the active radio service.
type MessageSender interface {
	SendText(chatKey, text string) <-chan radio.SendResult
}

// TracerouteAction starts traceroute requests for UI actions.
type TracerouteAction interface {
	StartTraceroute(ctx context.Context, target app.TracerouteTarget) (connectors.TracerouteUpdate, error)
}

// NodeSettingsAction loads and saves node settings from UI.
type NodeSettingsAction interface {
	LoadUserSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeUserSettings, error)
	SaveUserSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeUserSettings) error
	LoadSecuritySettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeSecuritySettings, error)
	SaveSecuritySettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeSecuritySettings) error
}

// DataDependencies contains read-only state consumed by UI tabs.
type DataDependencies struct {
	Config            config.AppConfig
	Paths             app.Paths
	ChatStore         *domain.ChatStore
	NodeStore         *domain.NodeStore
	Bus               bus.MessageBus
	LastSelectedChat  string
	LocalNodeID       func() string
	CurrentConfig     func() config.AppConfig
	CurrentConnStatus func() (connectors.ConnectionStatus, bool)
}

// ActionDependencies contains user-triggered operations invoked from UI.
type ActionDependencies struct {
	Sender               MessageSender
	Traceroute           TracerouteAction
	OnSave               func(cfg config.AppConfig) error
	OnChatSelected       func(chatKey string)
	OnMapViewportChanged func(zoom, x, y int)
	OnClearDB            func() error
	OnClearCache         func() error
	OnStartUpdateChecker func()
	OnQuit               func()
	NodeSettings         NodeSettingsAction
}

// PlatformDependencies contains OS-specific helpers used by UI actions.
type PlatformDependencies struct {
	BluetoothScanner      BluetoothScanner
	OpenBluetoothSettings func() error
}

// UIHooks overrides default UI interactions for tests and custom embedding.
type UIHooks struct {
	CurrentWindow           func() fyne.Window
	RunOnUI                 func(func())
	RunAsync                func(func())
	ShowBluetoothScanDialog func(window fyne.Window, devices []DiscoveredBluetoothDevice, onSelect func(DiscoveredBluetoothDevice))
	ShowErrorDialog         func(err error, window fyne.Window)
	ShowInfoDialog          func(title, message string, window fyne.Window)
}

// LaunchOptions controls initial window behavior at startup.
type LaunchOptions struct {
	StartHidden bool
}

// RuntimeDependencies is the complete dependency graph required to run the UI.
type RuntimeDependencies struct {
	Data     DataDependencies
	Actions  ActionDependencies
	Platform PlatformDependencies
	UIHooks  UIHooks
	Launch   LaunchOptions
}
