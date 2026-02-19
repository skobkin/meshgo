package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeSettingsServiceLoadChannelSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	var (
		call         uint32
		channelIndex int32 = -1
	)
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			call++
			channelIndex++
			request := payload.GetGetChannelRequest()
			if request == 0 {
				t.Fatalf("expected get channel request payload")
			}
			if request != call {
				t.Fatalf("unexpected get channel request index: got %d want %d", request, call)
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for get channel request")
			}

			channel := &generated.Channel{Index: channelIndex, Role: generated.Channel_DISABLED}
			switch call {
			case 1:
				channel.Role = generated.Channel_PRIMARY
				channel.Settings = &generated.ChannelSettings{
					Name:            "",
					Psk:             []byte{1},
					Id:              101,
					UplinkEnabled:   true,
					DownlinkEnabled: false,
					ModuleSettings: &generated.ModuleSettings{
						PositionPrecision: 32,
						IsMuted:           false,
					},
				}
			case 2:
				channel.Role = generated.Channel_SECONDARY
				channel.Settings = &generated.ChannelSettings{
					Name:            "Ops",
					Psk:             bytes.Repeat([]byte{0x11}, 16),
					Id:              202,
					UplinkEnabled:   false,
					DownlinkEnabled: true,
					ModuleSettings: &generated.ModuleSettings{
						PositionPrecision: 13,
						IsMuted:           true,
					},
				}
			default:
				channel.Settings = &generated.ChannelSettings{}
			}

			packetID := uint32(500) + call
			messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
				From:      to,
				RequestID: 777,
				ReplyID:   packetID,
				Message: &generated.AdminMessage{
					PayloadVariant: &generated.AdminMessage_GetChannelResponse{
						GetChannelResponse: channel,
					},
				},
			})

			return stringFromUint32(packetID), nil
		},
	}
	service := NewNodeSettingsService(
		messageBus,
		sender,
		func() (busmsg.ConnectionStatus, bool) { return busmsg.ConnectionStatus{}, false },
		logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	settings, err := service.LoadChannelSettings(ctx, NodeSettingsTarget{NodeID: "!0000002A", IsLocal: true})
	if err != nil {
		t.Fatalf("load channel settings: %v", err)
	}
	if call != uint32(NodeChannelMaxSlots) {
		t.Fatalf("expected %d get channel requests, got %d", NodeChannelMaxSlots, call)
	}
	if settings.NodeID != "!0000002A" {
		t.Fatalf("unexpected node id: %q", settings.NodeID)
	}
	if settings.MaxSlots != NodeChannelMaxSlots {
		t.Fatalf("unexpected max slots: %d", settings.MaxSlots)
	}
	if len(settings.Channels) != 2 {
		t.Fatalf("expected 2 active channels, got %d", len(settings.Channels))
	}
	if settings.Channels[0].ID != 101 {
		t.Fatalf("unexpected primary channel id: %d", settings.Channels[0].ID)
	}
	if settings.Channels[0].PositionPrecision != 32 {
		t.Fatalf("unexpected primary position precision: %d", settings.Channels[0].PositionPrecision)
	}
	if !settings.Channels[0].UplinkEnabled {
		t.Fatalf("expected primary uplink enabled")
	}
	if settings.Channels[1].Name != "Ops" {
		t.Fatalf("unexpected secondary channel name: %q", settings.Channels[1].Name)
	}
	if !settings.Channels[1].Muted {
		t.Fatalf("expected secondary channel muted")
	}
}

func TestNodeSettingsServiceSaveChannelSettings_ChangedOnly(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	currentA := &generated.ChannelSettings{Name: "A", Psk: bytes.Repeat([]byte{0xAA}, 16), Id: 11}
	currentB := &generated.ChannelSettings{Name: "B", Psk: bytes.Repeat([]byte{0xBB}, 16), Id: 22}

	var (
		getCalls      int
		setChannels   []*generated.Channel
		beginCalls    int
		commitCalls   int
		statusPackets uint32 = 1000
	)
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if req := payload.GetGetChannelRequest(); req != 0 {
				if !wantResponse {
					t.Fatalf("expected wantResponse=true for get channel request")
				}
				getCalls++
				channel := &generated.Channel{
					Index:    testChannelIndexByRequest(t, req),
					Role:     generated.Channel_DISABLED,
					Settings: &generated.ChannelSettings{},
				}
				if req == 1 {
					channel.Role = generated.Channel_PRIMARY
					channel.Settings = currentA
				}
				if req == 2 {
					channel.Role = generated.Channel_SECONDARY
					channel.Settings = currentB
				}
				packetID := statusPackets
				statusPackets++
				messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
					From:      to,
					RequestID: 777,
					ReplyID:   packetID,
					Message: &generated.AdminMessage{
						PayloadVariant: &generated.AdminMessage_GetChannelResponse{GetChannelResponse: channel},
					},
				})

				return stringFromUint32(packetID), nil
			}

			if payload.GetBeginEditSettings() {
				if wantResponse {
					t.Fatalf("expected wantResponse=false for begin edit")
				}
				beginCalls++
			} else if set := payload.GetSetChannel(); set != nil {
				if wantResponse {
					t.Fatalf("expected wantResponse=false for set channel")
				}
				setChannels = append(setChannels, set)
			} else if payload.GetCommitEditSettings() {
				if wantResponse {
					t.Fatalf("expected wantResponse=false for commit edit")
				}
				commitCalls++
			} else {
				t.Fatalf("unexpected payload in save flow")
			}

			packetID := statusPackets
			statusPackets++
			messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
				DeviceMessageID: stringFromUint32(packetID),
				Status:          domain.MessageStatusSent,
			})

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := service.SaveChannelSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeChannelSettingsList{
		NodeID:   "!00000001",
		MaxSlots: NodeChannelMaxSlots,
		Channels: []NodeChannelSettings{
			{Name: "B", PSK: bytes.Repeat([]byte{0xBB}, 16), ID: 22},
			{Name: "A", PSK: bytes.Repeat([]byte{0xAA}, 16), ID: 11},
		},
	})
	if err != nil {
		t.Fatalf("save channel settings: %v", err)
	}

	if getCalls != NodeChannelMaxSlots {
		t.Fatalf("expected %d get channel requests, got %d", NodeChannelMaxSlots, getCalls)
	}
	if beginCalls != 1 {
		t.Fatalf("expected begin_edit_settings to be sent once, got %d", beginCalls)
	}
	if commitCalls != 1 {
		t.Fatalf("expected commit_edit_settings to be sent once, got %d", commitCalls)
	}
	if len(setChannels) != 2 {
		t.Fatalf("expected 2 changed channels to be sent, got %d", len(setChannels))
	}

	if setChannels[0].GetIndex() != 0 || setChannels[0].GetRole() != generated.Channel_PRIMARY || setChannels[0].GetSettings().GetName() != "B" {
		t.Fatalf("unexpected first set_channel payload")
	}
	if setChannels[1].GetIndex() != 1 || setChannels[1].GetRole() != generated.Channel_SECONDARY || setChannels[1].GetSettings().GetName() != "A" {
		t.Fatalf("unexpected second set_channel payload")
	}
}

func TestNodeSettingsServiceSaveChannelSettings_StopOnFirstSetFailure(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	messageBus := bus.New(logger)
	defer messageBus.Close()

	var (
		getCalls       int
		setCalls       int
		commitCalls    int
		beginCalls     int
		packetSequence uint32 = 2000
	)
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if req := payload.GetGetChannelRequest(); req != 0 {
				getCalls++
				channel := &generated.Channel{
					Index:    testChannelIndexByRequest(t, req),
					Role:     generated.Channel_DISABLED,
					Settings: &generated.ChannelSettings{},
				}
				if req == 1 {
					channel.Role = generated.Channel_PRIMARY
					channel.Settings = &generated.ChannelSettings{Name: "A", Psk: bytes.Repeat([]byte{0xAA}, 16), Id: 11}
				}
				packetID := packetSequence
				packetSequence++
				messageBus.Publish(bus.TopicAdminMessage, busmsg.AdminMessageEvent{
					From:      to,
					RequestID: 777,
					ReplyID:   packetID,
					Message: &generated.AdminMessage{
						PayloadVariant: &generated.AdminMessage_GetChannelResponse{GetChannelResponse: channel},
					},
				})

				return stringFromUint32(packetID), nil
			}

			packetID := packetSequence
			packetSequence++
			if payload.GetBeginEditSettings() {
				beginCalls++
				messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
					DeviceMessageID: stringFromUint32(packetID),
					Status:          domain.MessageStatusSent,
				})

				return stringFromUint32(packetID), nil
			}
			if payload.GetSetChannel() != nil {
				setCalls++
				status := domain.MessageStatusSent
				reason := ""
				if setCalls == 1 {
					status = domain.MessageStatusFailed
					reason = "mock channel failure"
				}
				messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
					DeviceMessageID: stringFromUint32(packetID),
					Status:          status,
					Reason:          reason,
				})

				return stringFromUint32(packetID), nil
			}
			if payload.GetCommitEditSettings() {
				commitCalls++
				messageBus.Publish(bus.TopicMessageStatus, domain.MessageStatusUpdate{
					DeviceMessageID: stringFromUint32(packetID),
					Status:          domain.MessageStatusSent,
				})

				return stringFromUint32(packetID), nil
			}

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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := service.SaveChannelSettings(ctx, NodeSettingsTarget{NodeID: "!00000001", IsLocal: true}, NodeChannelSettingsList{
		NodeID:   "!00000001",
		MaxSlots: NodeChannelMaxSlots,
		Channels: []NodeChannelSettings{
			{Name: "B", PSK: bytes.Repeat([]byte{0xBB}, 16), ID: 22},
			{Name: "A", PSK: bytes.Repeat([]byte{0xAA}, 16), ID: 11},
		},
	})
	if err == nil {
		t.Fatalf("expected save failure")
	}
	if !strings.Contains(err.Error(), "set channel 1/") {
		t.Fatalf("expected set-channel progress error, got: %v", err)
	}
	if getCalls != NodeChannelMaxSlots {
		t.Fatalf("expected %d get channel requests, got %d", NodeChannelMaxSlots, getCalls)
	}
	if beginCalls != 1 {
		t.Fatalf("expected begin_edit_settings to be sent once, got %d", beginCalls)
	}
	if setCalls != 1 {
		t.Fatalf("expected set_channel to stop after first failure, got %d", setCalls)
	}
	if commitCalls != 0 {
		t.Fatalf("expected commit_edit_settings to be skipped after failure, got %d", commitCalls)
	}
}

func testChannelIndexByRequest(t *testing.T, request uint32) int32 {
	t.Helper()

	switch request {
	case 1:
		return 0
	case 2:
		return 1
	case 3:
		return 2
	case 4:
		return 3
	case 5:
		return 4
	case 6:
		return 5
	case 7:
		return 6
	case 8:
		return 7
	default:
		t.Fatalf("unexpected channel request index: %d", request)

		return 0
	}
}
