package app

import (
	"math"
	"strings"
	"unicode/utf16"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const (
	LoRaModemPresetLongFast     = int32(generated.Config_LoRaConfig_LONG_FAST)
	LoRaModemPresetMediumSlow   = int32(generated.Config_LoRaConfig_MEDIUM_SLOW)
	LoRaModemPresetMediumFast   = int32(generated.Config_LoRaConfig_MEDIUM_FAST)
	LoRaModemPresetShortSlow    = int32(generated.Config_LoRaConfig_SHORT_SLOW)
	LoRaModemPresetShortFast    = int32(generated.Config_LoRaConfig_SHORT_FAST)
	LoRaModemPresetLongModerate = int32(generated.Config_LoRaConfig_LONG_MODERATE)
	LoRaModemPresetShortTurbo   = int32(generated.Config_LoRaConfig_SHORT_TURBO)
	LoRaModemPresetLongTurbo    = int32(generated.Config_LoRaConfig_LONG_TURBO)

	// LoRaModemPresetLongSlow and LoRaModemPresetVeryLongSlow are preserved for firmware compatibility; upstream enum names are deprecated.
	LoRaModemPresetLongSlow     int32 = 1
	LoRaModemPresetVeryLongSlow int32 = 2
)

type loRaRegionInfo struct {
	StartMHz float32
	EndMHz   float32
	WideLoRa bool
}

var loRaRegionInfoByCode = map[int32]loRaRegionInfo{
	0:  {StartMHz: 902.0, EndMHz: 928.0},
	1:  {StartMHz: 902.0, EndMHz: 928.0},
	2:  {StartMHz: 433.0, EndMHz: 434.0},
	3:  {StartMHz: 869.4, EndMHz: 869.65},
	4:  {StartMHz: 470.0, EndMHz: 510.0},
	5:  {StartMHz: 920.5, EndMHz: 923.5},
	6:  {StartMHz: 915.0, EndMHz: 928.0},
	7:  {StartMHz: 920.0, EndMHz: 923.0},
	8:  {StartMHz: 920.0, EndMHz: 925.0},
	9:  {StartMHz: 868.7, EndMHz: 869.2},
	10: {StartMHz: 865.0, EndMHz: 867.0},
	11: {StartMHz: 864.0, EndMHz: 868.0},
	12: {StartMHz: 920.0, EndMHz: 925.0},
	13: {StartMHz: 2400.0, EndMHz: 2483.5, WideLoRa: true},
	14: {StartMHz: 433.0, EndMHz: 434.7},
	15: {StartMHz: 868.0, EndMHz: 868.6},
	16: {StartMHz: 433.0, EndMHz: 435.0},
	17: {StartMHz: 919.0, EndMHz: 924.0},
	18: {StartMHz: 917.0, EndMHz: 925.0},
	19: {StartMHz: 433.0, EndMHz: 434.7},
	20: {StartMHz: 868.0, EndMHz: 869.4},
	21: {StartMHz: 915.0, EndMHz: 918.0},
	22: {StartMHz: 433.05, EndMHz: 434.79},
	23: {StartMHz: 433.075, EndMHz: 434.775},
	24: {StartMHz: 863.0, EndMHz: 868.0, WideLoRa: true},
	25: {StartMHz: 865.0, EndMHz: 868.0},
	26: {StartMHz: 902.0, EndMHz: 907.5},
}

var loRaBandwidthByPresetMHz = map[int32]float32{
	LoRaModemPresetVeryLongSlow: 0.0625,
	LoRaModemPresetLongTurbo:    0.5,
	LoRaModemPresetLongFast:     0.25,
	LoRaModemPresetLongModerate: 0.125,
	LoRaModemPresetLongSlow:     0.125,
	LoRaModemPresetMediumFast:   0.25,
	LoRaModemPresetMediumSlow:   0.25,
	LoRaModemPresetShortFast:    0.25,
	LoRaModemPresetShortSlow:    0.25,
	LoRaModemPresetShortTurbo:   0.5,
}

func LoRaPrimaryChannelTitle(settings NodeLoRaSettings, knownTitle string) string {
	if title := strings.TrimSpace(knownTitle); title != "" {
		return title
	}
	if !settings.UsePreset {
		return "Custom"
	}

	switch settings.ModemPreset {
	case LoRaModemPresetShortTurbo:
		return "ShortTurbo"
	case LoRaModemPresetShortFast:
		return "ShortFast"
	case LoRaModemPresetShortSlow:
		return "ShortSlow"
	case LoRaModemPresetMediumFast:
		return "MediumFast"
	case LoRaModemPresetMediumSlow:
		return "MediumSlow"
	case LoRaModemPresetLongFast:
		return "LongFast"
	case LoRaModemPresetLongSlow:
		return "LongSlow"
	case LoRaModemPresetLongModerate:
		return "LongMod"
	case LoRaModemPresetVeryLongSlow:
		return "VLongSlow"
	case LoRaModemPresetLongTurbo:
		return "LongTurbo"
	default:
		return "Invalid"
	}
}

func LoRaNumChannels(settings NodeLoRaSettings) uint32 {
	region, ok := loRaRegionInfoByCode[settings.Region]
	if !ok {
		return 0
	}
	bandwidthMHz := LoRaBandwidthMHz(settings)
	if bandwidthMHz <= 0 {
		return 1
	}
	channelCount := math.Floor(float64((region.EndMHz - region.StartMHz) / bandwidthMHz))
	if channelCount > 0 && channelCount <= float64(^uint32(0)) {
		return uint32(channelCount)
	}

	return 1
}

func LoRaEffectiveChannelNum(settings NodeLoRaSettings, primaryChannelTitle string) uint32 {
	if settings.ChannelNum != 0 {
		return settings.ChannelNum
	}
	channelCount := LoRaNumChannels(settings)
	if channelCount == 0 {
		return 0
	}
	hash := loRaDJB2(primaryChannelTitle)

	return (hash % channelCount) + 1
}

func LoRaEffectiveRadioFreq(settings NodeLoRaSettings, primaryChannelTitle string) float32 {
	if settings.OverrideFrequency != 0 {
		return settings.OverrideFrequency + settings.FrequencyOffset
	}
	region, ok := loRaRegionInfoByCode[settings.Region]
	if !ok {
		return 0
	}
	bandwidthMHz := LoRaBandwidthMHz(settings)
	channelNum := LoRaEffectiveChannelNum(settings, primaryChannelTitle)
	if bandwidthMHz <= 0 || channelNum == 0 {
		return 0
	}

	return (region.StartMHz + bandwidthMHz/2) + (float32(channelNum)-1)*bandwidthMHz
}

func LoRaBandwidthMHz(settings NodeLoRaSettings) float32 {
	region := loRaRegionInfoByCode[settings.Region]
	if settings.UsePreset {
		presetBandwidth, ok := loRaBandwidthByPresetMHz[settings.ModemPreset]
		if !ok {
			return 0
		}
		if region.WideLoRa {
			return presetBandwidth * 3.25
		}

		return presetBandwidth
	}

	switch settings.Bandwidth {
	case 31:
		return 0.03125
	case 62:
		return 0.0625
	case 200:
		return 0.203125
	case 400:
		return 0.40625
	case 800:
		return 0.8125
	case 1600:
		return 1.625
	default:
		return float32(settings.Bandwidth) / 1000.0
	}
}

func loRaDJB2(name string) uint32 {
	hash := uint32(5381)
	for _, unit := range utf16.Encode([]rune(name)) {
		hash += (hash << 5) + uint32(unit)
	}

	return hash
}
