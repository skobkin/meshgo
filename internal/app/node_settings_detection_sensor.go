package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadDetectionSensorSettings(ctx context.Context, target NodeSettingsTarget) (NodeDetectionSensorSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_DETECTIONSENSOR_CONFIG, "get_module_config.detection_sensor")
	if err != nil {
		return NodeDetectionSensorSettings{}, err
	}
	detection := cfg.GetDetectionSensor()
	if detection == nil {
		return NodeDetectionSensorSettings{}, fmt.Errorf("detection sensor module config payload is empty")
	}

	return NodeDetectionSensorSettings{
		NodeID:               strings.TrimSpace(target.NodeID),
		Enabled:              detection.GetEnabled(),
		MinimumBroadcastSecs: detection.GetMinimumBroadcastSecs(),
		StateBroadcastSecs:   detection.GetStateBroadcastSecs(),
		SendBell:             detection.GetSendBell(),
		Name:                 detection.GetName(),
		MonitorPin:           detection.GetMonitorPin(),
		DetectionTriggerType: int32(detection.GetDetectionTriggerType()),
		UsePullup:            detection.GetUsePullup(),
	}, nil
}

func (s *NodeSettingsService) SaveDetectionSensorSettings(ctx context.Context, target NodeSettingsTarget, settings NodeDetectionSensorSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.detection_sensor", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_DetectionSensor{
			DetectionSensor: &generated.ModuleConfig_DetectionSensorConfig{
				Enabled:              settings.Enabled,
				MinimumBroadcastSecs: settings.MinimumBroadcastSecs,
				StateBroadcastSecs:   settings.StateBroadcastSecs,
				SendBell:             settings.SendBell,
				Name:                 settings.Name,
				MonitorPin:           settings.MonitorPin,
				DetectionTriggerType: generated.ModuleConfig_DetectionSensorConfig_TriggerType(settings.DetectionTriggerType),
				UsePullup:            settings.UsePullup,
			},
		},
	})
}
