package ui

import (
	"testing"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeNetworkIPv4MappingMatchesAndroidByteOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		text   string
		packed uint32
	}{
		{name: "zero", text: "0.0.0.0", packed: 0},
		{name: "private address", text: "192.168.1.1", packed: 16885952},
		{name: "dns address", text: "8.8.4.4", packed: 67373064},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if formatted := nodeNetworkFormatIPv4(tc.packed); formatted != tc.text {
				t.Fatalf("formatted IPv4 = %q, want %q", formatted, tc.text)
			}
			parsed, err := nodeNetworkParseIPv4(tc.text)
			if err != nil {
				t.Fatalf("parse IPv4: %v", err)
			}
			if parsed != tc.packed {
				t.Fatalf("parsed IPv4 = %d, want %d", parsed, tc.packed)
			}
		})
	}
}

func TestNodeNetworkParseIPv4RejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"192.168.1", "192.168.1.256", "::1", "not-an-address"} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if _, err := nodeNetworkParseIPv4(value); err == nil {
				t.Fatalf("expected %q to be rejected", value)
			}
		})
	}
}

func TestNodeNetworkUDPBroadcastTogglePreservesUnknownProtocolFlags(t *testing.T) {
	t.Parallel()

	const unknownFlag uint32 = 1 << 7
	udpFlag := uint32(generated.Config_NetworkConfig_UDP_BROADCAST)

	enabled := nodeNetworkSetProtocolEnabled(unknownFlag, udpFlag, true)
	if !nodeNetworkProtocolEnabled(enabled, udpFlag) {
		t.Fatal("expected UDP broadcast flag to be enabled")
	}
	if enabled&unknownFlag == 0 {
		t.Fatal("expected unknown protocol flag to be preserved when enabling UDP")
	}

	disabled := nodeNetworkSetProtocolEnabled(enabled, udpFlag, false)
	if nodeNetworkProtocolEnabled(disabled, udpFlag) {
		t.Fatal("expected UDP broadcast flag to be disabled")
	}
	if disabled != unknownFlag {
		t.Fatalf("protocol flags = %d, want preserved unknown flag %d", disabled, unknownFlag)
	}
}

func TestNodeNetworkAddressModeOptionsMatchProtocol(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label string
		value generated.Config_NetworkConfig_AddressMode
	}{
		{label: "DHCP", value: generated.Config_NetworkConfig_DHCP},
		{label: "Static", value: generated.Config_NetworkConfig_STATIC},
	}

	for _, tc := range testCases {
		parsed, err := nodeSettingsParseInt32SelectLabel("address mode", tc.label, nodeNetworkAddressModeOptions)
		if err != nil {
			t.Fatalf("parse address mode %q: %v", tc.label, err)
		}
		if parsed != int32(tc.value) {
			t.Fatalf("address mode %q = %d, want %d", tc.label, parsed, tc.value)
		}
	}
}
