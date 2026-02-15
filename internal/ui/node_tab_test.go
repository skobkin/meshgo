package ui

import (
	"bytes"
	"context"
	"encoding/base64"
	"sync/atomic"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

type nodeSettingsActionSpy struct {
	loadSecurityCalls  atomic.Int32
	loadPositionCalls  atomic.Int32
	loadPowerCalls     atomic.Int32
	loadDisplayCalls   atomic.Int32
	loadBluetoothCalls atomic.Int32
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

func (s *nodeSettingsActionSpy) SecurityLoadCalls() int {
	return int(s.loadSecurityCalls.Load())
}

func (s *nodeSettingsActionSpy) PositionLoadCalls() int {
	return int(s.loadPositionCalls.Load())
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
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
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

func TestNodeTabPositionSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
		Data: DataDependencies{
			LocalNodeID: func() string { return "!00000001" },
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
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
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
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
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
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

func TestNodeTabBluetoothSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	dep := RuntimeDependencies{
		Data: DataDependencies{
			LocalNodeID: func() string { return "!00000001" },
			CurrentConnStatus: func() (connectors.ConnectionStatus, bool) {
				return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
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
