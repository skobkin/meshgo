package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadPaxcounterSettings(ctx context.Context, target NodeSettingsTarget) (NodePaxcounterSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_PAXCOUNTER_CONFIG, "get_module_config.paxcounter")
	if err != nil {
		return NodePaxcounterSettings{}, err
	}
	paxcounter := cfg.GetPaxcounter()
	if paxcounter == nil {
		return NodePaxcounterSettings{}, fmt.Errorf("paxcounter module config payload is empty")
	}

	return NodePaxcounterSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            paxcounter.GetEnabled(),
		UpdateIntervalSecs: paxcounter.GetPaxcounterUpdateInterval(),
		WifiThreshold:      paxcounter.GetWifiThreshold(),
		BLEThreshold:       paxcounter.GetBleThreshold(),
	}, nil
}

func (s *NodeSettingsService) SavePaxcounterSettings(ctx context.Context, target NodeSettingsTarget, settings NodePaxcounterSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.paxcounter", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Paxcounter{
			Paxcounter: &generated.ModuleConfig_PaxcounterConfig{
				Enabled:                  settings.Enabled,
				PaxcounterUpdateInterval: settings.UpdateIntervalSecs,
				WifiThreshold:            settings.WifiThreshold,
				BleThreshold:             settings.BLEThreshold,
			},
		},
	})
}
