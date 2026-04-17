package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

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
			messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
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
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{}, false
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
			messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
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
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
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
			messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
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
		func() (busmsg.ConnectionStatus, bool) {
			return busmsg.ConnectionStatus{State: busmsg.ConnectionStateConnected}, true
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

func float64Ptr(v float64) *float64 {
	return &v
}

func int32Ptr(v int32) *int32 {
	return &v
}
