package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadStatusMessageSettings(ctx context.Context, target NodeSettingsTarget) (NodeStatusMessageSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STATUSMESSAGE_CONFIG, "get_module_config.status_message")
	if err != nil {
		return NodeStatusMessageSettings{}, err
	}
	status := cfg.GetStatusmessage()
	if status == nil {
		return NodeStatusMessageSettings{}, fmt.Errorf("status message module config payload is empty")
	}

	return NodeStatusMessageSettings{
		NodeID:     strings.TrimSpace(target.NodeID),
		NodeStatus: status.GetNodeStatus(),
	}, nil
}

func (s *NodeSettingsService) SaveStatusMessageSettings(ctx context.Context, target NodeSettingsTarget, settings NodeStatusMessageSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.status_message", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Statusmessage{
			Statusmessage: &generated.ModuleConfig_StatusMessageConfig{
				NodeStatus: settings.NodeStatus,
			},
		},
	})
}
