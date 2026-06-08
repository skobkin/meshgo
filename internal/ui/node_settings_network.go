package ui

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

var nodeNetworkAddressModeOptions = []nodeSettingsInt32Option{
	{Label: "DHCP", Value: int32(generated.Config_NetworkConfig_DHCP)},
	{Label: "Static", Value: int32(generated.Config_NetworkConfig_STATIC)},
}

func newNodeNetworkSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep,
		saveGate,
		"device.network",
		"Loading network settings…",
		"Network settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
			return dep.Actions.NodeSettings.LoadNetworkSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNetworkSettings) error {
			return dep.Actions.NodeSettings.SaveNetworkSettings(ctx, target, settings)
		},
		func(v app.NodeNetworkSettings) app.NodeNetworkSettings { return v },
		func(v app.NodeNetworkSettings) string {
			return fmt.Sprintf("%s|%t|%s|%s|%s|%t|%d|%d|%d|%d|%d|%s|%d|%t",
				v.NodeID, v.WifiEnabled, v.WifiSSID, v.WifiPSK, v.NTPServer, v.EthernetEnabled, v.AddressMode,
				v.IPv4Address, v.IPv4Gateway, v.IPv4Subnet, v.IPv4DNS, v.RsyslogServer, v.EnabledProtocols, v.IPv6Enabled)
		},
		buildNodeNetworkSettingsForm,
	)
}

func buildNodeNetworkSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeNetworkSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	wifiEnabled := newSettingsCheck(onChanged)
	wifiSSID := widget.NewEntry()
	wifiSSID.OnChanged = func(string) { onChanged() }
	wifiPSK := widget.NewPasswordEntry()
	wifiPSK.OnChanged = func(string) { onChanged() }
	ntpServer := widget.NewEntry()
	ntpServer.OnChanged = func(string) { onChanged() }
	ethernetEnabled := newSettingsCheck(onChanged)
	addressMode := widget.NewSelect(nil, func(string) { onChanged() })
	ipv4Address := widget.NewEntry()
	ipv4Address.OnChanged = func(string) { onChanged() }
	ipv4Gateway := widget.NewEntry()
	ipv4Gateway.OnChanged = func(string) { onChanged() }
	ipv4Subnet := widget.NewEntry()
	ipv4Subnet.OnChanged = func(string) { onChanged() }
	ipv4DNS := widget.NewEntry()
	ipv4DNS.OnChanged = func(string) { onChanged() }
	rsyslogServer := widget.NewEntry()
	rsyslogServer.OnChanged = func(string) { onChanged() }
	udpBroadcastEnabled := newSettingsCheck(onChanged)
	ipv6Enabled := newSettingsCheck(onChanged)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("WiFi enabled", wifiEnabled),
		widget.NewFormItem("WiFi SSID", wifiSSID),
		widget.NewFormItem("WiFi password", wifiPSK),
		widget.NewFormItem("NTP server", ntpServer),
		widget.NewFormItem("Ethernet enabled", ethernetEnabled),
		widget.NewFormItem("Address mode", addressMode),
		widget.NewFormItem("IPv4 address", ipv4Address),
		widget.NewFormItem("IPv4 gateway", ipv4Gateway),
		widget.NewFormItem("IPv4 subnet", ipv4Subnet),
		widget.NewFormItem("IPv4 DNS", ipv4DNS),
		widget.NewFormItem("Rsyslog server", rsyslogServer),
		widget.NewFormItem("UDP broadcast enabled", udpBroadcastEnabled),
		widget.NewFormItem("IPv6 enabled", ipv6Enabled),
	)

	return nodeManagedSettingsForm[app.NodeNetworkSettings]{
		content: form,
		set: func(v app.NodeNetworkSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			wifiEnabled.SetChecked(v.WifiEnabled)
			wifiSSID.SetText(v.WifiSSID)
			wifiPSK.SetText(v.WifiPSK)
			ntpServer.SetText(v.NTPServer)
			ethernetEnabled.SetChecked(v.EthernetEnabled)
			nodeSettingsSetInt32Select(addressMode, nodeNetworkAddressModeOptions, v.AddressMode, nodeSettingsCustomInt32Label)
			ipv4Address.SetText(nodeNetworkFormatIPv4(v.IPv4Address))
			ipv4Gateway.SetText(nodeNetworkFormatIPv4(v.IPv4Gateway))
			ipv4Subnet.SetText(nodeNetworkFormatIPv4(v.IPv4Subnet))
			ipv4DNS.SetText(nodeNetworkFormatIPv4(v.IPv4DNS))
			rsyslogServer.SetText(v.RsyslogServer)
			udpBroadcastEnabled.SetChecked(nodeNetworkProtocolEnabled(
				v.EnabledProtocols,
				uint32(generated.Config_NetworkConfig_UDP_BROADCAST),
			))
			ipv6Enabled.SetChecked(v.IPv6Enabled)
		},
		read: func(base app.NodeNetworkSettings, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.WifiEnabled = wifiEnabled.Checked
			base.WifiSSID = strings.TrimSpace(wifiSSID.Text)
			base.WifiPSK = wifiPSK.Text
			base.NTPServer = strings.TrimSpace(ntpServer.Text)
			base.EthernetEnabled = ethernetEnabled.Checked
			value, err := nodeSettingsParseInt32SelectLabel("address mode", addressMode.Selected, nodeNetworkAddressModeOptions)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("address mode", err)
			}
			base.AddressMode = value
			base.IPv4Address, err = nodeNetworkParseIPv4(ipv4Address.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 address", err)
			}
			base.IPv4Gateway, err = nodeNetworkParseIPv4(ipv4Gateway.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 gateway", err)
			}
			base.IPv4Subnet, err = nodeNetworkParseIPv4(ipv4Subnet.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 subnet", err)
			}
			base.IPv4DNS, err = nodeNetworkParseIPv4(ipv4DNS.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 DNS", err)
			}
			base.RsyslogServer = strings.TrimSpace(rsyslogServer.Text)
			base.EnabledProtocols = nodeNetworkSetProtocolEnabled(
				base.EnabledProtocols,
				uint32(generated.Config_NetworkConfig_UDP_BROADCAST),
				udpBroadcastEnabled.Checked,
			)
			base.IPv6Enabled = ipv6Enabled.Checked

			return base, nil
		},
		setSaving: disableWidgets(
			wifiEnabled, wifiSSID, wifiPSK, ntpServer, ethernetEnabled, addressMode, ipv4Address,
			ipv4Gateway, ipv4Subnet, ipv4DNS, rsyslogServer, udpBroadcastEnabled, ipv6Enabled,
		),
	}
}

func nodeNetworkFormatIPv4(value uint32) string {
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], value)

	return netip.AddrFrom4(raw).String()
}

func nodeNetworkParseIPv4(raw string) (uint32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	address, err := netip.ParseAddr(value)
	if err != nil || !address.Is4() {
		return 0, fmt.Errorf("must be a dotted-decimal IPv4 address")
	}
	bytes := address.As4()

	return binary.LittleEndian.Uint32(bytes[:]), nil
}

func nodeNetworkProtocolEnabled(protocols uint32, flag uint32) bool {
	return protocols&flag != 0
}

func nodeNetworkSetProtocolEnabled(protocols uint32, flag uint32, enabled bool) uint32 {
	if enabled {
		return protocols | flag
	}

	return protocols &^ flag
}
