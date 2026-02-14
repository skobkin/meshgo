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

func stringFromUint32(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
