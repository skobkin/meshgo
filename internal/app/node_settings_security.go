package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadSecuritySettings(ctx context.Context, target NodeSettingsTarget) (NodeSecuritySettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeSecuritySettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeSecuritySettings{}, parseErr
	}
	s.logger.Info("requesting node security settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.security", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_SECURITY_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node security settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeSecuritySettings{}, err
		}
		s.logger.Warn(
			"requesting node security settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node security settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeSecuritySettings{}, fmt.Errorf("security config response is empty")
	}
	security := cfg.GetSecurity()
	if security == nil {
		s.logger.Warn("requesting node security settings returned empty security payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeSecuritySettings{}, fmt.Errorf("security config payload is empty")
	}
	s.logger.Info("received node security settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeSecuritySettings{
		NodeID:              strings.TrimSpace(target.NodeID),
		PublicKey:           cloneBytes(security.GetPublicKey()),
		PrivateKey:          cloneBytes(security.GetPrivateKey()),
		AdminKeys:           cloneBytesList(security.GetAdminKey()),
		IsManaged:           security.GetIsManaged(),
		SerialEnabled:       security.GetSerialEnabled(),
		DebugLogAPIEnabled:  security.GetDebugLogApiEnabled(),
		AdminChannelEnabled: security.GetAdminChannelEnabled(),
	}, nil
}

func (s *NodeSettingsService) SaveSecuritySettings(ctx context.Context, target NodeSettingsTarget, settings NodeSecuritySettings) error {
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
	s.logger.Info("saving node security settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
				PayloadVariant: &generated.Config_Security{
					Security: &generated.Config_SecurityConfig{
						PublicKey:           cloneBytes(settings.PublicKey),
						PrivateKey:          cloneBytes(settings.PrivateKey),
						AdminKey:            cloneBytesList(settings.AdminKeys),
						IsManaged:           settings.IsManaged,
						SerialEnabled:       settings.SerialEnabled,
						DebugLogApiEnabled:  settings.DebugLogAPIEnabled,
						AdminChannelEnabled: settings.AdminChannelEnabled,
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.security", admin); err != nil {
		s.logger.Warn("set security config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set security config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node security settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
