package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

type stubAdminSender struct {
	send func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error)
}

func (s stubAdminSender) SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
	return s.send(to, channel, wantResponse, payload)
}

func TestNodeSettingsServiceLoadUserSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	isUnmessageable := true
	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if !payload.GetGetOwnerRequest() {
				t.Fatalf("expected get owner request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get owner request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   42,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetOwnerResponse{
						GetOwnerResponse: &generated.User{
							LongName:       "Test Node",
							ShortName:      "TN",
							IsLicensed:     true,
							IsUnmessagable: &isUnmessageable,
						},
					},
				},
			})

			return "42", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadUserSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load user settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if settings.LongName != "Test Node" {
		t.Fatalf("unexpected long name: %q", settings.LongName)
	}
	if settings.ShortName != "TN" {
		t.Fatalf("unexpected short name: %q", settings.ShortName)
	}
	if !settings.HamLicensed {
		t.Fatalf("expected HAM licensed to be true")
	}
	if !settings.IsUnmessageable {
		t.Fatalf("expected unmessageable to be true")
	}
}

func TestNodeSettingsServiceSaveUserSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_owner", "commit"}
	packetIDs := []uint32{100, 101, 102}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_owner":
				if payload.GetSetOwner() == nil {
					t.Fatalf("expected set owner payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SaveUserSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeUserSettings{
		NodeID:          "!00000001",
		LongName:        "Updated Node",
		ShortName:       "UN",
		HamLicensed:     true,
		IsUnmessageable: false,
	})
	if err != nil {
		t.Fatalf("save user settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadSecuritySettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	publicKey := bytes.Repeat([]byte{0x11}, 32)
	privateKey := bytes.Repeat([]byte{0x22}, 32)
	adminKeyA := bytes.Repeat([]byte{0x33}, 32)
	adminKeyB := bytes.Repeat([]byte{0x44}, 32)

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_SECURITY_CONFIG {
				t.Fatalf("expected security config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get security config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   64,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Security{
								Security: &generated.Config_SecurityConfig{
									PublicKey:           publicKey,
									PrivateKey:          privateKey,
									AdminKey:            [][]byte{adminKeyA, adminKeyB},
									IsManaged:           true,
									SerialEnabled:       true,
									DebugLogApiEnabled:  true,
									AdminChannelEnabled: false,
								},
							},
						},
					},
				},
			})

			return "64", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadSecuritySettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load security settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if !bytes.Equal(settings.PublicKey, publicKey) {
		t.Fatalf("unexpected public key")
	}
	if !bytes.Equal(settings.PrivateKey, privateKey) {
		t.Fatalf("unexpected private key")
	}
	if len(settings.AdminKeys) != 2 {
		t.Fatalf("unexpected admin keys count: %d", len(settings.AdminKeys))
	}
	if !bytes.Equal(settings.AdminKeys[0], adminKeyA) {
		t.Fatalf("unexpected admin key #1")
	}
	if !bytes.Equal(settings.AdminKeys[1], adminKeyB) {
		t.Fatalf("unexpected admin key #2")
	}
	if !settings.IsManaged {
		t.Fatalf("expected managed mode to be true")
	}
	if !settings.SerialEnabled {
		t.Fatalf("expected serial enabled to be true")
	}
	if !settings.DebugLogAPIEnabled {
		t.Fatalf("expected debug log API enabled to be true")
	}
	if settings.AdminChannelEnabled {
		t.Fatalf("expected admin channel enabled to be false")
	}
}

func TestNodeSettingsServiceSaveSecuritySettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	publicKey := bytes.Repeat([]byte{0x01}, 32)
	privateKey := bytes.Repeat([]byte{0x02}, 32)
	adminKeyA := bytes.Repeat([]byte{0x03}, 32)
	adminKeyB := bytes.Repeat([]byte{0x04}, 32)

	expectedPayloadKinds := []string{"begin", "set_security", "commit"}
	packetIDs := []uint32{200, 201, 202}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_security":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				security := cfg.GetSecurity()
				if security == nil {
					t.Fatalf("expected security config payload")
				}
				if !bytes.Equal(security.GetPublicKey(), publicKey) {
					t.Fatalf("unexpected public key payload")
				}
				if !bytes.Equal(security.GetPrivateKey(), privateKey) {
					t.Fatalf("unexpected private key payload")
				}
				if len(security.GetAdminKey()) != 2 {
					t.Fatalf("unexpected admin keys payload count: %d", len(security.GetAdminKey()))
				}
				if !bytes.Equal(security.GetAdminKey()[0], adminKeyA) {
					t.Fatalf("unexpected admin key #1 payload")
				}
				if !bytes.Equal(security.GetAdminKey()[1], adminKeyB) {
					t.Fatalf("unexpected admin key #2 payload")
				}
				if !security.GetIsManaged() {
					t.Fatalf("expected managed mode to be true")
				}
				if !security.GetSerialEnabled() {
					t.Fatalf("expected serial enabled to be true")
				}
				if !security.GetDebugLogApiEnabled() {
					t.Fatalf("expected debug log API enabled to be true")
				}
				if security.GetAdminChannelEnabled() {
					t.Fatalf("expected admin channel enabled to be false")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SaveSecuritySettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeSecuritySettings{
		NodeID:              "!00000001",
		PublicKey:           publicKey,
		PrivateKey:          privateKey,
		AdminKeys:           [][]byte{adminKeyA, adminKeyB},
		IsManaged:           true,
		SerialEnabled:       true,
		DebugLogAPIEnabled:  true,
		AdminChannelEnabled: false,
	})
	if err != nil {
		t.Fatalf("save security settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadDeviceSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_DEVICE_CONFIG {
				t.Fatalf("expected device config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get device config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   96,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Device{
								Device: &generated.Config_DeviceConfig{
									Role:                   generated.Config_DeviceConfig_TRACKER,
									ButtonGpio:             17,
									BuzzerGpio:             18,
									RebroadcastMode:        generated.Config_DeviceConfig_LOCAL_ONLY,
									NodeInfoBroadcastSecs:  600,
									DoubleTapAsButtonPress: true,
									DisableTripleClick:     false,
									Tzdef:                  "Europe/Amsterdam",
									LedHeartbeatDisabled:   true,
									BuzzerMode:             generated.Config_DeviceConfig_SYSTEM_ONLY,
								},
							},
						},
					},
				},
			})

			return "96", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadDeviceSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load device settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if settings.Role != int32(generated.Config_DeviceConfig_TRACKER) {
		t.Fatalf("unexpected role: %d", settings.Role)
	}
	if settings.ButtonGPIO != 17 {
		t.Fatalf("unexpected button gpio: %d", settings.ButtonGPIO)
	}
	if settings.BuzzerGPIO != 18 {
		t.Fatalf("unexpected buzzer gpio: %d", settings.BuzzerGPIO)
	}
	if settings.RebroadcastMode != int32(generated.Config_DeviceConfig_LOCAL_ONLY) {
		t.Fatalf("unexpected rebroadcast mode: %d", settings.RebroadcastMode)
	}
	if settings.NodeInfoBroadcastSecs != 600 {
		t.Fatalf("unexpected node info interval: %d", settings.NodeInfoBroadcastSecs)
	}
	if !settings.DoubleTapAsButtonPress {
		t.Fatalf("expected double tap as button press to be true")
	}
	if settings.DisableTripleClick {
		t.Fatalf("expected disable triple click to be false")
	}
	if settings.Tzdef != "Europe/Amsterdam" {
		t.Fatalf("unexpected tzdef: %q", settings.Tzdef)
	}
	if !settings.LedHeartbeatDisabled {
		t.Fatalf("expected led heartbeat disabled to be true")
	}
	if settings.BuzzerMode != int32(generated.Config_DeviceConfig_SYSTEM_ONLY) {
		t.Fatalf("unexpected buzzer mode: %d", settings.BuzzerMode)
	}
}

func TestNodeSettingsServiceSaveDeviceSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_device", "commit"}
	packetIDs := []uint32{300, 301, 302}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_device":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				device := cfg.GetDevice()
				if device == nil {
					t.Fatalf("expected device config payload")
				}
				if device.GetRole() != generated.Config_DeviceConfig_CLIENT_BASE {
					t.Fatalf("unexpected role payload")
				}
				if device.GetButtonGpio() != 41 {
					t.Fatalf("unexpected button gpio payload")
				}
				if device.GetBuzzerGpio() != 42 {
					t.Fatalf("unexpected buzzer gpio payload")
				}
				if device.GetRebroadcastMode() != generated.Config_DeviceConfig_CORE_PORTNUMS_ONLY {
					t.Fatalf("unexpected rebroadcast mode payload")
				}
				if device.GetNodeInfoBroadcastSecs() != 1200 {
					t.Fatalf("unexpected node info interval payload")
				}
				if !device.GetDoubleTapAsButtonPress() {
					t.Fatalf("expected double tap payload")
				}
				if !device.GetDisableTripleClick() {
					t.Fatalf("expected disable triple click payload")
				}
				if device.GetTzdef() != "America/Los_Angeles" {
					t.Fatalf("unexpected tzdef payload")
				}
				if !device.GetLedHeartbeatDisabled() {
					t.Fatalf("expected led heartbeat disabled payload")
				}
				if device.GetBuzzerMode() != generated.Config_DeviceConfig_DIRECT_MSG_ONLY {
					t.Fatalf("unexpected buzzer mode payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SaveDeviceSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeDeviceSettings{
		NodeID:                 "!00000001",
		Role:                   int32(generated.Config_DeviceConfig_CLIENT_BASE),
		ButtonGPIO:             41,
		BuzzerGPIO:             42,
		RebroadcastMode:        int32(generated.Config_DeviceConfig_CORE_PORTNUMS_ONLY),
		NodeInfoBroadcastSecs:  1200,
		DoubleTapAsButtonPress: true,
		DisableTripleClick:     true,
		Tzdef:                  "America/Los_Angeles",
		LedHeartbeatDisabled:   true,
		BuzzerMode:             int32(generated.Config_DeviceConfig_DIRECT_MSG_ONLY),
	})
	if err != nil {
		t.Fatalf("save device settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadPositionSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	positionFlags := uint32(generated.Config_PositionConfig_ALTITUDE) | uint32(generated.Config_PositionConfig_HEADING)

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_POSITION_CONFIG {
				t.Fatalf("expected position config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get position config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   128,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Position{
								Position: &generated.Config_PositionConfig{
									PositionBroadcastSecs:             900,
									PositionBroadcastSmartEnabled:     true,
									FixedPosition:                     true,
									GpsUpdateInterval:                 120,
									PositionFlags:                     positionFlags,
									RxGpio:                            33,
									TxGpio:                            34,
									BroadcastSmartMinimumDistance:     150,
									BroadcastSmartMinimumIntervalSecs: 30,
									GpsEnGpio:                         35,
									GpsMode:                           generated.Config_PositionConfig_ENABLED,
								},
							},
						},
					},
				},
			})

			return "128", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadPositionSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load position settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if settings.PositionBroadcastSecs != 900 {
		t.Fatalf("unexpected position broadcast interval: %d", settings.PositionBroadcastSecs)
	}
	if !settings.PositionBroadcastSmartEnabled {
		t.Fatalf("expected smart position to be enabled")
	}
	if !settings.FixedPosition {
		t.Fatalf("expected fixed position to be enabled")
	}
	if settings.GpsUpdateInterval != 120 {
		t.Fatalf("unexpected GPS update interval: %d", settings.GpsUpdateInterval)
	}
	if settings.PositionFlags != positionFlags {
		t.Fatalf("unexpected position flags: %d", settings.PositionFlags)
	}
	if settings.RxGPIO != 33 {
		t.Fatalf("unexpected GPS RX GPIO: %d", settings.RxGPIO)
	}
	if settings.TxGPIO != 34 {
		t.Fatalf("unexpected GPS TX GPIO: %d", settings.TxGPIO)
	}
	if settings.BroadcastSmartMinimumDistance != 150 {
		t.Fatalf("unexpected smart minimum distance: %d", settings.BroadcastSmartMinimumDistance)
	}
	if settings.BroadcastSmartMinimumIntervalSecs != 30 {
		t.Fatalf("unexpected smart minimum interval: %d", settings.BroadcastSmartMinimumIntervalSecs)
	}
	if settings.GpsEnGPIO != 35 {
		t.Fatalf("unexpected GPS EN GPIO: %d", settings.GpsEnGPIO)
	}
	if settings.GpsMode != int32(generated.Config_PositionConfig_ENABLED) {
		t.Fatalf("unexpected GPS mode: %d", settings.GpsMode)
	}
}

func TestNodeSettingsServiceSavePositionSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_fixed_position", "set_position", "commit"}
	packetIDs := []uint32{400, 401, 402, 403}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_fixed_position":
				position := payload.GetSetFixedPosition()
				if position == nil {
					t.Fatalf("expected set fixed position payload")
				}
				if position.GetLatitudeI() != 515072000 {
					t.Fatalf("unexpected fixed latitude payload: %d", position.GetLatitudeI())
				}
				if position.GetLongitudeI() != -127800 {
					t.Fatalf("unexpected fixed longitude payload: %d", position.GetLongitudeI())
				}
				if position.GetAltitude() != 87 {
					t.Fatalf("unexpected fixed altitude payload: %d", position.GetAltitude())
				}
			case "set_position":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				position := cfg.GetPosition()
				if position == nil {
					t.Fatalf("expected position config payload")
				}
				if position.GetPositionBroadcastSecs() != 1800 {
					t.Fatalf("unexpected position broadcast interval payload")
				}
				if !position.GetPositionBroadcastSmartEnabled() {
					t.Fatalf("expected smart position payload")
				}
				if !position.GetFixedPosition() {
					t.Fatalf("expected fixed position payload")
				}
				if position.GetGpsUpdateInterval() != 300 {
					t.Fatalf("unexpected GPS update interval payload")
				}
				if position.GetPositionFlags() != (uint32(generated.Config_PositionConfig_ALTITUDE) | uint32(generated.Config_PositionConfig_SPEED)) {
					t.Fatalf("unexpected position flags payload")
				}
				if position.GetRxGpio() != 21 {
					t.Fatalf("unexpected GPS RX GPIO payload")
				}
				if position.GetTxGpio() != 22 {
					t.Fatalf("unexpected GPS TX GPIO payload")
				}
				if position.GetBroadcastSmartMinimumDistance() != 200 {
					t.Fatalf("unexpected smart minimum distance payload")
				}
				if position.GetBroadcastSmartMinimumIntervalSecs() != 45 {
					t.Fatalf("unexpected smart minimum interval payload")
				}
				if position.GetGpsEnGpio() != 23 {
					t.Fatalf("unexpected GPS EN GPIO payload")
				}
				if position.GetGpsMode() != generated.Config_PositionConfig_NOT_PRESENT {
					t.Fatalf("unexpected GPS mode payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SavePositionSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodePositionSettings{
		NodeID:                            "!00000001",
		PositionBroadcastSecs:             1800,
		PositionBroadcastSmartEnabled:     true,
		FixedPosition:                     true,
		FixedLatitude:                     float64Ptr(51.5072),
		FixedLongitude:                    float64Ptr(-0.01278),
		FixedAltitude:                     int32Ptr(87),
		GpsUpdateInterval:                 300,
		PositionFlags:                     uint32(generated.Config_PositionConfig_ALTITUDE) | uint32(generated.Config_PositionConfig_SPEED),
		RxGPIO:                            21,
		TxGPIO:                            22,
		BroadcastSmartMinimumDistance:     200,
		BroadcastSmartMinimumIntervalSecs: 45,
		GpsEnGPIO:                         23,
		GpsMode:                           int32(generated.Config_PositionConfig_NOT_PRESENT),
	})
	if err != nil {
		t.Fatalf("save position settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceSavePositionSettings_RemoveFixedPosition_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "remove_fixed_position", "set_position", "commit"}
	packetIDs := []uint32{500, 501, 502, 503}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "remove_fixed_position":
				if !payload.GetRemoveFixedPosition() {
					t.Fatalf("expected remove fixed position payload")
				}
			case "set_position":
				cfg := payload.GetSetConfig()
				if cfg == nil || cfg.GetPosition() == nil {
					t.Fatalf("expected set position config payload")
				}
				if cfg.GetPosition().GetFixedPosition() {
					t.Fatalf("expected fixed position flag to be false")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SavePositionSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodePositionSettings{
		NodeID:                            "!00000001",
		PositionBroadcastSecs:             1800,
		PositionBroadcastSmartEnabled:     true,
		FixedPosition:                     false,
		RemoveFixedPosition:               true,
		GpsUpdateInterval:                 300,
		PositionFlags:                     uint32(generated.Config_PositionConfig_ALTITUDE) | uint32(generated.Config_PositionConfig_SPEED),
		RxGPIO:                            21,
		TxGPIO:                            22,
		BroadcastSmartMinimumDistance:     200,
		BroadcastSmartMinimumIntervalSecs: 45,
		GpsEnGPIO:                         23,
		GpsMode:                           int32(generated.Config_PositionConfig_NOT_PRESENT),
	})
	if err != nil {
		t.Fatalf("save position settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadPowerSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_POWER_CONFIG {
				t.Fatalf("expected power config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get power config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   160,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Power{
								Power: &generated.Config_PowerConfig{
									IsPowerSaving:              true,
									OnBatteryShutdownAfterSecs: 7200,
									AdcMultiplierOverride:      1.25,
									WaitBluetoothSecs:          120,
									SdsSecs:                    86400,
									LsSecs:                     600,
									MinWakeSecs:                15,
									DeviceBatteryInaAddress:    66,
									PowermonEnables:            3,
								},
							},
						},
					},
				},
			})

			return "160", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadPowerSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load power settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if !settings.IsPowerSaving {
		t.Fatalf("expected power saving to be true")
	}
	if settings.OnBatteryShutdownAfterSecs != 7200 {
		t.Fatalf("unexpected shutdown on power loss: %d", settings.OnBatteryShutdownAfterSecs)
	}
	if settings.AdcMultiplierOverride != 1.25 {
		t.Fatalf("unexpected ADC multiplier override: %v", settings.AdcMultiplierOverride)
	}
	if settings.WaitBluetoothSecs != 120 {
		t.Fatalf("unexpected wait bluetooth seconds: %d", settings.WaitBluetoothSecs)
	}
	if settings.SdsSecs != 86400 {
		t.Fatalf("unexpected super deep sleep seconds: %d", settings.SdsSecs)
	}
	if settings.LsSecs != 600 {
		t.Fatalf("unexpected light sleep seconds: %d", settings.LsSecs)
	}
	if settings.MinWakeSecs != 15 {
		t.Fatalf("unexpected minimum wake seconds: %d", settings.MinWakeSecs)
	}
	if settings.DeviceBatteryInaAddress != 66 {
		t.Fatalf("unexpected battery INA address: %d", settings.DeviceBatteryInaAddress)
	}
	if settings.PowermonEnables != 3 {
		t.Fatalf("unexpected powermon enables: %d", settings.PowermonEnables)
	}
}

func TestNodeSettingsServiceSavePowerSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_power", "commit"}
	packetIDs := []uint32{600, 601, 602}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_power":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				power := cfg.GetPower()
				if power == nil {
					t.Fatalf("expected power config payload")
				}
				if !power.GetIsPowerSaving() {
					t.Fatalf("expected power saving payload")
				}
				if power.GetOnBatteryShutdownAfterSecs() != 3600 {
					t.Fatalf("unexpected shutdown on power loss payload")
				}
				if power.GetAdcMultiplierOverride() != 1.25 {
					t.Fatalf("unexpected ADC multiplier override payload")
				}
				if power.GetWaitBluetoothSecs() != 300 {
					t.Fatalf("unexpected wait bluetooth seconds payload")
				}
				if power.GetSdsSecs() != 43200 {
					t.Fatalf("unexpected super deep sleep seconds payload")
				}
				if power.GetLsSecs() != 600 {
					t.Fatalf("unexpected light sleep seconds payload")
				}
				if power.GetMinWakeSecs() != 20 {
					t.Fatalf("unexpected minimum wake seconds payload")
				}
				if power.GetDeviceBatteryInaAddress() != 64 {
					t.Fatalf("unexpected battery INA address payload")
				}
				if power.GetPowermonEnables() != 7 {
					t.Fatalf("unexpected powermon enables payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SavePowerSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodePowerSettings{
		NodeID:                     "!00000001",
		IsPowerSaving:              true,
		OnBatteryShutdownAfterSecs: 3600,
		AdcMultiplierOverride:      1.25,
		WaitBluetoothSecs:          300,
		SdsSecs:                    43200,
		LsSecs:                     600,
		MinWakeSecs:                20,
		DeviceBatteryInaAddress:    64,
		PowermonEnables:            7,
	})
	if err != nil {
		t.Fatalf("save power settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadDisplaySettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_DISPLAY_CONFIG {
				t.Fatalf("expected display config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get display config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   170,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Display{
								Display: &generated.Config_DisplayConfig{
									ScreenOnSecs:           600,
									AutoScreenCarouselSecs: 15,
									CompassNorthTop:        true,
									FlipScreen:             true,
									Units:                  generated.Config_DisplayConfig_IMPERIAL,
									Oled:                   generated.Config_DisplayConfig_OLED_SH1106,
									Displaymode:            generated.Config_DisplayConfig_INVERTED,
									HeadingBold:            true,
									WakeOnTapOrMotion:      true,
									CompassOrientation:     generated.Config_DisplayConfig_DEGREES_180_INVERTED,
									Use_12HClock:           true,
								},
							},
						},
					},
				},
			})

			return "170", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadDisplaySettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load display settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if settings.ScreenOnSecs != 600 {
		t.Fatalf("unexpected screen on seconds: %d", settings.ScreenOnSecs)
	}
	if settings.AutoScreenCarouselSecs != 15 {
		t.Fatalf("unexpected auto screen carousel seconds: %d", settings.AutoScreenCarouselSecs)
	}
	if !settings.CompassNorthTop {
		t.Fatalf("expected compass north top to be true")
	}
	if !settings.FlipScreen {
		t.Fatalf("expected flip screen to be true")
	}
	if settings.Units != int32(generated.Config_DisplayConfig_IMPERIAL) {
		t.Fatalf("unexpected display units: %d", settings.Units)
	}
	if settings.Oled != int32(generated.Config_DisplayConfig_OLED_SH1106) {
		t.Fatalf("unexpected OLED type: %d", settings.Oled)
	}
	if settings.DisplayMode != int32(generated.Config_DisplayConfig_INVERTED) {
		t.Fatalf("unexpected display mode: %d", settings.DisplayMode)
	}
	if !settings.HeadingBold {
		t.Fatalf("expected heading bold to be true")
	}
	if !settings.WakeOnTapOrMotion {
		t.Fatalf("expected wake on tap or motion to be true")
	}
	if settings.CompassOrientation != int32(generated.Config_DisplayConfig_DEGREES_180_INVERTED) {
		t.Fatalf("unexpected compass orientation: %d", settings.CompassOrientation)
	}
	if !settings.Use12HClock {
		t.Fatalf("expected use 12-hour clock to be true")
	}
}

func TestNodeSettingsServiceSaveDisplaySettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_display", "commit"}
	packetIDs := []uint32{700, 701, 702}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_display":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				display := cfg.GetDisplay()
				if display == nil {
					t.Fatalf("expected display config payload")
				}
				if display.GetScreenOnSecs() != 900 {
					t.Fatalf("unexpected screen on seconds payload")
				}
				if display.GetAutoScreenCarouselSecs() != 30 {
					t.Fatalf("unexpected carousel seconds payload")
				}
				//nolint:staticcheck // Kept for Android parity while this proto field remains present upstream.
				if !display.GetCompassNorthTop() {
					t.Fatalf("expected compass north top payload")
				}
				if !display.GetFlipScreen() {
					t.Fatalf("expected flip screen payload")
				}
				if display.GetUnits() != generated.Config_DisplayConfig_IMPERIAL {
					t.Fatalf("unexpected display units payload")
				}
				if display.GetOled() != generated.Config_DisplayConfig_OLED_SH1107 {
					t.Fatalf("unexpected OLED type payload")
				}
				if display.GetDisplaymode() != generated.Config_DisplayConfig_TWOCOLOR {
					t.Fatalf("unexpected display mode payload")
				}
				if !display.GetHeadingBold() {
					t.Fatalf("expected heading bold payload")
				}
				if !display.GetWakeOnTapOrMotion() {
					t.Fatalf("expected wake on tap or motion payload")
				}
				if display.GetCompassOrientation() != generated.Config_DisplayConfig_DEGREES_270 {
					t.Fatalf("unexpected compass orientation payload")
				}
				if !display.GetUse_12HClock() {
					t.Fatalf("expected use 12-hour clock payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SaveDisplaySettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeDisplaySettings{
		NodeID:                 "!00000001",
		ScreenOnSecs:           900,
		AutoScreenCarouselSecs: 30,
		CompassNorthTop:        true,
		FlipScreen:             true,
		Units:                  int32(generated.Config_DisplayConfig_IMPERIAL),
		Oled:                   int32(generated.Config_DisplayConfig_OLED_SH1107),
		DisplayMode:            int32(generated.Config_DisplayConfig_TWOCOLOR),
		HeadingBold:            true,
		WakeOnTapOrMotion:      true,
		CompassOrientation:     int32(generated.Config_DisplayConfig_DEGREES_270),
		Use12HClock:            true,
	})
	if err != nil {
		t.Fatalf("save display settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func TestNodeSettingsServiceLoadBluetoothSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	sender := stubAdminSender{
		send: func(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_BLUETOOTH_CONFIG {
				t.Fatalf("expected bluetooth config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get bluetooth config request")
			}
			messageBus.Publish(connectors.TopicAdminMessage, radio.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   180,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetConfigResponse{
						GetConfigResponse: &generated.Config{
							PayloadVariant: &generated.Config_Bluetooth{
								Bluetooth: &generated.Config_BluetoothConfig{
									Enabled:  true,
									Mode:     generated.Config_BluetoothConfig_FIXED_PIN,
									FixedPin: 123456,
								},
							},
						},
					},
				},
			})

			return "180", nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{}, false
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	settings, err := service.LoadBluetoothSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load bluetooth settings: %v", err)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if !settings.Enabled {
		t.Fatalf("expected bluetooth enabled to be true")
	}
	if settings.Mode != int32(generated.Config_BluetoothConfig_FIXED_PIN) {
		t.Fatalf("unexpected pairing mode: %d", settings.Mode)
	}
	if settings.FixedPIN != 123456 {
		t.Fatalf("unexpected fixed pin: %d", settings.FixedPIN)
	}
}

func TestNodeSettingsServiceSaveBluetoothSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	expectedPayloadKinds := []string{"begin", "set_bluetooth", "commit"}
	packetIDs := []uint32{800, 801, 802}
	call := 0
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			if call >= len(expectedPayloadKinds) {
				t.Fatalf("unexpected send call %d", call)
			}
			switch expectedPayloadKinds[call] {
			case "begin":
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case "set_bluetooth":
				cfg := payload.GetSetConfig()
				if cfg == nil {
					t.Fatalf("expected set config payload")
				}
				bluetooth := cfg.GetBluetooth()
				if bluetooth == nil {
					t.Fatalf("expected bluetooth config payload")
				}
				if !bluetooth.GetEnabled() {
					t.Fatalf("expected bluetooth enabled payload")
				}
				if bluetooth.GetMode() != generated.Config_BluetoothConfig_FIXED_PIN {
					t.Fatalf("unexpected pairing mode payload")
				}
				if bluetooth.GetFixedPin() != 654321 {
					t.Fatalf("unexpected fixed pin payload")
				}
			case "commit":
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unknown expected payload kind at call %d", call)
			}

			packetID := packetIDs[call]
			messageBus.Publish(connectors.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})
			call++

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (connectors.ConnectionStatus, bool) {
			return connectors.ConnectionStatus{State: connectors.ConnectionStateConnected}, true
		},
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := service.SaveBluetoothSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeBluetoothSettings{
		NodeID:   "!00000001",
		Enabled:  true,
		Mode:     int32(generated.Config_BluetoothConfig_FIXED_PIN),
		FixedPIN: 654321,
	})
	if err != nil {
		t.Fatalf("save bluetooth settings: %v", err)
	}
	if call != len(expectedPayloadKinds) {
		t.Fatalf("unexpected send calls count: got %d want %d", call, len(expectedPayloadKinds))
	}
}

func stringFromUint32(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}

func float64Ptr(v float64) *float64 {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}
