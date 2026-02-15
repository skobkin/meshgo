package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadRangeTestSettings(ctx context.Context, target NodeSettingsTarget) (NodeRangeTestSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeRangeTestSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeRangeTestSettings{}, parseErr
	}
	s.logger.Info("requesting node range test settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_module_config.range_test", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetModuleConfigRequest{
				GetModuleConfigRequest: generated.AdminMessage_RANGETEST_CONFIG,
			},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node range test settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeRangeTestSettings{}, err
		}
		s.logger.Warn(
			"requesting node range test settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetModuleConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node range test settings returned empty module config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeRangeTestSettings{}, fmt.Errorf("range test module config response is empty")
	}
	rangeTest := cfg.GetRangeTest()
	if rangeTest == nil {
		s.logger.Warn("requesting node range test settings returned empty range test payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeRangeTestSettings{}, fmt.Errorf("range test module config payload is empty")
	}
	s.logger.Info("received node range test settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeRangeTestSettings{
		NodeID:        strings.TrimSpace(target.NodeID),
		Enabled:       rangeTest.GetEnabled(),
		Sender:        rangeTest.GetSender(),
		Save:          rangeTest.GetSave(),
		ClearOnReboot: rangeTest.GetClearOnReboot(),
	}, nil
}

func (s *NodeSettingsService) SaveRangeTestSettings(ctx context.Context, target NodeSettingsTarget, settings NodeRangeTestSettings) error {
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
	s.logger.Info("saving node range test settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
		PayloadVariant: &generated.AdminMessage_SetModuleConfig{
			SetModuleConfig: &generated.ModuleConfig{
				PayloadVariant: &generated.ModuleConfig_RangeTest{
					RangeTest: &generated.ModuleConfig_RangeTestConfig{
						Enabled:       settings.Enabled,
						Sender:        settings.Sender,
						Save:          settings.Save,
						ClearOnReboot: settings.ClearOnReboot,
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.range_test", admin); err != nil {
		s.logger.Warn("set range test module config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set range test module config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node range test settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
