package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadBluetoothSettings(ctx context.Context, target NodeSettingsTarget) (NodeBluetoothSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeBluetoothSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeBluetoothSettings{}, parseErr
	}
	s.logger.Info("requesting node bluetooth settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.bluetooth", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_BLUETOOTH_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node bluetooth settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeBluetoothSettings{}, err
		}
		s.logger.Warn(
			"requesting node bluetooth settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node bluetooth settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeBluetoothSettings{}, fmt.Errorf("bluetooth config response is empty")
	}
	bluetooth := cfg.GetBluetooth()
	if bluetooth == nil {
		s.logger.Warn("requesting node bluetooth settings returned empty bluetooth payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeBluetoothSettings{}, fmt.Errorf("bluetooth config payload is empty")
	}
	s.logger.Info("received node bluetooth settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeBluetoothSettings{
		NodeID:   strings.TrimSpace(target.NodeID),
		Enabled:  bluetooth.GetEnabled(),
		Mode:     int32(bluetooth.GetMode()),
		FixedPIN: bluetooth.GetFixedPin(),
	}, nil
}

func (s *NodeSettingsService) SaveBluetoothSettings(ctx context.Context, target NodeSettingsTarget, settings NodeBluetoothSettings) error {
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
	s.logger.Info("saving node bluetooth settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
				PayloadVariant: &generated.Config_Bluetooth{
					Bluetooth: &generated.Config_BluetoothConfig{
						Enabled:  settings.Enabled,
						Mode:     generated.Config_BluetoothConfig_PairingMode(settings.Mode),
						FixedPin: settings.FixedPIN,
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.bluetooth", admin); err != nil {
		s.logger.Warn("set bluetooth config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set bluetooth config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node bluetooth settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
