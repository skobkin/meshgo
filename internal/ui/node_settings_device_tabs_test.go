package ui

import (
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"
)

func TestNodeTabNetworkSettingsLoadIsLazy(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("Fyne GUI interaction tests are not stable under the race detector")
	}

	spy := &nodeSettingsActionSpy{}
	tab := newNodeTab(newNodeSettingsRuntimeDeps(spy))
	_ = fynetest.NewTempWindow(t, tab)

	time.Sleep(100 * time.Millisecond)
	if got := spy.NetworkLoadCalls(); got != 0 {
		t.Fatalf("expected no eager network load before selecting Network tab, got %d", got)
	}

	mustSelectAppTabByText(t, tab, "Network")
	waitForCondition(t, func() bool { return spy.NetworkLoadCalls() == 1 })

	mustSelectAppTabByText(t, tab, "Display")
	mustSelectAppTabByText(t, tab, "Network")
	time.Sleep(100 * time.Millisecond)
	if got := spy.NetworkLoadCalls(); got != 1 {
		t.Fatalf("expected one lazy initial network load, got %d", got)
	}
}
