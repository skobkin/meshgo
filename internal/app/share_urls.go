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
		LoraConfig: cloneLoRaConfigForShare(lora),
	}
	for _, item := range settings {
		channelSet.Settings = append(channelSet.Settings, nodeChannelSettingsToProto(item))
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

func cloneLoRaConfigForShare(settings NodeLoRaSettings) *generated.Config_LoRaConfig {
	return &generated.Config_LoRaConfig{
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
}
