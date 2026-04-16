package app

import (
	"testing"

	"github.com/skobkin/meshgo/internal/bus"
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

func TestNodeSettingsServiceImportProfile_AppliesProvidedFields(t *testing.T) {
	t.Parallel()

	longName := "Imported node"
	ringtone := "beep"
	cannedMessages := "one\ntwo"
	profile := &generated.DeviceProfile{
		LongName: &longName,
		Config: &generated.LocalConfig{
			Device: &generated.Config_DeviceConfig{ButtonGpio: 12},
		},
		ModuleConfig: &generated.LocalModuleConfig{
			Serial: &generated.ModuleConfig_SerialConfig{Enabled: true},
		},
		Ringtone:       &ringtone,
		CannedMessages: &cannedMessages,
	}

	var messageBus bus.MessageBus
	call := 0
	packetID := uint32(100)
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			call++
			switch call {
			case 1:
				if wantResponse || !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings status write")
				}
				publishSentStatus(messageBus, packetID)
			case 2:
				if !wantResponse || !payload.GetGetOwnerRequest() {
					t.Fatalf("expected get owner request during import")
				}
				isUnmessageable := false
				publishAdminReply(messageBus, to, packetID, &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetOwnerResponse{
						GetOwnerResponse: &generated.User{
							LongName:       "Current",
							ShortName:      "CU",
							IsLicensed:     true,
							IsUnmessagable: &isUnmessageable,
						},
					},
				})
			case 3:
				if wantResponse {
					t.Fatalf("expected set owner write without wantResponse")
				}
				if payload.GetSetOwner().GetLongName() != longName {
					t.Fatalf("unexpected imported owner payload: %+v", payload.GetSetOwner())
				}
				publishSentStatus(messageBus, packetID)
			case 4:
				if payload.GetSetConfig().GetDevice().GetButtonGpio() != 12 {
					t.Fatalf("unexpected imported device config payload")
				}
				publishSentStatus(messageBus, packetID)
			case 5:
				if !payload.GetSetModuleConfig().GetSerial().GetEnabled() {
					t.Fatalf("unexpected imported serial module payload")
				}
				publishSentStatus(messageBus, packetID)
			case 6:
				if payload.GetSetRingtoneMessage() != ringtone {
					t.Fatalf("unexpected imported ringtone payload")
				}
				publishSentStatus(messageBus, packetID)
			case 7:
				if payload.GetSetCannedMessageModuleMessages() != cannedMessages {
					t.Fatalf("unexpected imported canned messages payload")
				}
				publishSentStatus(messageBus, packetID)
			case 8:
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
				publishSentStatus(messageBus, packetID)
			default:
				t.Fatalf("unexpected send call %d", call)
			}
			defer func() { packetID++ }()

			return stringFromUint32(packetID), nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	if err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile); err != nil {
		t.Fatalf("import profile: %v", err)
	}
	if call != 8 {
		t.Fatalf("unexpected send calls count: got %d want 8", call)
	}
}
