package ui

import (
	"strings"
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

	if !hasLabelText(content, "information is unavailable") {
		t.Fatalf("expected unavailable fallback text")
	}
	if !hasLabelText(content, "Remote Administration") {
		t.Fatalf("expected remote administration section")
	}
	if hasLabelText(content, "Last Heard") {
		t.Fatalf("did not expect separate Last Heard section title")
	}
}

func TestNewNodeOverviewContent_RemoteActionsEnabled(t *testing.T) {
	store := domain.NewNodeStore()
	battery := uint32(90)
	temperature := 21.5
	aqi := 42.0
	channelUtil := 3.5
	store.Upsert(domain.Node{
		NodeID:             "!00000001",
		LongName:           "Alpha",
		LastHeardAt:        time.Now(),
		BatteryLevel:       &battery,
		Temperature:        &temperature,
		AirQualityIndex:    &aqi,
		ChannelUtilization: &channelUtil,
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
	if hasLabelText(content, "Uptime") {
		t.Fatalf("did not expect uptime field when uptime is unknown")
	}
	if copy := mustFindOverviewButtonByText(t, content, "Copy"); !copy.Disabled() {
		t.Fatalf("expected public key copy action disabled when key is unknown")
	}
	if got := countOverviewRefreshButtons(content); got < 5 {
		t.Fatalf("expected refresh action buttons for remote node, got %d", got)
	}
}

func TestNewNodeOverviewContent_LocalNodeHidesRefreshActions(t *testing.T) {
	store := domain.NewNodeStore()
	battery := uint32(90)
	temperature := 21.5
	aqi := 42.0
	channelUtil := 3.5
	store.Upsert(domain.Node{
		NodeID:             "!00000001",
		LongName:           "Alpha",
		LastHeardAt:        time.Now(),
		BatteryLevel:       &battery,
		Temperature:        &temperature,
		AirQualityIndex:    &aqi,
		ChannelUtilization: &channelUtil,
	})

	content, stop := newNodeOverviewContent(nodeOverviewOptions{
		Title:     "Node",
		NodeStore: store,
		NodeID: func() string {
			return "!00000001"
		},
		ShowActions:   true,
		ModeLocalNode: true,
	})
	defer stop()
	_ = fynetest.NewTempWindow(t, content)

	if got := countOverviewRefreshButtons(content); got != 0 {
		t.Fatalf("expected no refresh action buttons for local node, got %d", got)
	}
}

func TestNewNodeOverviewContent_PublicKeyCopyEnabledWhenKeyIsKnown(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{
		NodeID:      "!00000001",
		LongName:    "Alpha",
		LastHeardAt: time.Now(),
		PublicKey:   []byte{1, 2, 3, 4},
	})

	content, stop := newNodeOverviewContent(nodeOverviewOptions{
		Title:     "Node",
		NodeStore: store,
		NodeID: func() string {
			return "!00000001"
		},
		ShowActions: true,
	})
	defer stop()
	_ = fynetest.NewTempWindow(t, content)

	if copy := mustFindOverviewButtonByText(t, content, "Copy"); copy.Disabled() {
		t.Fatalf("expected public key copy action enabled when key is known")
	}
}

func TestOverviewTelemetryHelpers(t *testing.T) {
	uptime := uint32(3661)
	channelUtil := 12.5
	airUtil := 3.75
	temperature := 22.4
	humidity := 40.5
	pressure := 1010.0
	soilTemperature := 18.7
	soilMoisture := uint32(43)
	gasResistance := 0.77
	lux := 123.0
	uvLux := 321.0
	radiation := 0.16
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
		SoilTemperature:       &soilTemperature,
		SoilMoisture:          &soilMoisture,
		GasResistance:         &gasResistance,
		Lux:                   &lux,
		UVLux:                 &uvLux,
		Radiation:             &radiation,
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
	} else if !strings.Contains(got, "Gas resistance") || !strings.Contains(got, "Dew point") || !strings.Contains(got, "Light") || !strings.Contains(got, "Soil temperature") || !strings.Contains(got, "Soil moisture") || !strings.Contains(got, "UV light") || !strings.Contains(got, "Radiation") {
		t.Fatalf("expected extended environment metrics in output, got %q", got)
	}
	if got := overviewOtherTelemetry(node); got == "" {
		t.Fatalf("expected other telemetry text")
	}
	if got := overviewPosition(node); got == "" {
		t.Fatalf("expected position text")
	} else if !strings.Contains(got, "Precision: Approx.") {
		t.Fatalf("expected human-readable precision label, got %q", got)
	}
}

func TestOverviewMetricsColumnCount(t *testing.T) {
	tests := []struct {
		name    string
		count   int
		columns int
	}{
		{name: "few metrics use two columns", count: 1, columns: 2},
		{name: "medium metrics use three columns", count: 4, columns: 3},
		{name: "large metrics use four columns", count: 10, columns: 4},
		{name: "very large metrics use five columns", count: 20, columns: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overviewMetricsColumnCount(tt.count); got != tt.columns {
				t.Fatalf("expected %d columns for %d metrics, got %d", tt.columns, tt.count, got)
			}
		})
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

func countOverviewRefreshButtons(root fyne.CanvasObject) int {
	count := 0
	for _, object := range fynetest.LaidOutObjects(root) {
		button, ok := object.(*widget.Button)
		if !ok {
			continue
		}
		if button.Icon != nil && strings.Contains(strings.ToLower(button.Icon.Name()), "refresh") {
			count++
		}
	}

	return count
}
