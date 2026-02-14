package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadPowerSettings(ctx context.Context, target NodeSettingsTarget) (NodePowerSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodePowerSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodePowerSettings{}, parseErr
	}
	s.logger.Info("requesting node power settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.power", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_POWER_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node power settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodePowerSettings{}, err
		}
		s.logger.Warn(
			"requesting node power settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node power settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodePowerSettings{}, fmt.Errorf("power config response is empty")
	}
	power := cfg.GetPower()
	if power == nil {
		s.logger.Warn("requesting node power settings returned empty power payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodePowerSettings{}, fmt.Errorf("power config payload is empty")
	}
	s.logger.Info("received node power settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodePowerSettings{
		NodeID:                     strings.TrimSpace(target.NodeID),
		IsPowerSaving:              power.GetIsPowerSaving(),
		OnBatteryShutdownAfterSecs: power.GetOnBatteryShutdownAfterSecs(),
		AdcMultiplierOverride:      power.GetAdcMultiplierOverride(),
		WaitBluetoothSecs:          power.GetWaitBluetoothSecs(),
		SdsSecs:                    power.GetSdsSecs(),
		LsSecs:                     power.GetLsSecs(),
		MinWakeSecs:                power.GetMinWakeSecs(),
		DeviceBatteryInaAddress:    power.GetDeviceBatteryInaAddress(),
		PowermonEnables:            power.GetPowermonEnables(),
	}, nil
}

func (s *NodeSettingsService) SavePowerSettings(ctx context.Context, target NodeSettingsTarget, settings NodePowerSettings) error {
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
	s.logger.Info("saving node power settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
				PayloadVariant: &generated.Config_Power{
					Power: &generated.Config_PowerConfig{
						IsPowerSaving:              settings.IsPowerSaving,
						OnBatteryShutdownAfterSecs: settings.OnBatteryShutdownAfterSecs,
						AdcMultiplierOverride:      settings.AdcMultiplierOverride,
						WaitBluetoothSecs:          settings.WaitBluetoothSecs,
						SdsSecs:                    settings.SdsSecs,
						LsSecs:                     settings.LsSecs,
						MinWakeSecs:                settings.MinWakeSecs,
						DeviceBatteryInaAddress:    settings.DeviceBatteryInaAddress,
						PowermonEnables:            settings.PowermonEnables,
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.power", admin); err != nil {
		s.logger.Warn("set power config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set power config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node power settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
