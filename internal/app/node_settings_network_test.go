package app

import (
	"testing"

	"github.com/skobkin/meshgo/internal/bus"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeSettingsServiceLoadNetworkSettings_MatchesReplyID(t *testing.T) {
	t.Parallel()

	var messageBus bus.MessageBus
	sender := stubAdminSender{
		send: func(to uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if payload.GetGetConfigRequest() != generated.AdminMessage_NETWORK_CONFIG {
				t.Fatalf("expected network config request payload")
			}
			if !wantResponse {
				t.Fatalf("expected wantResponse=true for network config load")
			}

			publishAdminReply(messageBus, to, 64, &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_GetConfigResponse{
					GetConfigResponse: &generated.Config{
						PayloadVariant: &generated.Config_Network{
							Network: &generated.Config_NetworkConfig{
								WifiEnabled:      true,
								WifiSsid:         "mesh-wifi",
								WifiPsk:          "topsecret",
								NtpServer:        "pool.ntp.org",
								EthEnabled:       true,
								AddressMode:      generated.Config_NetworkConfig_STATIC,
								RsyslogServer:    "loghost",
								EnabledProtocols: 7,
								Ipv6Enabled:      true,
								Ipv4Config: &generated.Config_NetworkConfig_IpV4Config{
									Ip:      1,
									Gateway: 2,
									Subnet:  3,
									Dns:     4,
								},
							},
						},
					},
				},
			})

			return "64", nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, false)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	settings, err := service.LoadNetworkSettings(ctx, mustHexNodeTarget())
	if err != nil {
		t.Fatalf("load network settings: %v", err)
	}
	if !settings.WifiEnabled || settings.WifiSSID != "mesh-wifi" || settings.AddressMode != int32(generated.Config_NetworkConfig_STATIC) {
		t.Fatalf("unexpected network settings: %+v", settings)
	}
	if settings.IPv4Address != 1 || settings.IPv4Gateway != 2 || settings.IPv4Subnet != 3 || settings.IPv4DNS != 4 {
		t.Fatalf("unexpected IPv4 settings: %+v", settings)
	}
}

func TestNodeSettingsServiceSaveNetworkSettings_ImmediateStatusEvents(t *testing.T) {
	t.Parallel()

	var messageBus bus.MessageBus
	call := 0
	packetIDs := []uint32{100, 101, 102}
	sender := stubAdminSender{
		send: func(_ uint32, _ uint32, wantResponse bool, payload *generated.AdminMessage) (string, error) {
			if wantResponse {
				t.Fatalf("expected wantResponse=false for save flow")
			}
			switch call {
			case 0:
				if !payload.GetBeginEditSettings() {
					t.Fatalf("expected begin edit settings payload")
				}
			case 1:
				network := payload.GetSetConfig().GetNetwork()
				if network == nil {
					t.Fatalf("expected network config payload")
				}
				if network.GetWifiSsid() != "mesh-wifi" || network.GetAddressMode() != generated.Config_NetworkConfig_STATIC {
					t.Fatalf("unexpected network payload: %+v", network)
				}
				if network.GetIpv4Config().GetDns() != 40 {
					t.Fatalf("unexpected IPv4 dns payload: %d", network.GetIpv4Config().GetDns())
				}
			case 2:
				if !payload.GetCommitEditSettings() {
					t.Fatalf("expected commit edit settings payload")
				}
			default:
				t.Fatalf("unexpected send call %d", call)
			}
			publishSentStatus(messageBus, packetIDs[call])
			call++

			return stringFromUint32(packetIDs[call-1]), nil
		},
	}
	service, busRef := newTestNodeSettingsService(t, sender, true)
	messageBus = busRef

	ctx, cancel := contextWithTimeout(t)
	defer cancel()

	err := service.SaveNetworkSettings(ctx, mustLocalNodeTarget(), NodeNetworkSettings{
		NodeID:           "!00000001",
		WifiEnabled:      true,
		WifiSSID:         "mesh-wifi",
		WifiPSK:          "topsecret",
		NTPServer:        "pool.ntp.org",
		EthernetEnabled:  true,
		AddressMode:      int32(generated.Config_NetworkConfig_STATIC),
		IPv4Address:      10,
		IPv4Gateway:      20,
		IPv4Subnet:       30,
		IPv4DNS:          40,
		RsyslogServer:    "loghost",
		EnabledProtocols: 7,
		IPv6Enabled:      true,
	})
	if err != nil {
		t.Fatalf("save network settings: %v", err)
	}
	if call != 3 {
		t.Fatalf("unexpected send calls count: %d", call)
	}
}
