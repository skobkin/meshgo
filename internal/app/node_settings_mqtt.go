package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadMQTTSettings(ctx context.Context, target NodeSettingsTarget) (NodeMQTTSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_MQTT_CONFIG, "get_module_config.mqtt")
	if err != nil {
		return NodeMQTTSettings{}, err
	}
	mqtt := cfg.GetMqtt()
	if mqtt == nil {
		return NodeMQTTSettings{}, fmt.Errorf("MQTT module config payload is empty")
	}

	settings := NodeMQTTSettings{
		NodeID:               strings.TrimSpace(target.NodeID),
		Enabled:              mqtt.GetEnabled(),
		Address:              mqtt.GetAddress(),
		Username:             mqtt.GetUsername(),
		Password:             mqtt.GetPassword(),
		EncryptionEnabled:    mqtt.GetEncryptionEnabled(),
		JSONEnabled:          mqtt.GetJsonEnabled(),
		TLSEnabled:           mqtt.GetTlsEnabled(),
		Root:                 mqtt.GetRoot(),
		ProxyToClientEnabled: mqtt.GetProxyToClientEnabled(),
		MapReportingEnabled:  mqtt.GetMapReportingEnabled(),
	}
	if mapReport := mqtt.GetMapReportSettings(); mapReport != nil {
		settings.MapReportPublishIntervalSecs = mapReport.GetPublishIntervalSecs()
		settings.MapReportPositionPrecision = mapReport.GetPositionPrecision()
		settings.MapReportShouldReportLocation = mapReport.GetShouldReportLocation()
	}

	return settings, nil
}

func (s *NodeSettingsService) SaveMQTTSettings(ctx context.Context, target NodeSettingsTarget, settings NodeMQTTSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.mqtt", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Mqtt{
			Mqtt: &generated.ModuleConfig_MQTTConfig{
				Enabled:              settings.Enabled,
				Address:              settings.Address,
				Username:             settings.Username,
				Password:             settings.Password,
				EncryptionEnabled:    settings.EncryptionEnabled,
				JsonEnabled:          settings.JSONEnabled,
				TlsEnabled:           settings.TLSEnabled,
				Root:                 settings.Root,
				ProxyToClientEnabled: settings.ProxyToClientEnabled,
				MapReportingEnabled:  settings.MapReportingEnabled,
				MapReportSettings: &generated.ModuleConfig_MapReportSettings{
					PublishIntervalSecs:  settings.MapReportPublishIntervalSecs,
					PositionPrecision:    settings.MapReportPositionPrecision,
					ShouldReportLocation: settings.MapReportShouldReportLocation,
				},
			},
		},
	})
}
