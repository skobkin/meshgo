package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadRangeTestSettings(ctx context.Context, target NodeSettingsTarget) (NodeRangeTestSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_RANGETEST_CONFIG, "get_module_config.range_test")
	if err != nil {
		return NodeRangeTestSettings{}, err
	}
	rangeTest := cfg.GetRangeTest()
	if rangeTest == nil {
		return NodeRangeTestSettings{}, fmt.Errorf("range test module config payload is empty")
	}

	return NodeRangeTestSettings{
		NodeID:        strings.TrimSpace(target.NodeID),
		Enabled:       rangeTest.GetEnabled(),
		Sender:        rangeTest.GetSender(),
		Save:          rangeTest.GetSave(),
		ClearOnReboot: rangeTest.GetClearOnReboot(),
	}, nil
}

func (s *NodeSettingsService) SaveRangeTestSettings(ctx context.Context, target NodeSettingsTarget, settings NodeRangeTestSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.range_test", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_RangeTest{
			RangeTest: &generated.ModuleConfig_RangeTestConfig{
				Enabled:       settings.Enabled,
				Sender:        settings.Sender,
				Save:          settings.Save,
				ClearOnReboot: settings.ClearOnReboot,
			},
		},
	})
}
