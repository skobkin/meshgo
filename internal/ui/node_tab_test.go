package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type nodeSettingsActionSpy struct {
	loadSecurityCalls  atomic.Int32
	loadLoRaCalls      atomic.Int32
	loadChannelsCalls  atomic.Int32
	loadPositionCalls  atomic.Int32
	loadPowerCalls     atomic.Int32
	loadDisplayCalls   atomic.Int32
	loadBluetoothCalls atomic.Int32
	loadNetworkCalls   atomic.Int32
	loadMQTTCalls      atomic.Int32
	loadSerialCalls    atomic.Int32
	loadExternalCalls  atomic.Int32
	loadRangeTestCalls atomic.Int32
	loadTelemetryCalls atomic.Int32
	loadAudioCalls     atomic.Int32
	loadDetectionCalls atomic.Int32
	loadPaxCalls       atomic.Int32
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
	return app.NodeRemoteHardwareSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveRemoteHardwareSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeRemoteHardwareSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadNeighborInfoSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
	return app.NodeNeighborInfoSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveNeighborInfoSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeNeighborInfoSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) LoadAmbientLightingSettings(_ context.Context, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
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
	return app.NodeStatusMessageSettings{NodeID: target.NodeID}, nil
}

func (s *nodeSettingsActionSpy) SaveStatusMessageSettings(_ context.Context, _ app.NodeSettingsTarget, _ app.NodeStatusMessageSettings) error {
	return nil
}

func (s *nodeSettingsActionSpy) SecurityLoadCalls() int {
	return int(s.loadSecurityCalls.Load())
}

func (s *nodeSettingsActionSpy) LoRaLoadCalls() int {
	return int(s.loadLoRaCalls.Load())
}

func (s *nodeSettingsActionSpy) PositionLoadCalls() int {
	return int(s.loadPositionCalls.Load())
}

func (s *nodeSettingsActionSpy) ChannelsLoadCalls() int {
	return int(s.loadChannelsCalls.Load())
}

func (s *nodeSettingsActionSpy) PowerLoadCalls() int {
	return int(s.loadPowerCalls.Load())
}

func (s *nodeSettingsActionSpy) DisplayLoadCalls() int {
	return int(s.loadDisplayCalls.Load())
}

func (s *nodeSettingsActionSpy) BluetoothLoadCalls() int {
	return int(s.loadBluetoothCalls.Load())
}

func (s *nodeSettingsActionSpy) NetworkLoadCalls() int {
	return int(s.loadNetworkCalls.Load())
}

func (s *nodeSettingsActionSpy) MQTTLoadCalls() int {
	return int(s.loadMQTTCalls.Load())
}

func (s *nodeSettingsActionSpy) SerialLoadCalls() int {
	return int(s.loadSerialCalls.Load())
}

func (s *nodeSettingsActionSpy) ExternalLoadCalls() int {
	return int(s.loadExternalCalls.Load())
}

func (s *nodeSettingsActionSpy) RangeTestLoadCalls() int {
	return int(s.loadRangeTestCalls.Load())
}

func (s *nodeSettingsActionSpy) TelemetryLoadCalls() int {
	return int(s.loadTelemetryCalls.Load())
}

func (s *nodeSettingsActionSpy) AudioLoadCalls() int {
	return int(s.loadAudioCalls.Load())
}

func (s *nodeSettingsActionSpy) DetectionLoadCalls() int {
	return int(s.loadDetectionCalls.Load())
}

func (s *nodeSettingsActionSpy) PaxLoadCalls() int {
	return int(s.loadPaxCalls.Load())
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

func TestParseSecurityAdminKeysInput_Valid(t *testing.T) {
	a := bytes.Repeat([]byte{0x11}, 32)
	b := bytes.Repeat([]byte{0x22}, 32)
	input := base64.StdEncoding.EncodeToString(a) + "\n" + base64.RawStdEncoding.EncodeToString(b)

	keys, err := parseSecurityAdminKeysInput(input)
	if err != nil {
		t.Fatalf("parse admin keys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("unexpected keys count: %d", len(keys))
	}
	if !bytes.Equal(keys[0], a) {
		t.Fatalf("unexpected first key")
	}
	if !bytes.Equal(keys[1], b) {
		t.Fatalf("unexpected second key")
	}
}

func TestParseSecurityAdminKeysInput_TooManyKeys(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x11}, 32))
	input := key + "\n" + key + "\n" + key + "\n" + key

	_, err := parseSecurityAdminKeysInput(input)
	if err == nil {
		t.Fatalf("expected error for too many keys")
	}
}

func TestParseSecurityAdminKeysInput_InvalidKeyLength(t *testing.T) {
	input := base64.StdEncoding.EncodeToString([]byte("short"))

	_, err := parseSecurityAdminKeysInput(input)
	if err == nil {
		t.Fatalf("expected error for invalid key length")
	}
}

func TestParseSecurityAdminKeysInput_DuplicateKeys(t *testing.T) {
	key := bytes.Repeat([]byte{0x11}, 32)
	encoded := base64.StdEncoding.EncodeToString(key)
	input := encoded + "\n" + encoded

	_, err := parseSecurityAdminKeysInput(input)
	if err == nil {
		t.Fatalf("expected error for duplicate keys")
	}
}

func TestParseSecurityAdminKeysInput_DuplicateKeysDifferentEncoding(t *testing.T) {
	key := bytes.Repeat([]byte{0x22}, 32)
	std := base64.StdEncoding.EncodeToString(key)
	raw := base64.RawStdEncoding.EncodeToString(key)
	input := std + "\n" + raw

	_, err := parseSecurityAdminKeysInput(input)
	if err == nil {
		t.Fatalf("expected error for duplicate decoded keys")
	}
}

func TestNodeSettingsProgress(t *testing.T) {
	if got := nodeSettingsProgress(-1, 3); got != 0 {
		t.Fatalf("expected clamped 0 for negative completed, got %f", got)
	}
	if got := nodeSettingsProgress(4, 3); got != 1 {
		t.Fatalf("expected clamped 1 for over-complete, got %f", got)
	}
	if got := nodeSettingsProgress(1, 4); got != 0.25 {
		t.Fatalf("expected 0.25 progress, got %f", got)
	}
}

func TestNodeTabSecuritySettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.SecurityLoadCalls(); got != 0 {
		t.Fatalf("expected no eager security load before selecting Security tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Security")
	waitForCondition(t, func() bool {
		return spy.SecurityLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "LoRa")
	mustSelectAppTabByText(t, tab, "Security")
	time.Sleep(100 * time.Millisecond)
	if got := spy.SecurityLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial security load, got %d", got)
	}
}

func TestNodeTabContainsNodeOverviewAsFirstTopLevelTab(t *testing.T) {
	tab := newNodeTabWithOnShow(RuntimeDependencies{})
	topTabs, ok := tab.(*container.AppTabs)
	if !ok {
		t.Fatalf("expected node tab root to be app tabs, got %T", tab)
	}
	if len(topTabs.Items) == 0 {
		t.Fatalf("expected top-level tabs")
	}
	if topTabs.Items[0].Text != "Node overview" {
		t.Fatalf("expected first top-level tab to be Node overview, got %q", topTabs.Items[0].Text)
	}
}

func TestNodeTabLoRaSettingsLoadStartsOnRadioConfigurationTabSelect(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTabWithOnShow(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.LoRaLoadCalls(); got != 0 {
		t.Fatalf("expected no eager LoRa load before selecting Radio configuration tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Radio configuration")
	waitForCondition(t, func() bool { return spy.LoRaLoadCalls() == 1 })
	time.Sleep(100 * time.Millisecond)
	if got := spy.LoRaLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial LoRa load after selecting Radio configuration tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Security")
	mustSelectAppTabByText(t, tab, "LoRa")
	time.Sleep(100 * time.Millisecond)
	if got := spy.LoRaLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial LoRa load after Node tab open, got %d", got)
	}
}

func TestNodeTabChannelsSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.ChannelsLoadCalls(); got != 0 {
		t.Fatalf("expected no eager channels load before selecting Channels tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Channels")
	waitForCondition(t, func() bool {
		return spy.ChannelsLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Security")
	mustSelectAppTabByText(t, tab, "Channels")
	time.Sleep(100 * time.Millisecond)
	if got := spy.ChannelsLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial channels load, got %d", got)
	}
}

func TestNodeTabPositionSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.PositionLoadCalls(); got != 0 {
		t.Fatalf("expected no eager position load before selecting Position tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Position")
	waitForCondition(t, func() bool {
		return spy.PositionLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Device")
	mustSelectAppTabByText(t, tab, "Position")
	time.Sleep(100 * time.Millisecond)
	if got := spy.PositionLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial position load, got %d", got)
	}
}

func TestNodeTabPowerSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.PowerLoadCalls(); got != 0 {
		t.Fatalf("expected no eager power load before selecting Power tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Power")
	waitForCondition(t, func() bool {
		return spy.PowerLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Device")
	mustSelectAppTabByText(t, tab, "Power")
	time.Sleep(100 * time.Millisecond)
	if got := spy.PowerLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial power load, got %d", got)
	}
}

func TestNodeTabDisplaySettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.DisplayLoadCalls(); got != 0 {
		t.Fatalf("expected no eager display load before selecting Display tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Display")
	waitForCondition(t, func() bool {
		return spy.DisplayLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Device")
	mustSelectAppTabByText(t, tab, "Display")
	time.Sleep(100 * time.Millisecond)
	if got := spy.DisplayLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial display load, got %d", got)
	}
}

func TestNodeTabNetworkSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.NetworkLoadCalls(); got != 0 {
		t.Fatalf("expected no eager network load before selecting Network tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Network")
	waitForCondition(t, func() bool {
		return spy.NetworkLoadCalls() == 1
	})
}

func TestNodeTabBluetoothSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.BluetoothLoadCalls(); got != 0 {
		t.Fatalf("expected no eager bluetooth load before selecting Bluetooth tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Bluetooth")
	waitForCondition(t, func() bool {
		return spy.BluetoothLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Display")
	mustSelectAppTabByText(t, tab, "Bluetooth")
	time.Sleep(100 * time.Millisecond)
	if got := spy.BluetoothLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial bluetooth load, got %d", got)
	}
}

func TestNodeTabMQTTSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.MQTTLoadCalls(); got != 0 {
		t.Fatalf("expected no eager MQTT load before selecting Module/MQTT tabs, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Module configuration")
	mustSelectAppTabByText(t, tab, "MQTT")
	waitForCondition(t, func() bool {
		return spy.MQTTLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "Serial")
	mustSelectAppTabByText(t, tab, "MQTT")
	time.Sleep(100 * time.Millisecond)
	if got := spy.MQTTLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial MQTT load, got %d", got)
	}
}

func TestNodeTabSerialSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.SerialLoadCalls(); got != 0 {
		t.Fatalf("expected no eager serial load before selecting Module/Serial tabs, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Module configuration")
	mustSelectAppTabByText(t, tab, "Serial")
	waitForCondition(t, func() bool {
		return spy.SerialLoadCalls() == 1
	})
}

func TestNodeTabRangeTestSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.RangeTestLoadCalls(); got != 0 {
		t.Fatalf("expected no eager range test load before selecting Module/Range test tabs, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Module configuration")
	mustSelectAppTabByText(t, tab, "Range test")
	waitForCondition(t, func() bool {
		return spy.RangeTestLoadCalls() == 1
	})

	mustSelectAppTabByText(t, tab, "MQTT")
	mustSelectAppTabByText(t, tab, "Range test")
	time.Sleep(100 * time.Millisecond)
	if got := spy.RangeTestLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial range test load, got %d", got)
	}
}

func TestNodeTabIncludesImportExportAndMaintenanceTabs(t *testing.T) {
	tab := newNodeTabWithOnShow(RuntimeDependencies{})
	topTabs, ok := tab.(*container.AppTabs)
	if !ok {
		t.Fatalf("expected node tab root to be app tabs, got %T", tab)
	}
	if len(topTabs.Items) < 6 {
		t.Fatalf("expected all top-level tabs to be present, got %d", len(topTabs.Items))
	}
	if topTabs.Items[4].Text != "Import/Export" {
		t.Fatalf("expected Import/Export tab at index 4, got %q", topTabs.Items[4].Text)
	}
	if topTabs.Items[5].Text != "Maintenance" {
		t.Fatalf("expected Maintenance tab at index 5, got %q", topTabs.Items[5].Text)
	}
}

func TestNodeTabNewSettingsUseSelectBasedIntervalsAndEnums(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	mustSelectAppTabByText(t, tab, "Module configuration")

	mustSelectAppTabByText(t, tab, "Display")
	waitForCondition(t, func() bool {
		return spy.DisplayLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "Always on")

	mustSelectAppTabByText(t, tab, "Power")
	waitForCondition(t, func() bool {
		return spy.PowerLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "45 seconds")

	mustSelectAppTabByText(t, tab, "Telemetry")
	waitForCondition(t, func() bool {
		return spy.TelemetryLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "30 minutes")

	mustSelectAppTabByText(t, tab, "Paxcounter")
	waitForCondition(t, func() bool {
		return spy.PaxLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "15 minutes")

	mustSelectAppTabByText(t, tab, "External notification")
	waitForCondition(t, func() bool {
		return spy.ExternalLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "10 ms")

	mustSelectAppTabByText(t, tab, "Detection Sensor")
	waitForCondition(t, func() bool {
		return spy.DetectionLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "Falling edge")
}

func TestNodeTabNewSettingsUseSelectBasedEnumsForSerialAndAudio(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
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

	tab := newNodeTab(dep)
	_ = fynetest.NewTempWindow(t, tab)

	mustSelectAppTabByText(t, tab, "Module configuration")

	mustSelectAppTabByText(t, tab, "Serial")
	waitForCondition(t, func() bool {
		return spy.SerialLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "115200")
	mustFindSelectWithOption(t, tab, "Text message")

	mustSelectAppTabByText(t, tab, "Audio")
	waitForCondition(t, func() bool {
		return spy.AudioLoadCalls() == 1
	})
	mustFindSelectWithOption(t, tab, "3200")
}
