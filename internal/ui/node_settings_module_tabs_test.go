package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/container"
	fynetest "fyne.io/fyne/v2/test"
)

func TestNodeTabModuleConfigurationIncludesNewTabsInAndroidOrder(t *testing.T) {
	tab := newNodeTabWithOnShow(RuntimeDependencies{})
	topTabs, ok := tab.(*container.AppTabs)
	if !ok {
		t.Fatalf("expected node tab root to be app tabs, got %T", tab)
	}
	var moduleTabs *container.AppTabs
	for _, item := range topTabs.Items {
		if item.Text == "Module configuration" {
			var castOk bool
			moduleTabs, castOk = item.Content.(*container.AppTabs)
			if !castOk {
				t.Fatalf("expected module configuration content to be app tabs, got %T", item.Content)
			}

			break
		}
	}
	if moduleTabs == nil {
		t.Fatalf("expected module configuration tab")
	}

	got := make([]string, 0, len(moduleTabs.Items))
	for _, item := range moduleTabs.Items {
		got = append(got, item.Text)
	}

	want := []string{
		"MQTT",
		"Serial",
		"External notification",
		"Store & Forward",
		"Range test",
		"Telemetry",
		"Canned Message",
		"Audio",
		"Remote Hardware",
		"Neighbor Info",
		"Ambient Lighting",
		"Detection Sensor",
		"Paxcounter",
		"Status Message",
	}
	if len(got) != len(want) {
		t.Fatalf("unexpected module tab count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected module tab at index %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestNodeTabNewModuleSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	testCases := []struct {
		name  string
		tab   string
		other string
		calls func(*nodeSettingsActionSpy) int
	}{
		{name: "Serial", tab: "Serial", other: "MQTT", calls: (*nodeSettingsActionSpy).SerialLoadCalls},
		{name: "ExternalNotification", tab: "External notification", other: "Serial", calls: (*nodeSettingsActionSpy).ExternalLoadCalls},
		{name: "StoreForward", tab: "Store & Forward", other: "External notification", calls: (*nodeSettingsActionSpy).StoreForwardLoadCalls},
		{name: "Telemetry", tab: "Telemetry", other: "Range test", calls: (*nodeSettingsActionSpy).TelemetryLoadCalls},
		{name: "CannedMessage", tab: "Canned Message", other: "Telemetry", calls: (*nodeSettingsActionSpy).CannedMessageLoadCalls},
		{name: "Audio", tab: "Audio", other: "Canned Message", calls: (*nodeSettingsActionSpy).AudioLoadCalls},
		{name: "RemoteHardware", tab: "Remote Hardware", other: "Audio", calls: (*nodeSettingsActionSpy).RemoteHardwareLoadCalls},
		{name: "NeighborInfo", tab: "Neighbor Info", other: "Remote Hardware", calls: (*nodeSettingsActionSpy).NeighborInfoLoadCalls},
		{name: "AmbientLighting", tab: "Ambient Lighting", other: "Neighbor Info", calls: (*nodeSettingsActionSpy).AmbientLightingLoadCalls},
		{name: "DetectionSensor", tab: "Detection Sensor", other: "Ambient Lighting", calls: (*nodeSettingsActionSpy).DetectionLoadCalls},
		{name: "Paxcounter", tab: "Paxcounter", other: "Detection Sensor", calls: (*nodeSettingsActionSpy).PaxLoadCalls},
		{name: "StatusMessage", tab: "Status Message", other: "Paxcounter", calls: (*nodeSettingsActionSpy).StatusMessageLoadCalls},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &nodeSettingsActionSpy{}
			tab := newNodeTab(newNodeSettingsRuntimeDeps(spy))
			_ = fynetest.NewTempWindow(t, tab)

			time.Sleep(100 * time.Millisecond)
			if got := tc.calls(spy); got != 0 {
				t.Fatalf("expected no eager load before selecting %s, got %d", tc.tab, got)
			}

			mustSelectAppTabByText(t, tab, "Module configuration")
			mustSelectAppTabByText(t, tab, tc.tab)
			waitForCondition(t, func() bool { return tc.calls(spy) == 1 })

			mustSelectAppTabByText(t, tab, tc.other)
			mustSelectAppTabByText(t, tab, tc.tab)
			time.Sleep(100 * time.Millisecond)
			if got := tc.calls(spy); got != 1 {
				t.Fatalf("expected one lazy initial load for %s, got %d", tc.tab, got)
			}
		})
	}
}

func TestNodeTabNewModuleSettingsExposeExpectedSelectControls(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	tab := newNodeTab(newNodeSettingsRuntimeDeps(spy))
	_ = fynetest.NewTempWindow(t, tab)

	mustSelectAppTabByText(t, tab, "Module configuration")

	testCases := []struct {
		tab    string
		calls  func(*nodeSettingsActionSpy) int
		option string
	}{
		{tab: "Serial", calls: (*nodeSettingsActionSpy).SerialLoadCalls, option: "115200"},
		{tab: "Serial", calls: (*nodeSettingsActionSpy).SerialLoadCalls, option: "Text message"},
		{tab: "External notification", calls: (*nodeSettingsActionSpy).ExternalLoadCalls, option: "10 ms"},
		{tab: "Telemetry", calls: (*nodeSettingsActionSpy).TelemetryLoadCalls, option: "30 minutes"},
		{tab: "Audio", calls: (*nodeSettingsActionSpy).AudioLoadCalls, option: "3200"},
		{tab: "Neighbor Info", calls: (*nodeSettingsActionSpy).NeighborInfoLoadCalls, option: "6 hours"},
		{tab: "Detection Sensor", calls: (*nodeSettingsActionSpy).DetectionLoadCalls, option: "Falling edge"},
		{tab: "Paxcounter", calls: (*nodeSettingsActionSpy).PaxLoadCalls, option: "15 minutes"},
	}

	for _, tc := range testCases {
		mustSelectAppTabByText(t, tab, tc.tab)
		waitForCondition(t, func() bool { return tc.calls(spy) == 1 })
		mustFindSelectWithOption(t, tab, tc.option)
	}
}
