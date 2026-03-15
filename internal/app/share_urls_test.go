package app

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

func TestBuildChannelShareURLReplace(t *testing.T) {
	rawURL, err := BuildChannelShareURL([]NodeChannelSettings{
		{
			Name:            "General",
			PSK:             []byte{0x01},
			ID:              11,
			UplinkEnabled:   true,
			DownlinkEnabled: true,
		},
	}, NodeLoRaSettings{
		UsePreset:   true,
		ModemPreset: int32(generated.Config_LoRaConfig_LONG_FAST),
		Region:      int32(generated.Config_LoRaConfig_EU_868),
	}, false)
	if err != nil {
		t.Fatalf("build replace URL: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != channelSharePrefix {
		t.Fatalf("unexpected channel URL prefix: %q", got)
	}
	if parsed.RawQuery != "" {
		t.Fatalf("expected no query, got %q", parsed.RawQuery)
	}

	var set generated.ChannelSet
	if err := proto.Unmarshal(decodeRawFragment(t, parsed.Fragment), &set); err != nil {
		t.Fatalf("decode channel set: %v", err)
	}
	if len(set.GetSettings()) != 1 {
		t.Fatalf("expected one channel, got %d", len(set.GetSettings()))
	}
	if got := set.GetSettings()[0].GetName(); got != "General" {
		t.Fatalf("unexpected channel name: %q", got)
	}
	if set.GetLoraConfig().GetModemPreset() != generated.Config_LoRaConfig_LONG_FAST {
		t.Fatalf("unexpected modem preset: %v", set.GetLoraConfig().GetModemPreset())
	}
}

func TestBuildChannelShareURLMatchesAndroidPresetPayload(t *testing.T) {
	rawURL, err := BuildChannelShareURL([]NodeChannelSettings{
		{
			PSK:             []byte{0x01},
			UplinkEnabled:   true,
			DownlinkEnabled: true,
		},
	}, NodeLoRaSettings{
		UsePreset:           true,
		ModemPreset:         int32(generated.Config_LoRaConfig_LONG_FAST),
		Bandwidth:           250,
		SpreadFactor:        11,
		CodingRate:          5,
		Region:              int32(generated.Config_LoRaConfig_RU),
		HopLimit:            7,
		TxEnabled:           true,
		TxPower:             20,
		Sx126XRxBoostedGain: true,
		ConfigOkToMqtt:      true,
	}, false)
	if err != nil {
		t.Fatalf("build Android-compatible replace URL: %v", err)
	}

	const want = "https://meshtastic.org/e/#CgcSAQEoATABEhYIARj6ASALOAlAB0gBUBRYAmgByAYB"
	if rawURL != want {
		t.Fatalf("unexpected Android-compatible URL:\nwant: %s\ngot:  %s", want, rawURL)
	}
}

func TestBuildChannelShareURLAdd(t *testing.T) {
	rawURL, err := BuildChannelShareURL([]NodeChannelSettings{
		{
			PSK:             []byte{0x01},
			UplinkEnabled:   true,
			DownlinkEnabled: true,
		},
	}, NodeLoRaSettings{
		UsePreset:           true,
		ModemPreset:         int32(generated.Config_LoRaConfig_LONG_FAST),
		Bandwidth:           250,
		SpreadFactor:        11,
		CodingRate:          5,
		Region:              int32(generated.Config_LoRaConfig_RU),
		HopLimit:            7,
		TxEnabled:           true,
		TxPower:             20,
		Sx126XRxBoostedGain: true,
		ConfigOkToMqtt:      true,
	}, true)
	if err != nil {
		t.Fatalf("build add URL: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.RawQuery; got != channelShareAddFlag {
		t.Fatalf("unexpected add query: %q", got)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path + "#" + parsed.Fragment; got != "https://meshtastic.org/e/#CgcSAQEoATABEhYIARj6ASALOAlAB0gBUBRYAmgByAYB" {
		t.Fatalf("unexpected add payload URL body: %q", got)
	}
}

func TestBuildSharedContactURL(t *testing.T) {
	isUnmessageable := true
	const roleRouter = "ROUTER"
	rawURL, err := BuildSharedContactURL(domain.Node{
		NodeID:          "!0000002a",
		LongName:        "Alpha",
		ShortName:       "AL",
		PublicKey:       []byte{1, 2, 3},
		BoardModel:      generated.HardwareModel_T_ECHO.String(),
		Role:            roleRouter,
		IsUnmessageable: &isUnmessageable,
	})
	if err != nil {
		t.Fatalf("build shared contact URL: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path + "#"; got != contactSharePrefix {
		t.Fatalf("unexpected contact URL prefix: %q", got)
	}

	var contact generated.SharedContact
	if err := proto.Unmarshal(decodeRawFragment(t, parsed.Fragment), &contact); err != nil {
		t.Fatalf("decode shared contact: %v", err)
	}
	if contact.GetNodeNum() != 0x2a {
		t.Fatalf("unexpected node num: %d", contact.GetNodeNum())
	}
	if got := strings.TrimSpace(contact.GetUser().GetLongName()); got != "Alpha" {
		t.Fatalf("unexpected long name: %q", got)
	}
	if got := contact.GetUser().GetHwModel(); got != generated.HardwareModel_T_ECHO {
		t.Fatalf("unexpected hw model: %v", got)
	}
	if got := contact.GetUser().GetRole().String(); got != roleRouter {
		t.Fatalf("unexpected role: %v", got)
	}
	if !contact.GetUser().GetIsUnmessagable() {
		t.Fatalf("expected unmessageable flag to be preserved")
	}
}

func decodeRawFragment(t *testing.T, fragment string) []byte {
	t.Helper()

	out, err := base64.RawURLEncoding.DecodeString(fragment)
	if err != nil {
		t.Fatalf("decode fragment: %v", err)
	}

	return out
}
