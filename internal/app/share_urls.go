package app

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

const (
	meshtasticHost      = "meshtastic.org"
	channelSharePath    = "/e/"
	contactSharePath    = "/v/"
	channelSharePrefix  = "https://" + meshtasticHost + channelSharePath
	contactSharePrefix  = "https://" + meshtasticHost + contactSharePath + "#"
	channelShareAddFlag = "add=true"
)

func BuildChannelShareURL(settings []NodeChannelSettings, lora NodeLoRaSettings, shouldAdd bool) (string, error) {
	if len(settings) == 0 {
		return "", fmt.Errorf("at least one channel is required")
	}

	channelSet := &generated.ChannelSet{
		Settings:   make([]*generated.ChannelSettings, 0, len(settings)),
		LoraConfig: cloneLoRaConfigForShare(settings, lora),
	}
	for _, item := range settings {
		channelSet.Settings = append(channelSet.Settings, cloneChannelSettingsForShare(item))
	}

	encoded, err := proto.Marshal(channelSet)
	if err != nil {
		return "", fmt.Errorf("marshal channel set: %w", err)
	}

	rawURL := channelSharePrefix
	if shouldAdd {
		rawURL += "?" + channelShareAddFlag
	}
	rawURL += "#" + base64.RawURLEncoding.EncodeToString(encoded)

	return rawURL, nil
}

func BuildSharedContactURL(node domain.Node) (string, error) {
	nodeID := strings.TrimSpace(node.NodeID)
	if nodeID == "" {
		return "", fmt.Errorf("node id is required")
	}

	nodeNum, err := parseNodeID(nodeID)
	if err != nil {
		return "", err
	}

	user := &generated.User{
		Id:        nodeID,
		LongName:  strings.TrimSpace(node.LongName),
		ShortName: strings.TrimSpace(node.ShortName),
	}
	if len(node.PublicKey) > 0 {
		user.PublicKey = cloneBytes(node.PublicKey)
	}
	if node.IsUnmessageable != nil {
		user.IsUnmessagable = boolPtr(*node.IsUnmessageable)
	}

	contact := &generated.SharedContact{
		NodeNum: nodeNum,
		User:    user,
	}
	encoded, err := proto.Marshal(contact)
	if err != nil {
		return "", fmt.Errorf("marshal shared contact: %w", err)
	}

	return contactSharePrefix + base64.RawURLEncoding.EncodeToString(encoded), nil
}

func cloneChannelSettingsForShare(settings NodeChannelSettings) *generated.ChannelSettings {
	shared := &generated.ChannelSettings{
		Psk:             cloneBytes(settings.PSK),
		Name:            strings.TrimSpace(settings.Name),
		Id:              settings.ID,
		UplinkEnabled:   settings.UplinkEnabled,
		DownlinkEnabled: settings.DownlinkEnabled,
	}

	if settings.PositionPrecision == 0 && !settings.Muted {
		return shared
	}

	shared.ModuleSettings = &generated.ModuleSettings{
		PositionPrecision: settings.PositionPrecision,
		IsMuted:           settings.Muted,
	}

	return shared
}

func cloneLoRaConfigForShare(channels []NodeChannelSettings, settings NodeLoRaSettings) *generated.Config_LoRaConfig {
	shared := &generated.Config_LoRaConfig{
		UsePreset:           settings.UsePreset,
		ModemPreset:         generated.Config_LoRaConfig_ModemPreset(settings.ModemPreset),
		Bandwidth:           settings.Bandwidth,
		SpreadFactor:        settings.SpreadFactor,
		CodingRate:          settings.CodingRate,
		FrequencyOffset:     settings.FrequencyOffset,
		Region:              generated.Config_LoRaConfig_RegionCode(settings.Region),
		HopLimit:            settings.HopLimit,
		TxEnabled:           settings.TxEnabled,
		TxPower:             settings.TxPower,
		ChannelNum:          settings.ChannelNum,
		OverrideDutyCycle:   settings.OverrideDutyCycle,
		Sx126XRxBoostedGain: settings.Sx126XRxBoostedGain,
		OverrideFrequency:   settings.OverrideFrequency,
		PaFanDisabled:       settings.PaFanDisabled,
		IgnoreIncoming:      cloneUint32Slice(settings.IgnoreIncoming),
		IgnoreMqtt:          settings.IgnoreMqtt,
		ConfigOkToMqtt:      settings.ConfigOkToMqtt,
	}

	if !settings.UsePreset {
		return shared
	}

	// Android share links normalize preset-based channel sets by omitting explicit coding rate
	// and making the hashed channel slot explicit when the device config leaves it unset.
	shared.CodingRate = 0
	if shared.ChannelNum == 0 {
		shared.ChannelNum = deriveShareChannelNum(channels, settings)
	}

	return shared
}

func deriveShareChannelNum(channels []NodeChannelSettings, settings NodeLoRaSettings) uint32 {
	primaryName := LoRaPrimaryChannelTitle(settings, "")
	if len(channels) > 0 {
		primaryName = LoRaPrimaryChannelTitle(settings, channels[0].Name)
	}
	if primaryName == "" {
		return 0
	}

	numChannels := LoRaNumChannels(settings)
	if numChannels <= 0 {
		return 0
	}

	return LoRaEffectiveChannelNum(settings, primaryName)
}
