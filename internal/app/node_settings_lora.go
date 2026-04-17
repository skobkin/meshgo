package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadLoRaSettings(ctx context.Context, target NodeSettingsTarget) (NodeLoRaSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_LORA_CONFIG, "get_config.lora")
	if err != nil {
		return NodeLoRaSettings{}, err
	}
	lora := cfg.GetLora()
	if lora == nil {
		return NodeLoRaSettings{}, fmt.Errorf("LoRa config payload is empty")
	}

	return NodeLoRaSettings{
		NodeID:              strings.TrimSpace(target.NodeID),
		UsePreset:           lora.GetUsePreset(),
		ModemPreset:         int32(lora.GetModemPreset()),
		Bandwidth:           lora.GetBandwidth(),
		SpreadFactor:        lora.GetSpreadFactor(),
		CodingRate:          lora.GetCodingRate(),
		FrequencyOffset:     lora.GetFrequencyOffset(),
		Region:              int32(lora.GetRegion()),
		HopLimit:            lora.GetHopLimit(),
		TxEnabled:           lora.GetTxEnabled(),
		TxPower:             lora.GetTxPower(),
		ChannelNum:          lora.GetChannelNum(),
		OverrideDutyCycle:   lora.GetOverrideDutyCycle(),
		Sx126XRxBoostedGain: lora.GetSx126XRxBoostedGain(),
		OverrideFrequency:   lora.GetOverrideFrequency(),
		PaFanDisabled:       lora.GetPaFanDisabled(),
		IgnoreIncoming:      cloneUint32Slice(lora.GetIgnoreIncoming()),
		IgnoreMqtt:          lora.GetIgnoreMqtt(),
		ConfigOkToMqtt:      lora.GetConfigOkToMqtt(),
	}, nil
}

func (s *NodeSettingsService) SaveLoRaSettings(ctx context.Context, target NodeSettingsTarget, settings NodeLoRaSettings) error {
	return s.saveConfig(ctx, target, "set_config.lora", &generated.Config{
		PayloadVariant: &generated.Config_Lora{
			Lora: &generated.Config_LoRaConfig{
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
			},
		},
	})
}
