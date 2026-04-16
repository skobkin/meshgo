package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

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
	addressMode := widget.NewEntry()
	addressMode.OnChanged = func(string) { onChanged() }
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
	enabledProtocols := widget.NewEntry()
	enabledProtocols.OnChanged = func(string) { onChanged() }
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
		widget.NewFormItem("Enabled protocols", enabledProtocols),
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
			addressMode.SetText(strconv.FormatInt(int64(v.AddressMode), 10))
			ipv4Address.SetText(strconv.FormatUint(uint64(v.IPv4Address), 10))
			ipv4Gateway.SetText(strconv.FormatUint(uint64(v.IPv4Gateway), 10))
			ipv4Subnet.SetText(strconv.FormatUint(uint64(v.IPv4Subnet), 10))
			ipv4DNS.SetText(strconv.FormatUint(uint64(v.IPv4DNS), 10))
			rsyslogServer.SetText(v.RsyslogServer)
			enabledProtocols.SetText(strconv.FormatUint(uint64(v.EnabledProtocols), 10))
			ipv6Enabled.SetChecked(v.IPv6Enabled)
		},
		read: func(base app.NodeNetworkSettings, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.WifiEnabled = wifiEnabled.Checked
			base.WifiSSID = strings.TrimSpace(wifiSSID.Text)
			base.WifiPSK = wifiPSK.Text
			base.NTPServer = strings.TrimSpace(ntpServer.Text)
			base.EthernetEnabled = ethernetEnabled.Checked
			value, err := parseOptionalInt32(addressMode.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("address mode", err)
			}
			base.AddressMode = value
			base.IPv4Address, err = parseOptionalUint32(ipv4Address.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 address", err)
			}
			base.IPv4Gateway, err = parseOptionalUint32(ipv4Gateway.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 gateway", err)
			}
			base.IPv4Subnet, err = parseOptionalUint32(ipv4Subnet.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 subnet", err)
			}
			base.IPv4DNS, err = parseOptionalUint32(ipv4DNS.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 DNS", err)
			}
			base.RsyslogServer = strings.TrimSpace(rsyslogServer.Text)
			base.EnabledProtocols, err = parseOptionalUint32(enabledProtocols.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("enabled protocols", err)
			}
			base.IPv6Enabled = ipv6Enabled.Checked

			return base, nil
		},
		setSaving: disableWidgets(
			wifiEnabled, wifiSSID, wifiPSK, ntpServer, ethernetEnabled, addressMode, ipv4Address,
			ipv4Gateway, ipv4Subnet, ipv4DNS, rsyslogServer, enabledProtocols, ipv6Enabled,
		),
	}
}
