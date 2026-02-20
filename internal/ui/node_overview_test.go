package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNewNodeOverviewContent_UnavailableNodeShowsFallback(t *testing.T) {
	content, stop := newNodeOverviewContent(nodeOverviewOptions{
		Title:     "Node",
		NodeStore: domain.NewNodeStore(),
		NodeID: func() string {
			return "!00000001"
		},
		ShowActions: true,
	})
	defer stop()
	_ = fynetest.NewTempWindow(t, content)

	if !hasLabelText(content, "Node information is unavailable.") {
		t.Fatalf("expected unavailable fallback text")
	}
	if !hasLabelText(content, "Remote Administration") {
		t.Fatalf("expected remote administration section")
	}
}

func TestNewNodeOverviewContent_RemoteActionsEnabled(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{
		NodeID:      "!00000001",
		LongName:    "Alpha",
		LastHeardAt: time.Now(),
	})

	content, stop := newNodeOverviewContent(nodeOverviewOptions{
		Title:     "Node",
		NodeStore: store,
		NodeID: func() string {
			return "!00000001"
		},
		ShowActions:     true,
		ModeLocalNode:   false,
		OnDirectMessage: func(domain.Node) {},
		OnTraceroute:    func(domain.Node) {},
	})
	defer stop()
	_ = fynetest.NewTempWindow(t, content)

	chat := mustFindOverviewButtonByText(t, content, "Chat")
	if chat.Disabled() {
		t.Fatalf("expected chat action enabled for remote node")
	}
	trace := mustFindOverviewButtonByText(t, content, "Traceroute")
	if trace.Disabled() {
		t.Fatalf("expected traceroute action enabled for remote node")
	}
}

func TestOverviewTelemetryHelpers(t *testing.T) {
	uptime := uint32(3661)
	channelUtil := 12.5
	airUtil := 3.75
	temperature := 22.4
	humidity := 40.5
	pressure := 1010.0
	aqi := 80.0
	voltage := 4.01
	latitude := 1.1
	longitude := 2.2
	precisionBits := uint32(12)

	node := domain.Node{
		UptimeSeconds:         &uptime,
		ChannelUtilization:    &channelUtil,
		AirUtilTx:             &airUtil,
		Temperature:           &temperature,
		Humidity:              &humidity,
		Pressure:              &pressure,
		AirQualityIndex:       &aqi,
		Voltage:               &voltage,
		LastHeardAt:           time.Now(),
		PositionUpdatedAt:     time.Now(),
		Latitude:              &latitude,
		Longitude:             &longitude,
		PositionPrecisionBits: &precisionBits,
	}

	if got := overviewUptime(node.UptimeSeconds); got == "unknown" {
		t.Fatalf("expected uptime to be formatted")
	}
	if got := overviewPowerTelemetry(node); got == "" {
		t.Fatalf("expected power telemetry text")
	}
	if got := overviewEnvironmentTelemetry(node); got == "" {
		t.Fatalf("expected environment telemetry text")
	}
	if got := overviewOtherTelemetry(node); got == "" {
		t.Fatalf("expected other telemetry text")
	}
	if got := overviewPosition(node); got == "" {
		t.Fatalf("expected position text")
	}
}

func mustFindOverviewButtonByText(t *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	for _, object := range fynetest.LaidOutObjects(root) {
		button, ok := object.(*widget.Button)
		if !ok {
			continue
		}
		if button.Text == text {
			return button
		}
	}
	t.Fatalf("button %q not found", text)

	return nil
}
