package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadNeighborInfoSettings(ctx context.Context, target NodeSettingsTarget) (NodeNeighborInfoSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_NEIGHBORINFO_CONFIG, "get_module_config.neighbor_info")
	if err != nil {
		return NodeNeighborInfoSettings{}, err
	}
	neighbor := cfg.GetNeighborInfo()
	if neighbor == nil {
		return NodeNeighborInfoSettings{}, fmt.Errorf("neighbor info module config payload is empty")
	}

	return NodeNeighborInfoSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            neighbor.GetEnabled(),
		UpdateIntervalSecs: neighbor.GetUpdateInterval(),
		TransmitOverLoRa:   neighbor.GetTransmitOverLora(),
	}, nil
}

func (s *NodeSettingsService) SaveNeighborInfoSettings(ctx context.Context, target NodeSettingsTarget, settings NodeNeighborInfoSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.neighbor_info", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_NeighborInfo{
			NeighborInfo: &generated.ModuleConfig_NeighborInfoConfig{
				Enabled:          settings.Enabled,
				UpdateInterval:   settings.UpdateIntervalSecs,
				TransmitOverLora: settings.TransmitOverLoRa,
			},
		},
	})
}
