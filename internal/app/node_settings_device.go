package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadDeviceSettings(ctx context.Context, target NodeSettingsTarget) (NodeDeviceSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeDeviceSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeDeviceSettings{}, parseErr
	}
	s.logger.Info("requesting node device settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.device", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_DEVICE_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node device settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeDeviceSettings{}, err
		}
		s.logger.Warn(
			"requesting node device settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node device settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeDeviceSettings{}, fmt.Errorf("device config response is empty")
	}
	device := cfg.GetDevice()
	if device == nil {
		s.logger.Warn("requesting node device settings returned empty device payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeDeviceSettings{}, fmt.Errorf("device config payload is empty")
	}
	s.logger.Info("received node device settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeDeviceSettings{
		NodeID:                 strings.TrimSpace(target.NodeID),
		Role:                   int32(device.GetRole()),
		ButtonGPIO:             device.GetButtonGpio(),
		BuzzerGPIO:             device.GetBuzzerGpio(),
		RebroadcastMode:        int32(device.GetRebroadcastMode()),
		NodeInfoBroadcastSecs:  device.GetNodeInfoBroadcastSecs(),
		DoubleTapAsButtonPress: device.GetDoubleTapAsButtonPress(),
		DisableTripleClick:     device.GetDisableTripleClick(),
		Tzdef:                  strings.TrimSpace(device.GetTzdef()),
		LedHeartbeatDisabled:   device.GetLedHeartbeatDisabled(),
		BuzzerMode:             int32(device.GetBuzzerMode()),
	}, nil
}

func (s *NodeSettingsService) SaveDeviceSettings(ctx context.Context, target NodeSettingsTarget, settings NodeDeviceSettings) error {
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
	s.logger.Info("saving node device settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
				PayloadVariant: &generated.Config_Device{
					Device: &generated.Config_DeviceConfig{
						Role:                   generated.Config_DeviceConfig_Role(settings.Role),
						ButtonGpio:             settings.ButtonGPIO,
						BuzzerGpio:             settings.BuzzerGPIO,
						RebroadcastMode:        generated.Config_DeviceConfig_RebroadcastMode(settings.RebroadcastMode),
						NodeInfoBroadcastSecs:  settings.NodeInfoBroadcastSecs,
						DoubleTapAsButtonPress: settings.DoubleTapAsButtonPress,
						DisableTripleClick:     settings.DisableTripleClick,
						Tzdef:                  strings.TrimSpace(settings.Tzdef),
						LedHeartbeatDisabled:   settings.LedHeartbeatDisabled,
						BuzzerMode:             generated.Config_DeviceConfig_BuzzerMode(settings.BuzzerMode),
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.device", admin); err != nil {
		s.logger.Warn("set device config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set device config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node device settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
