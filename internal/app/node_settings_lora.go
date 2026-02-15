package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadLoRaSettings(ctx context.Context, target NodeSettingsTarget) (NodeLoRaSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeLoRaSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeLoRaSettings{}, parseErr
	}
	s.logger.Info("requesting node LoRa settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.lora", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_LORA_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node LoRa settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeLoRaSettings{}, err
		}
		s.logger.Warn(
			"requesting node LoRa settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node LoRa settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeLoRaSettings{}, fmt.Errorf("LoRa config response is empty")
	}
	lora := cfg.GetLora()
	if lora == nil {
		s.logger.Warn("requesting node LoRa settings returned empty LoRa payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeLoRaSettings{}, fmt.Errorf("LoRa config payload is empty")
	}
	s.logger.Info("received node LoRa settings response", "node_id", strings.TrimSpace(target.NodeID))

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
	if s == nil || s.bus == nil || s.radio == nil {
		return fmt.Errorf("node settings service is not initialized")
	}
	if !s.isConnected() {
		return fmt.Errorf("device is not connected")
	}

	nodeNum, err := parseNodeID(target.NodeID)
	if err != nil {
		return err
	}
	s.logger.Info("saving node LoRa settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	release, err := s.beginSave()
	if err != nil {
		return err
	}
	defer release()

	saveCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "begin_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_BeginEditSettings{BeginEditSettings: true},
	}); err != nil {
		s.logger.Warn("begin edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("begin edit settings: %w", err)
	}

	admin := &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_SetConfig{
			SetConfig: &generated.Config{
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
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.lora", admin); err != nil {
		s.logger.Warn("set LoRa config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set LoRa config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node LoRa settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}

func cloneUint32Slice(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}

	out := make([]uint32, len(values))
	copy(out, values)

	return out
}
