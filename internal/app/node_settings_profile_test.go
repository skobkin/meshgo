package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

func TestNodeSettingsServiceExportProfileLoadsLoRaOnce(t *testing.T) {
	t.Parallel()

	var messageBus bus.MessageBus
	packetID := uint32(50)
	loraRequests := 0
	loraConfig := &generated.Config_LoRaConfig{
		UsePreset:   true,
		ModemPreset: generated.Config_LoRaConfig_LONG_FAST,
		Region:      generated.Config_LoRaConfig_EU_868,
		HopLimit:    5,
	}
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if !wantResponse {
				t.Fatal("expected profile export requests to require responses")
			}

			var response *generated.AdminMessage
			switch request := payload.PayloadVariant.(type) {
			case *generated.AdminMessage_GetOwnerRequest:
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetOwnerResponse{
						GetOwnerResponse: &generated.User{LongName: "Test", ShortName: "T"},
					},
				}
			case *generated.AdminMessage_GetConfigRequest:
				config := &generated.Config{}
				if request.GetConfigRequest == generated.AdminMessage_LORA_CONFIG {
					loraRequests++
					config.PayloadVariant = &generated.Config_Lora{Lora: loraConfig}
				}
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{GetConfigResponse: config},
				}
			case *generated.AdminMessage_GetChannelRequest:
				requestIndex := request.GetChannelRequest
				if requestIndex < 1 || requestIndex > NodeChannelMaxSlots {
					t.Fatalf("unexpected channel request index: %d", requestIndex)
				}
				index := int32(requestIndex - 1) //nolint:gosec // Explicitly bounded to the supported channel slot count.
				channel := &generated.Channel{Index: index, Role: generated.Channel_DISABLED}
				if index == 0 {
					channel.Role = generated.Channel_PRIMARY
					channel.Settings = &generated.ChannelSettings{Name: "Primary", Psk: []byte{1}}
				}
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetChannelResponse{GetChannelResponse: channel},
				}
			case *generated.AdminMessage_GetModuleConfigRequest:
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetModuleConfigResponse{
						GetModuleConfigResponse: &generated.ModuleConfig{},
					},
				}
			case *generated.AdminMessage_GetRingtoneRequest:
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetRingtoneResponse{},
				}
			case *generated.AdminMessage_GetCannedMessageModuleMessagesRequest:
				response = &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetCannedMessageModuleMessagesResponse{},
				}
			default:
				t.Fatalf("unexpected profile export request: %T", request)
			}

			publishAdminReply(messageBus, to, packetID, response)
			result := stringFromUint32(packetID)
			packetID++

			return result, nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	profile, err := service.ExportProfile(ctx, mustLocalNodeTarget())
	if err != nil {
		t.Fatalf("export profile: %v", err)
	}
	if loraRequests != 1 {
		t.Fatalf("unexpected LoRa request count: got %d want 1", loraRequests)
	}
	if profile.GetConfig().GetLora().GetHopLimit() != loraConfig.GetHopLimit() {
		t.Fatalf("unexpected exported LoRa config: %+v", profile.GetConfig().GetLora())
	}
	channelSet, err := ParseChannelShareURL(profile.GetChannelUrl())
	if err != nil {
		t.Fatalf("parse exported channel URL: %v", err)
	}
	if channelSet.GetLoraConfig().GetHopLimit() != loraConfig.GetHopLimit() {
		t.Fatalf("unexpected channel URL LoRa config: %+v", channelSet.GetLoraConfig())
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
				if !wantResponse || !payload.GetGetOwnerRequest() {
					t.Fatalf("expected get owner request before import transaction")
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
			case 2:
				if wantResponse || !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings after owner request")
				}
				publishSentStatus(messageBus, packetID)
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

	if err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile, NodeProfileImportOptions{}); err != nil {
		t.Fatalf("import profile: %v", err)
	}
	if call != 8 {
		t.Fatalf("unexpected send calls count: got %d want 8", call)
	}
}

func TestNodeSettingsServiceImportProfile_OwnerReadFailsBeforeTransaction(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("owner read failed")
	longName := "Imported node"
	profile := &generated.DeviceProfile{LongName: &longName}

	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			call++
			if !wantResponse || !payload.GetGetOwnerRequest() {
				t.Fatalf("expected only get owner request, got %+v", payload)
			}

			return "", expectedErr
		},
	}
	service, _ := newTestNodeSettingsService(t, sender, true)

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile, NodeProfileImportOptions{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected owner read error, got %v", err)
	}
	if strings.Contains(err.Error(), "settings may have been partially applied") {
		t.Fatalf("unexpected partial-application warning before transaction: %q", err)
	}
	if call != 1 {
		t.Fatalf("unexpected send calls count: got %d want 1", call)
	}
}

func TestNodeSettingsServiceImportProfile_AppliesChannelsLast(t *testing.T) {
	t.Parallel()

	channelURL, err := BuildChannelShareURL([]NodeChannelSettings{
		{Name: "Primary", PSK: []byte{1}, ID: 11},
		{Name: "Ops", PSK: []byte{2}, ID: 22},
	}, NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
		Region:      int32(generated.Config_LoRaConfig_EU_868),
	}, false)
	if err != nil {
		t.Fatalf("build channel URL: %v", err)
	}
	profile := &generated.DeviceProfile{
		Config: &generated.LocalConfig{
			Device: &generated.Config_DeviceConfig{ButtonGpio: 12},
			Lora: &generated.Config_LoRaConfig{
				Region:   generated.Config_LoRaConfig_US,
				HopLimit: 7,
			},
		},
		ChannelUrl: &channelURL,
	}

	var messageBus bus.MessageBus
	call := 0
	packetID := uint32(200)
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			call++
			if wantResponse {
				t.Fatalf("unexpected response request on call %d", call)
			}
			switch call {
			case 1:
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings first")
				}
			case 2:
				if payload.GetSetConfig().GetDevice().GetButtonGpio() != 12 {
					t.Fatalf("expected regular profile settings before channels")
				}
			case 3:
				lora := payload.GetSetConfig().GetLora()
				if lora.GetRegion() != generated.Config_LoRaConfig_EU_868 {
					t.Fatalf("expected channel URL LoRa config immediately before channels")
				}
				if lora.GetHopLimit() == 7 {
					t.Fatalf("expected channel URL LoRa config to replace conflicting profile config")
				}
			case 4, 5:
				index := int32(call - 4)
				channel := payload.GetSetChannel()
				if channel.GetIndex() != index {
					t.Fatalf("unexpected channel index: got %d want %d", channel.GetIndex(), index)
				}
				if channel.GetRole() == generated.Channel_DISABLED {
					t.Fatalf("expected channel %d to be enabled", index)
				}
			case 6, 7, 8, 9, 10, 11:
				index := int32(call - 4)
				channel := payload.GetSetChannel()
				if channel.GetIndex() != index || channel.GetRole() != generated.Channel_DISABLED {
					t.Fatalf("expected channel slot %d to be disabled, got %+v", index, channel)
				}
			case 12:
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit after all channel writes")
				}
			default:
				t.Fatalf("unexpected send call %d", call)
			}
			publishSentStatus(messageBus, packetID)
			result := stringFromUint32(packetID)
			packetID++

			return result, nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	if err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile, NodeProfileImportOptions{}); err != nil {
		t.Fatalf("import profile: %v", err)
	}
	if call != 12 {
		t.Fatalf("unexpected send calls count: got %d want 12", call)
	}
}

func TestNodeSettingsServiceImportProfile_KeepExistingChannels(t *testing.T) {
	t.Parallel()

	malformedChannelURL := "not a channel URL"
	profile := &generated.DeviceProfile{ChannelUrl: &malformedChannelURL}

	var messageBus bus.MessageBus
	call := 0
	packetID := uint32(300)
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, _ bool, payload *generated.AdminMessage) (string, error) {
			call++
			switch call {
			case 1:
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings first")
				}
			case 2:
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected channel URL to be ignored and transaction committed")
				}
			default:
				t.Fatalf("unexpected payload on call %d: %s", call, fmt.Sprint(payload))
			}
			publishSentStatus(messageBus, packetID)
			result := stringFromUint32(packetID)
			packetID++

			return result, nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	if err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile, NodeProfileImportOptions{
		KeepExistingChannels: true,
	}); err != nil {
		t.Fatalf("import profile while keeping channels: %v", err)
	}
	if call != 2 {
		t.Fatalf("unexpected send calls count: got %d want 2", call)
	}
}

func TestNodeSettingsServiceImportProfile_UsesLongAggregateTimeout(t *testing.T) {
	t.Parallel()

	var messageBus bus.MessageBus
	packetID := uint32(400)
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, _ bool, _ *generated.AdminMessage) (string, error) {
			publishSentStatus(messageBus, packetID)
			result := stringFromUint32(packetID)
			packetID++

			return result, nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsProfileImportTimeout)
	defer cancel()

	err := service.runEditSettingsWriteWithTimeout(
		ctx,
		mustLocalNodeTarget(),
		"install_profile",
		nodeSettingsProfileImportTimeout,
		func(saveCtx context.Context, _ uint32) error {
			deadline, ok := saveCtx.Deadline()
			if !ok {
				t.Fatal("expected profile import context deadline")
			}
			if remaining := time.Until(deadline); remaining <= nodeSettingsOpTimeout {
				t.Fatalf("profile import deadline was shortened to single-write timeout: %s", remaining)
			}

			return nil
		},
	)
	if err != nil {
		t.Fatalf("run profile import transaction: %v", err)
	}
}

func TestNodeSettingsServiceImportProfile_WarnsAboutPartialApplication(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("radio write failed")
	var messageBus bus.MessageBus
	call := 0
	packetID := uint32(500)
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, _ bool, payload *generated.AdminMessage) (string, error) {
			call++
			if call == 1 {
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings first")
				}
				publishSentStatus(messageBus, packetID)

				return stringFromUint32(packetID), nil
			}

			return "", expectedErr
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef
	profile := &generated.DeviceProfile{
		Config: &generated.LocalConfig{
			Device: &generated.Config_DeviceConfig{ButtonGpio: 12},
		},
	}

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	err := service.ImportProfile(ctx, mustLocalNodeTarget(), profile, NodeProfileImportOptions{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected radio error, got %v", err)
	}
	if !strings.Contains(err.Error(), "settings may have been partially applied") {
		t.Fatalf("expected partial-application warning, got %q", err)
	}
}
