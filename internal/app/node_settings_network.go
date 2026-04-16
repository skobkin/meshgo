package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadNetworkSettings(ctx context.Context, target NodeSettingsTarget) (NodeNetworkSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_NETWORK_CONFIG, "get_config.network")
	if err != nil {
		return NodeNetworkSettings{}, err
	}
	network := cfg.GetNetwork()
	if network == nil {
		return NodeNetworkSettings{}, fmt.Errorf("network config payload is empty")
	}

	settings := NodeNetworkSettings{
		NodeID:           strings.TrimSpace(target.NodeID),
		WifiEnabled:      network.GetWifiEnabled(),
		WifiSSID:         network.GetWifiSsid(),
		WifiPSK:          network.GetWifiPsk(),
		NTPServer:        network.GetNtpServer(),
		EthernetEnabled:  network.GetEthEnabled(),
		AddressMode:      int32(network.GetAddressMode()),
		RsyslogServer:    network.GetRsyslogServer(),
		EnabledProtocols: network.GetEnabledProtocols(),
		IPv6Enabled:      network.GetIpv6Enabled(),
	}
	if ipv4 := network.GetIpv4Config(); ipv4 != nil {
		settings.IPv4Address = ipv4.GetIp()
		settings.IPv4Gateway = ipv4.GetGateway()
		settings.IPv4Subnet = ipv4.GetSubnet()
		settings.IPv4DNS = ipv4.GetDns()
	}

	return settings, nil
}

func (s *NodeSettingsService) SaveNetworkSettings(ctx context.Context, target NodeSettingsTarget, settings NodeNetworkSettings) error {
	return s.saveConfig(ctx, target, "set_config.network", &generated.Config{
		PayloadVariant: &generated.Config_Network{
			Network: &generated.Config_NetworkConfig{
				WifiEnabled:      settings.WifiEnabled,
				WifiSsid:         settings.WifiSSID,
				WifiPsk:          settings.WifiPSK,
				NtpServer:        strings.TrimSpace(settings.NTPServer),
				EthEnabled:       settings.EthernetEnabled,
				AddressMode:      generated.Config_NetworkConfig_AddressMode(settings.AddressMode),
				RsyslogServer:    strings.TrimSpace(settings.RsyslogServer),
				EnabledProtocols: settings.EnabledProtocols,
				Ipv6Enabled:      settings.IPv6Enabled,
				Ipv4Config: &generated.Config_NetworkConfig_IpV4Config{
					Ip:      settings.IPv4Address,
					Gateway: settings.IPv4Gateway,
					Subnet:  settings.IPv4Subnet,
					Dns:     settings.IPv4DNS,
				},
			},
		},
	})
}
