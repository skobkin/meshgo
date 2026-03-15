package app

import (
	"testing"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestLoRaPrimaryChannelTitleFallback(t *testing.T) {
	settings := NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
	}
	if got := LoRaPrimaryChannelTitle(settings, ""); got != "LongFast" {
		t.Fatalf("expected LongFast, got %q", got)
	}
	if got := LoRaPrimaryChannelTitle(settings, "  custom  "); got != "custom" {
		t.Fatalf("expected known title, got %q", got)
	}
}

func TestLoRaNumChannelsUsesAndroidRules(t *testing.T) {
	settings := NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
		Region:      int32(generated.Config_LoRaConfig_EU_868),
	}
	if got := LoRaNumChannels(settings); got != 1 {
		t.Fatalf("expected 1 channel for EU_868 LONG_FAST, got %d", got)
	}

	settings.Region = int32(generated.Config_LoRaConfig_US)
	if got := LoRaNumChannels(settings); got != 104 {
		t.Fatalf("expected 104 channels for US LONG_FAST, got %d", got)
	}
}

func TestLoRaEffectiveChannelNum(t *testing.T) {
	settings := NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
		Region:      int32(generated.Config_LoRaConfig_US),
	}
	expected := LoRaEffectiveChannelNum(NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
		Region:      int32(generated.Config_LoRaConfig_US),
	}, "Primary")
	if got := LoRaEffectiveChannelNum(settings, "Primary"); got != expected {
		t.Fatalf("expected %d, got %d", expected, got)
	}

	settings.ChannelNum = 7
	if got := LoRaEffectiveChannelNum(settings, "Primary"); got != 7 {
		t.Fatalf("expected explicit channel 7, got %d", got)
	}
}

func TestLoRaEffectiveRadioFreq(t *testing.T) {
	settings := NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
		Region:      int32(generated.Config_LoRaConfig_US),
		ChannelNum:  1,
	}
	if got := LoRaEffectiveRadioFreq(settings, "Primary"); got != float32(902.125) {
		t.Fatalf("expected 902.125, got %v", got)
	}

	settings.ChannelNum = 0
	settings.OverrideFrequency = 915.5
	settings.FrequencyOffset = 0.2
	if got := LoRaEffectiveRadioFreq(settings, "Primary"); got != float32(915.7) {
		t.Fatalf("expected override freq 915.7, got %v", got)
	}
}

func TestLoRaBandwidthMHz(t *testing.T) {
	settings := NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: LoRaModemPresetLongFast,
		Region:      int32(generated.Config_LoRaConfig_LORA_24),
	}
	if got := LoRaBandwidthMHz(settings); got != float32(0.8125) {
		t.Fatalf("expected 0.8125 for wide LoRa LONG_FAST, got %v", got)
	}

	settings.UsePreset = false
	settings.Bandwidth = 200
	settings.Region = int32(generated.Config_LoRaConfig_US)
	if got := LoRaBandwidthMHz(settings); got != float32(0.203125) {
		t.Fatalf("expected 0.203125 for bandwidth code 200, got %v", got)
	}
}
