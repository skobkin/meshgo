package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadAmbientLightingSettings(ctx context.Context, target NodeSettingsTarget) (NodeAmbientLightingSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_AMBIENTLIGHTING_CONFIG, "get_module_config.ambient_lighting")
	if err != nil {
		return NodeAmbientLightingSettings{}, err
	}
	lighting := cfg.GetAmbientLighting()
	if lighting == nil {
		return NodeAmbientLightingSettings{}, fmt.Errorf("ambient lighting module config payload is empty")
	}

	return NodeAmbientLightingSettings{
		NodeID:   strings.TrimSpace(target.NodeID),
		LEDState: lighting.GetLedState(),
		Current:  lighting.GetCurrent(),
		Red:      lighting.GetRed(),
		Green:    lighting.GetGreen(),
		Blue:     lighting.GetBlue(),
	}, nil
}

func (s *NodeSettingsService) SaveAmbientLightingSettings(ctx context.Context, target NodeSettingsTarget, settings NodeAmbientLightingSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.ambient_lighting", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_AmbientLighting{
			AmbientLighting: &generated.ModuleConfig_AmbientLightingConfig{
				LedState: settings.LEDState,
				Current:  settings.Current,
				Red:      settings.Red,
				Green:    settings.Green,
				Blue:     settings.Blue,
			},
		},
	})
}
