package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadMQTTSettings(ctx context.Context, target NodeSettingsTarget) (NodeMQTTSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeMQTTSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeMQTTSettings{}, parseErr
	}
	s.logger.Info("requesting node MQTT settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_module_config.mqtt", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetModuleConfigRequest{
				GetModuleConfigRequest: generated.AdminMessage_MQTT_CONFIG,
			},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node MQTT settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeMQTTSettings{}, err
		}
		s.logger.Warn(
			"requesting node MQTT settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetModuleConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node MQTT settings returned empty module config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeMQTTSettings{}, fmt.Errorf("MQTT module config response is empty")
	}
	mqtt := cfg.GetMqtt()
	if mqtt == nil {
		s.logger.Warn("requesting node MQTT settings returned empty MQTT payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeMQTTSettings{}, fmt.Errorf("MQTT module config payload is empty")
	}
	s.logger.Info("received node MQTT settings response", "node_id", strings.TrimSpace(target.NodeID))

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
	s.logger.Info("saving node MQTT settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
		PayloadVariant: &generated.AdminMessage_SetModuleConfig{
			SetModuleConfig: &generated.ModuleConfig{
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
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.mqtt", admin); err != nil {
		s.logger.Warn("set MQTT module config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set MQTT module config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node MQTT settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
