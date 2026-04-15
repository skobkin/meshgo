package app

import (
	"testing"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestEncodeDecodeDeviceProfile_RoundTrip(t *testing.T) {
	longName := "Test node"
	shortName := "TN"
	ringtone := "beep"
	cannedMessages := "one\ntwo"
	channelURL := "https://meshtastic.org/e/#test"

	profile := &generated.DeviceProfile{
		LongName:       &longName,
		ShortName:      &shortName,
		ChannelUrl:     &channelURL,
		Ringtone:       &ringtone,
		CannedMessages: &cannedMessages,
		Config: &generated.LocalConfig{
			Device: &generated.Config_DeviceConfig{ButtonGpio: 12},
			Lora:   &generated.Config_LoRaConfig{HopLimit: 3},
		},
		ModuleConfig: &generated.LocalModuleConfig{
			Mqtt:          &generated.ModuleConfig_MQTTConfig{Enabled: true},
			Statusmessage: &generated.ModuleConfig_StatusMessageConfig{NodeStatus: "Ready"},
		},
	}

	raw, err := EncodeDeviceProfile(profile)
	if err != nil {
		t.Fatalf("encode profile: %v", err)
	}

	decoded, err := DecodeDeviceProfile(raw)
	if err != nil {
		t.Fatalf("decode profile: %v", err)
	}

	if decoded.GetLongName() != longName {
		t.Fatalf("unexpected long name: %q", decoded.GetLongName())
	}
	if decoded.GetShortName() != shortName {
		t.Fatalf("unexpected short name: %q", decoded.GetShortName())
	}
	if decoded.GetChannelUrl() != channelURL {
		t.Fatalf("unexpected channel url: %q", decoded.GetChannelUrl())
	}
	if decoded.GetRingtone() != ringtone {
		t.Fatalf("unexpected ringtone: %q", decoded.GetRingtone())
	}
	if decoded.GetCannedMessages() != cannedMessages {
		t.Fatalf("unexpected canned messages: %q", decoded.GetCannedMessages())
	}
	if decoded.GetConfig().GetDevice().GetButtonGpio() != 12 {
		t.Fatalf("unexpected device button gpio: %d", decoded.GetConfig().GetDevice().GetButtonGpio())
	}
	if !decoded.GetModuleConfig().GetMqtt().GetEnabled() {
		t.Fatalf("expected mqtt config to survive round-trip")
	}
	if decoded.GetModuleConfig().GetStatusmessage().GetNodeStatus() != "Ready" {
		t.Fatalf("expected status message config to survive round-trip")
	}
}
