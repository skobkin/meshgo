package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadTelemetrySettings(ctx context.Context, target NodeSettingsTarget) (NodeTelemetrySettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_TELEMETRY_CONFIG, "get_module_config.telemetry")
	if err != nil {
		return NodeTelemetrySettings{}, err
	}
	telemetry := cfg.GetTelemetry()
	if telemetry == nil {
		return NodeTelemetrySettings{}, fmt.Errorf("telemetry module config payload is empty")
	}

	return NodeTelemetrySettings{
		NodeID:                        strings.TrimSpace(target.NodeID),
		DeviceUpdateInterval:          telemetry.GetDeviceUpdateInterval(),
		EnvironmentUpdateInterval:     telemetry.GetEnvironmentUpdateInterval(),
		EnvironmentMeasurementEnabled: telemetry.GetEnvironmentMeasurementEnabled(),
		EnvironmentScreenEnabled:      telemetry.GetEnvironmentScreenEnabled(),
		EnvironmentDisplayFahrenheit:  telemetry.GetEnvironmentDisplayFahrenheit(),
		AirQualityEnabled:             telemetry.GetAirQualityEnabled(),
		AirQualityInterval:            telemetry.GetAirQualityInterval(),
		PowerMeasurementEnabled:       telemetry.GetPowerMeasurementEnabled(),
		PowerUpdateInterval:           telemetry.GetPowerUpdateInterval(),
		PowerScreenEnabled:            telemetry.GetPowerScreenEnabled(),
		HealthMeasurementEnabled:      telemetry.GetHealthMeasurementEnabled(),
		HealthUpdateInterval:          telemetry.GetHealthUpdateInterval(),
		HealthScreenEnabled:           telemetry.GetHealthScreenEnabled(),
		DeviceTelemetryEnabled:        telemetry.GetDeviceTelemetryEnabled(),
		AirQualityScreenEnabled:       telemetry.GetAirQualityScreenEnabled(),
	}, nil
}

func (s *NodeSettingsService) SaveTelemetrySettings(ctx context.Context, target NodeSettingsTarget, settings NodeTelemetrySettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.telemetry", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Telemetry{
			Telemetry: &generated.ModuleConfig_TelemetryConfig{
				DeviceUpdateInterval:          settings.DeviceUpdateInterval,
				EnvironmentUpdateInterval:     settings.EnvironmentUpdateInterval,
				EnvironmentMeasurementEnabled: settings.EnvironmentMeasurementEnabled,
				EnvironmentScreenEnabled:      settings.EnvironmentScreenEnabled,
				EnvironmentDisplayFahrenheit:  settings.EnvironmentDisplayFahrenheit,
				AirQualityEnabled:             settings.AirQualityEnabled,
				AirQualityInterval:            settings.AirQualityInterval,
				PowerMeasurementEnabled:       settings.PowerMeasurementEnabled,
				PowerUpdateInterval:           settings.PowerUpdateInterval,
				PowerScreenEnabled:            settings.PowerScreenEnabled,
				HealthMeasurementEnabled:      settings.HealthMeasurementEnabled,
				HealthUpdateInterval:          settings.HealthUpdateInterval,
				HealthScreenEnabled:           settings.HealthScreenEnabled,
				DeviceTelemetryEnabled:        settings.DeviceTelemetryEnabled,
				AirQualityScreenEnabled:       settings.AirQualityScreenEnabled,
			},
		},
	})
}
