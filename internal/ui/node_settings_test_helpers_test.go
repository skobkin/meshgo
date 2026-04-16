package ui

import (
	"context"
	"sync/atomic"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type nodeSettingsActionSpy struct {
	loadSecurityCalls      atomic.Int32
	loadLoRaCalls          atomic.Int32
	loadChannelsCalls      atomic.Int32
	loadPositionCalls      atomic.Int32
	loadPowerCalls         atomic.Int32
	loadDisplayCalls       atomic.Int32
	loadBluetoothCalls     atomic.Int32
	loadNetworkCalls       atomic.Int32
	loadMQTTCalls          atomic.Int32
	loadSerialCalls        atomic.Int32
	loadExternalCalls      atomic.Int32
	loadStoreForwardCalls  atomic.Int32
	loadRangeTestCalls     atomic.Int32
	loadTelemetryCalls     atomic.Int32
	loadCannedMessageCalls atomic.Int32
	loadAudioCalls         atomic.Int32
	loadRemoteHardwareCall atomic.Int32
	loadNeighborInfoCalls  atomic.Int32
	loadAmbientLightCalls  atomic.Int32
	loadDetectionCalls     atomic.Int32
	loadPaxCalls           atomic.Int32
	loadStatusMessageCalls atomic.Int32
}

func newNodeSettingsRuntimeDeps(spy *nodeSettingsActionSpy) RuntimeDependencies {
	return RuntimeDependencies{
		Data: DataDependencies{
			LocalNodeID: func() string { return "!00000001" },
			CurrentConnStatus: func() (busmsg.ConnectionStatus, bool) {
				return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
			},
		},
		Actions: ActionDependencies{
			NodeSettings: spy,
		},
	}
}

func (s *nodeSettingsActionSpy) LoadUserSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeUserSettings, error) {
	return app.NodeUserSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveUserSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeUserSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadSecuritySettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeSecuritySettings, error) {
	s.loadSecurityCalls.Add(1)

	return app.NodeSecuritySettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveSecuritySettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeSecuritySettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadLoRaSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeLoRaSettings, error) {
	s.loadLoRaCalls.Add(1)

	return app.NodeLoRaSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveLoRaSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeLoRaSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadChannelSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeChannelSettingsList, error) {
	s.loadChannelsCalls.Add(1)

	return app.NodeChannelSettingsList{NodeID: target.NodeID, MaxSlots: app.NodeChannelMaxSlots}, nil
}

func (s *nodeSettingsActionSpy) SaveChannelSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeChannelSettingsList) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadDeviceSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeDeviceSettings, error) {
	return app.NodeDeviceSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveDeviceSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeDeviceSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadPositionSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodePositionSettings, error) {
	s.loadPositionCalls.Add(1)

	return app.NodePositionSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SavePositionSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodePositionSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadPowerSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodePowerSettings, error) {
	s.loadPowerCalls.Add(1)

	return app.NodePowerSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SavePowerSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodePowerSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadDisplaySettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeDisplaySettings, error) {
	s.loadDisplayCalls.Add(1)

	return app.NodeDisplaySettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveDisplaySettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeDisplaySettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadBluetoothSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeBluetoothSettings, error) {
	s.loadBluetoothCalls.Add(1)

	return app.NodeBluetoothSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveBluetoothSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeBluetoothSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadNetworkSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
	s.loadNetworkCalls.Add(1)

	return app.NodeNetworkSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveNetworkSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeNetworkSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadMQTTSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeMQTTSettings, error) {
	s.loadMQTTCalls.Add(1)

	return app.NodeMQTTSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveMQTTSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeMQTTSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadSerialSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeSerialSettings, error) {
	s.loadSerialCalls.Add(1)

	return app.NodeSerialSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveSerialSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeSerialSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadExternalNotificationSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error) {
	s.loadExternalCalls.Add(1)

	return app.NodeExternalNotificationSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveExternalNotificationSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeExternalNotificationSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadStoreForwardSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error) {
	s.loadStoreForwardCalls.Add(1)

	return app.NodeStoreForwardSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveStoreForwardSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeStoreForwardSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadRangeTestSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeRangeTestSettings, error) {
	s.loadRangeTestCalls.Add(1)

	return app.NodeRangeTestSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveRangeTestSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeRangeTestSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadTelemetrySettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeTelemetrySettings, error) {
	s.loadTelemetryCalls.Add(1)

	return app.NodeTelemetrySettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveTelemetrySettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeTelemetrySettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadCannedMessageSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error) {
	s.loadCannedMessageCalls.Add(1)

	return app.NodeCannedMessageSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveCannedMessageSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeCannedMessageSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadAudioSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeAudioSettings, error) {
	s.loadAudioCalls.Add(1)

	return app.NodeAudioSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveAudioSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeAudioSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadRemoteHardwareSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error) {
	s.loadRemoteHardwareCall.Add(1)

	return app.NodeRemoteHardwareSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveRemoteHardwareSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeRemoteHardwareSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadNeighborInfoSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
	s.loadNeighborInfoCalls.Add(1)

	return app.NodeNeighborInfoSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveNeighborInfoSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeNeighborInfoSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadAmbientLightingSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
	s.loadAmbientLightCalls.Add(1)

	return app.NodeAmbientLightingSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveAmbientLightingSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeAmbientLightingSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadDetectionSensorSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error) {
	s.loadDetectionCalls.Add(1)

	return app.NodeDetectionSensorSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveDetectionSensorSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeDetectionSensorSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadPaxcounterSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error) {
	s.loadPaxCalls.Add(1)

	return app.NodePaxcounterSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SavePaxcounterSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodePaxcounterSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadStatusMessageSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error) {
	s.loadStatusMessageCalls.Add(1)

	return app.NodeStatusMessageSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveStatusMessageSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeStatusMessageSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) ExportProfile(_ context.Context, target app.NodeSettingsTarget) (*generated.DeviceProfile, error) {
	return &generated.DeviceProfile{LongName: &target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) ImportProfile(_ context.Context, _ app.NodeSettingsTarget, _ *generated.DeviceProfile) error {
	return nil
}

func (s *nodeSettingsActionSpy) RebootNode(_ context.Context, _ app.NodeSettingsTarget) error {
	return nil
}

func (s *nodeSettingsActionSpy) ShutdownNode(_ context.Context, _ app.NodeSettingsTarget) error {
	return nil
}

func (s *nodeSettingsActionSpy) FactoryResetNode(_ context.Context, _ app.NodeSettingsTarget) error {
	return nil
}

func (s *nodeSettingsActionSpy) ResetNodeDB(_ context.Context, _ app.NodeSettingsTarget, _ bool) error {
	return nil
}

func (s *nodeSettingsActionSpy) SecurityLoadCalls() int  { return int(s.loadSecurityCalls.Load()) }
func (s *nodeSettingsActionSpy) LoRaLoadCalls() int      { return int(s.loadLoRaCalls.Load()) }
func (s *nodeSettingsActionSpy) PositionLoadCalls() int  { return int(s.loadPositionCalls.Load()) }
func (s *nodeSettingsActionSpy) ChannelsLoadCalls() int  { return int(s.loadChannelsCalls.Load()) }
func (s *nodeSettingsActionSpy) PowerLoadCalls() int     { return int(s.loadPowerCalls.Load()) }
func (s *nodeSettingsActionSpy) DisplayLoadCalls() int   { return int(s.loadDisplayCalls.Load()) }
func (s *nodeSettingsActionSpy) BluetoothLoadCalls() int { return int(s.loadBluetoothCalls.Load()) }
func (s *nodeSettingsActionSpy) NetworkLoadCalls() int   { return int(s.loadNetworkCalls.Load()) }
func (s *nodeSettingsActionSpy) MQTTLoadCalls() int      { return int(s.loadMQTTCalls.Load()) }
func (s *nodeSettingsActionSpy) SerialLoadCalls() int    { return int(s.loadSerialCalls.Load()) }
func (s *nodeSettingsActionSpy) ExternalLoadCalls() int  { return int(s.loadExternalCalls.Load()) }
func (s *nodeSettingsActionSpy) StoreForwardLoadCalls() int {
	return int(s.loadStoreForwardCalls.Load())
}
func (s *nodeSettingsActionSpy) RangeTestLoadCalls() int { return int(s.loadRangeTestCalls.Load()) }
func (s *nodeSettingsActionSpy) TelemetryLoadCalls() int { return int(s.loadTelemetryCalls.Load()) }
func (s *nodeSettingsActionSpy) CannedMessageLoadCalls() int {
	return int(s.loadCannedMessageCalls.Load())
}
func (s *nodeSettingsActionSpy) AudioLoadCalls() int { return int(s.loadAudioCalls.Load()) }
func (s *nodeSettingsActionSpy) RemoteHardwareLoadCalls() int {
	return int(s.loadRemoteHardwareCall.Load())
}
func (s *nodeSettingsActionSpy) NeighborInfoLoadCalls() int {
	return int(s.loadNeighborInfoCalls.Load())
}
func (s *nodeSettingsActionSpy) AmbientLightingLoadCalls() int {
	return int(s.loadAmbientLightCalls.Load())
}
func (s *nodeSettingsActionSpy) DetectionLoadCalls() int { return int(s.loadDetectionCalls.Load()) }
func (s *nodeSettingsActionSpy) PaxLoadCalls() int       { return int(s.loadPaxCalls.Load()) }
func (s *nodeSettingsActionSpy) StatusMessageLoadCalls() int {
	return int(s.loadStatusMessageCalls.Load())
}
