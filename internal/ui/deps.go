package ui

import (
	"context"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	app_generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

// MessageSender sends user text messages through the active radio service.
type MessageSender interface {
	SendText(chatKey, text string, opts radio.TextSendOptions) <-chan radio.SendResult
}

// TracerouteAction starts traceroute requests for UI actions.
type TracerouteAction interface {
	StartTraceroute(ctx context.Context, target app.TracerouteTarget) (busmsg.TracerouteUpdate, error)
}

// NodeSettingsAction loads and saves node settings from UI.
type NodeSettingsAction interface {
	LoadUserSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeUserSettings, error)
	SaveUserSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeUserSettings) error
	LoadSecuritySettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeSecuritySettings, error)
	SaveSecuritySettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeSecuritySettings) error
	LoadLoRaSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeLoRaSettings, error)
	SaveLoRaSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeLoRaSettings) error
	LoadDeviceSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeDeviceSettings, error)
	SaveDeviceSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeDeviceSettings) error
	LoadPositionSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodePositionSettings, error)
	SavePositionSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodePositionSettings) error
	LoadPowerSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodePowerSettings, error)
	SavePowerSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodePowerSettings) error
	LoadDisplaySettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeDisplaySettings, error)
	SaveDisplaySettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeDisplaySettings) error
	LoadBluetoothSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeBluetoothSettings, error)
	SaveBluetoothSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeBluetoothSettings) error
	LoadNetworkSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error)
	SaveNetworkSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNetworkSettings) error
	LoadMQTTSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeMQTTSettings, error)
	SaveMQTTSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeMQTTSettings) error
	LoadSerialSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeSerialSettings, error)
	SaveSerialSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeSerialSettings) error
	LoadExternalNotificationSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error)
	SaveExternalNotificationSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeExternalNotificationSettings) error
	LoadStoreForwardSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error)
	SaveStoreForwardSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStoreForwardSettings) error
	LoadRangeTestSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeRangeTestSettings, error)
	SaveRangeTestSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeRangeTestSettings) error
	LoadTelemetrySettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeTelemetrySettings, error)
	SaveTelemetrySettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeTelemetrySettings) error
	LoadCannedMessageSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error)
	SaveCannedMessageSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeCannedMessageSettings) error
	LoadAudioSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAudioSettings, error)
	SaveAudioSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAudioSettings) error
	LoadRemoteHardwareSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error)
	SaveRemoteHardwareSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeRemoteHardwareSettings) error
	LoadNeighborInfoSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error)
	SaveNeighborInfoSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNeighborInfoSettings) error
	LoadAmbientLightingSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error)
	SaveAmbientLightingSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAmbientLightingSettings) error
	LoadDetectionSensorSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error)
	SaveDetectionSensorSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeDetectionSensorSettings) error
	LoadPaxcounterSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error)
	SavePaxcounterSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodePaxcounterSettings) error
	LoadStatusMessageSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error)
	SaveStatusMessageSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStatusMessageSettings) error
	LoadChannelSettings(ctx context.Context, target app.NodeSettingsTarget) (app.NodeChannelSettingsList, error)
	SaveChannelSettings(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeChannelSettingsList) error
	ExportProfile(ctx context.Context, target app.NodeSettingsTarget) (*app_generated.DeviceProfile, error)
	ImportProfile(ctx context.Context, target app.NodeSettingsTarget, profile *app_generated.DeviceProfile) error
	RebootNode(ctx context.Context, target app.NodeSettingsTarget) error
	ShutdownNode(ctx context.Context, target app.NodeSettingsTarget) error
	FactoryResetNode(ctx context.Context, target app.NodeSettingsTarget) error
	ResetNodeDB(ctx context.Context, target app.NodeSettingsTarget, preserveFavorites bool) error
}

// NodeOverviewAction handles node overview requests and history reads.
type NodeOverviewAction interface {
	RequestUserInfo(ctx context.Context, targetNodeID string, requester app.LocalNodeSnapshot) error
	RequestTelemetry(ctx context.Context, targetNodeID string, kind radio.TelemetryRequestKind) error
	ListTelemetryHistory(ctx context.Context, nodeID string, limit int) ([]domain.NodeTelemetryHistoryEntry, error)
	ListPositionHistory(ctx context.Context, nodeID string, limit int) ([]domain.NodePositionHistoryEntry, error)
	ListIdentityHistory(ctx context.Context, nodeID string, limit int) ([]domain.NodeIdentityHistoryEntry, error)
}

// NodeFavoriteAction handles marking remote nodes as favorite on local node DB.
type NodeFavoriteAction interface {
	SetFavorite(ctx context.Context, targetNodeID string, favorite bool) error
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
	LocalNodeSnapshot func() app.LocalNodeSnapshot
	CurrentConfig     func() config.AppConfig
	CurrentConnStatus func() (busmsg.ConnectionStatus, bool)
}

// ActionDependencies contains user-triggered operations invoked from UI.
type ActionDependencies struct {
	Sender                    MessageSender
	Traceroute                TracerouteAction
	OnSave                    func(cfg config.AppConfig) error
	OnChatSelected            func(chatKey string)
	OnDeleteDMChat            func(chatKey string) error
	OnMapViewportChanged      func(zoom, x, y int)
	OnMapDisplayConfigChanged func(cfg config.MapDisplayConfig)
	OnClearDB                 func() error
	OnClearCache              func() error
	OnStartUpdateChecker      func()
	OnQuit                    func()
	NodeSettings              NodeSettingsAction
	NodeOverview              NodeOverviewAction
	NodeFavorite              NodeFavoriteAction
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
