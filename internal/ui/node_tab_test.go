package ui

import (
	"bytes"
	"encoding/base64"
	"testing"
	"time"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

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
