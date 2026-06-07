package app

import (
	"encoding/base64"
	"fmt"
	"net/url"
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

func ParseChannelShareURL(rawURL string) (*generated.ChannelSet, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse channel URL: %w", err)
	}
	if !strings.EqualFold(parsed.Hostname(), meshtasticHost) &&
		!strings.EqualFold(parsed.Hostname(), "www."+meshtasticHost) {
		return nil, fmt.Errorf("channel URL must use %s", meshtasticHost)
	}

	hasChannelPath := false
	for _, segment := range strings.Split(strings.Trim(parsed.Path, "/"), "/") {
		if strings.EqualFold(segment, "e") {
			hasChannelPath = true

			break
		}
	}
	if !hasChannelPath {
		return nil, fmt.Errorf("channel URL path must contain %q", strings.Trim(channelSharePath, "/"))
	}

	fragment := strings.TrimSpace(parsed.Fragment)
	if fragment == "" {
		return nil, fmt.Errorf("channel URL payload is empty")
	}
	// Older clients put add=true after the fragment payload. It does not change
	// decoding, and profile imports always use replacement semantics.
	fragment, _, _ = strings.Cut(fragment, "?")

	encoded, err := base64.RawURLEncoding.DecodeString(fragment)
	if err != nil {
		encoded, err = base64.URLEncoding.DecodeString(fragment)
		if err != nil {
			return nil, fmt.Errorf("decode channel URL payload: %w", err)
		}
	}

	var channelSet generated.ChannelSet
	if err := proto.Unmarshal(encoded, &channelSet); err != nil {
		return nil, fmt.Errorf("decode channel set: %w", err)
	}
	if len(channelSet.GetSettings()) == 0 {
		return nil, fmt.Errorf("channel URL contains no channels")
	}
	if len(channelSet.GetSettings()) > NodeChannelMaxSlots {
		return nil, fmt.Errorf("channel URL contains %d channels; maximum is %d", len(channelSet.GetSettings()), NodeChannelMaxSlots)
	}

	return &channelSet, nil
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

	contact := &generated.SharedContact{
		NodeNum: nodeNum,
		User:    cloneUserForSharedContact(node),
	}
	encoded, err := proto.Marshal(contact)
	if err != nil {
		return "", fmt.Errorf("marshal shared contact: %w", err)
	}

	return contactSharePrefix + base64.RawURLEncoding.EncodeToString(encoded), nil
}

func cloneUserForSharedContact(node domain.Node) *generated.User {
	user := &generated.User{
		Id:        strings.TrimSpace(node.NodeID),
		LongName:  strings.TrimSpace(node.LongName),
		ShortName: strings.TrimSpace(node.ShortName),
	}
	if len(node.PublicKey) > 0 {
		user.PublicKey = cloneBytes(node.PublicKey)
	}
	if modelValue, ok := generated.HardwareModel_value[strings.TrimSpace(node.BoardModel)]; ok {
		user.HwModel = generated.HardwareModel(modelValue)
	}
	if roleValue, ok := generated.Config_DeviceConfig_Role_value[strings.TrimSpace(node.Role)]; ok {
		user.Role = generated.Config_DeviceConfig_Role(roleValue)
	}
	if node.IsUnmessageable != nil {
		user.IsUnmessagable = boolPtr(*node.IsUnmessageable)
	}

	return user
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
