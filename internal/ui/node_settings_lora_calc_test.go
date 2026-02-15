package ui

import (
	"testing"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeLoRaPrimaryTitleFromChannels(t *testing.T) {
	channels := domain.ChannelList{
		Items: []domain.ChannelInfo{
			{Index: 1, Title: "Secondary"},
			{Index: 0, Title: "Primary"},
		},
	}
	title, ok := nodeLoRaPrimaryTitleFromChannels(channels)
	if !ok {
		t.Fatal("expected primary title")
	}
	if title != "Primary" {
		t.Fatalf("expected Primary, got %q", title)
	}
}

func TestNodeLoRaPrimaryTitleFromChatStore(t *testing.T) {
	store := domain.NewChatStore()
	store.UpsertChat(domain.Chat{Key: domain.ChatKeyForChannel(0), Title: "Primary"})

	title, ok := nodeLoRaPrimaryTitleFromChatStore(store)
	if !ok {
		t.Fatal("expected title from chat store")
	}
	if title != "Primary" {
		t.Fatalf("expected Primary, got %q", title)
	}
}

func TestNodeLoRaPrimaryChannelTitleFallback(t *testing.T) {
	settings := app.NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
	}
	if got := nodeLoRaPrimaryChannelTitle(settings, ""); got != "LongFast" {
		t.Fatalf("expected LongFast, got %q", got)
	}
	if got := nodeLoRaPrimaryChannelTitle(settings, "  custom  "); got != "custom" {
		t.Fatalf("expected known title, got %q", got)
	}
}

func TestNodeLoRaNumChannelsUsesAndroidRules(t *testing.T) {
	settings := app.NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
		Region:      int32(generated.Config_LoRaConfig_EU_868),
	}
	if got := nodeLoRaNumChannels(settings); got != 1 {
		t.Fatalf("expected 1 channel for EU_868 LONG_FAST, got %d", got)
	}

	settings.Region = int32(generated.Config_LoRaConfig_US)
	if got := nodeLoRaNumChannels(settings); got != 104 {
		t.Fatalf("expected 104 channels for US LONG_FAST, got %d", got)
	}
}

func TestNodeLoRaEffectiveChannelNum(t *testing.T) {
	settings := app.NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
		Region:      int32(generated.Config_LoRaConfig_US),
	}
	expected := (nodeLoRaDJB2("Primary") % nodeLoRaNumChannels(settings)) + 1
	if got := nodeLoRaEffectiveChannelNum(settings, "Primary"); got != expected {
		t.Fatalf("expected %d, got %d", expected, got)
	}

	settings.ChannelNum = 7
	if got := nodeLoRaEffectiveChannelNum(settings, "Primary"); got != 7 {
		t.Fatalf("expected explicit channel 7, got %d", got)
	}
}

func TestNodeLoRaEffectiveRadioFreq(t *testing.T) {
	settings := app.NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
		Region:      int32(generated.Config_LoRaConfig_US),
		ChannelNum:  1,
	}
	if got := nodeLoRaEffectiveRadioFreq(settings, "Primary"); got != float32(902.125) {
		t.Fatalf("expected 902.125, got %v", got)
	}

	settings.ChannelNum = 0
	settings.OverrideFrequency = 915.5
	settings.FrequencyOffset = 0.2
	if got := nodeLoRaEffectiveRadioFreq(settings, "Primary"); got != float32(915.7) {
		t.Fatalf("expected override freq 915.7, got %v", got)
	}
}

func TestNodeLoRaBandwidthMHz(t *testing.T) {
	settings := app.NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
	}
	if got := nodeLoRaBandwidthMHz(settings, nodeLoRaRegionInfo{WideLoRa: true}); got != float32(0.8125) {
		t.Fatalf("expected 0.8125 for wide LoRa LONG_FAST, got %v", got)
	}

	settings.UsePreset = false
	settings.Bandwidth = 200
	if got := nodeLoRaBandwidthMHz(settings, nodeLoRaRegionInfo{}); got != float32(0.203125) {
		t.Fatalf("expected 0.203125 for bandwidth code 200, got %v", got)
	}
}

func TestNodeLoRaHasPaFan(t *testing.T) {
	tests := []struct {
		name       string
		boardModel string
		want       bool
	}{
		{name: "unknown model defaults to visible", boardModel: "", want: true},
		{name: "unset model defaults to visible", boardModel: "UNSET", want: true},
		{name: "betafpv has fan toggle", boardModel: generated.HardwareModel_BETAFPV_2400_TX.String(), want: true},
		{name: "bandit nano has fan toggle", boardModel: generated.HardwareModel_RADIOMASTER_900_BANDIT_NANO.String(), want: true},
		{name: "bandit has fan toggle", boardModel: generated.HardwareModel_RADIOMASTER_900_BANDIT.String(), want: true},
		{name: "other model hides fan toggle", boardModel: generated.HardwareModel_T_ECHO.String(), want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := nodeLoRaHasPaFan(tc.boardModel); got != tc.want {
				t.Fatalf("nodeLoRaHasPaFan(%q) = %v, want %v", tc.boardModel, got, tc.want)
			}
		})
	}
}
