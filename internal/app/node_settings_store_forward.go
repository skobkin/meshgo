package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadStoreForwardSettings(ctx context.Context, target NodeSettingsTarget) (NodeStoreForwardSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_STOREFORWARD_CONFIG, "get_module_config.store_forward")
	if err != nil {
		return NodeStoreForwardSettings{}, err
	}
	storeForward := cfg.GetStoreForward()
	if storeForward == nil {
		return NodeStoreForwardSettings{}, fmt.Errorf("store forward module config payload is empty")
	}

	return NodeStoreForwardSettings{
		NodeID:              strings.TrimSpace(target.NodeID),
		Enabled:             storeForward.GetEnabled(),
		Heartbeat:           storeForward.GetHeartbeat(),
		Records:             storeForward.GetRecords(),
		HistoryReturnMax:    storeForward.GetHistoryReturnMax(),
		HistoryReturnWindow: storeForward.GetHistoryReturnWindow(),
		IsServer:            storeForward.GetIsServer(),
	}, nil
}

func (s *NodeSettingsService) SaveStoreForwardSettings(ctx context.Context, target NodeSettingsTarget, settings NodeStoreForwardSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.store_forward", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_StoreForward{
			StoreForward: &generated.ModuleConfig_StoreForwardConfig{
				Enabled:             settings.Enabled,
				Heartbeat:           settings.Heartbeat,
				Records:             settings.Records,
				HistoryReturnMax:    settings.HistoryReturnMax,
				HistoryReturnWindow: settings.HistoryReturnWindow,
				IsServer:            settings.IsServer,
			},
		},
	})
}
