package app

import (
	"context"
	"fmt"
	"math"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadPositionSettings(ctx context.Context, target NodeSettingsTarget) (NodePositionSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodePositionSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodePositionSettings{}, parseErr
	}
	s.logger.Info("requesting node position settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.position", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_POSITION_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node position settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodePositionSettings{}, err
		}
		s.logger.Warn(
			"requesting node position settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node position settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodePositionSettings{}, fmt.Errorf("position config response is empty")
	}
	position := cfg.GetPosition()
	if position == nil {
		s.logger.Warn("requesting node position settings returned empty position payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodePositionSettings{}, fmt.Errorf("position config payload is empty")
	}
	s.logger.Info("received node position settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodePositionSettings{
		NodeID:                            strings.TrimSpace(target.NodeID),
		PositionBroadcastSecs:             position.GetPositionBroadcastSecs(),
		PositionBroadcastSmartEnabled:     position.GetPositionBroadcastSmartEnabled(),
		FixedPosition:                     position.GetFixedPosition(),
		GpsUpdateInterval:                 position.GetGpsUpdateInterval(),
		PositionFlags:                     position.GetPositionFlags(),
		RxGPIO:                            position.GetRxGpio(),
		TxGPIO:                            position.GetTxGpio(),
		BroadcastSmartMinimumDistance:     position.GetBroadcastSmartMinimumDistance(),
		BroadcastSmartMinimumIntervalSecs: position.GetBroadcastSmartMinimumIntervalSecs(),
		GpsEnGPIO:                         position.GetGpsEnGpio(),
		GpsMode:                           int32(position.GetGpsMode()),
	}, nil
}

func (s *NodeSettingsService) SavePositionSettings(ctx context.Context, target NodeSettingsTarget, settings NodePositionSettings) error {
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
	s.logger.Info("saving node position settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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

	if settings.FixedPosition {
		if settings.FixedLatitude == nil || settings.FixedLongitude == nil {
			return fmt.Errorf("fixed position requires latitude and longitude")
		}
		latI, lonI, err := encodeFixedPositionCoordinates(*settings.FixedLatitude, *settings.FixedLongitude)
		if err != nil {
			return err
		}
		altitude := int32(0)
		if settings.FixedAltitude != nil {
			altitude = *settings.FixedAltitude
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_fixed_position", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetFixedPosition{
				SetFixedPosition: &generated.Position{
					LatitudeI:  &latI,
					LongitudeI: &lonI,
					Altitude:   &altitude,
				},
			},
		}); err != nil {
			s.logger.Warn("set fixed position failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return fmt.Errorf("set fixed position: %w", err)
		}
	} else if settings.RemoveFixedPosition {
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "remove_fixed_position", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_RemoveFixedPosition{RemoveFixedPosition: true},
		}); err != nil {
			s.logger.Warn("remove fixed position failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return fmt.Errorf("remove fixed position: %w", err)
		}
	}

	admin := &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_SetConfig{
			SetConfig: &generated.Config{
				PayloadVariant: &generated.Config_Position{
					Position: &generated.Config_PositionConfig{
						PositionBroadcastSecs:             settings.PositionBroadcastSecs,
						PositionBroadcastSmartEnabled:     settings.PositionBroadcastSmartEnabled,
						FixedPosition:                     settings.FixedPosition,
						GpsUpdateInterval:                 settings.GpsUpdateInterval,
						PositionFlags:                     settings.PositionFlags,
						RxGpio:                            settings.RxGPIO,
						TxGpio:                            settings.TxGPIO,
						BroadcastSmartMinimumDistance:     settings.BroadcastSmartMinimumDistance,
						BroadcastSmartMinimumIntervalSecs: settings.BroadcastSmartMinimumIntervalSecs,
						GpsEnGpio:                         settings.GpsEnGPIO,
						GpsMode:                           generated.Config_PositionConfig_GpsMode(settings.GpsMode),
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.position", admin); err != nil {
		s.logger.Warn("set position config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set position config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node position settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}

func encodeFixedPositionCoordinates(latitude, longitude float64) (int32, int32, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return 0, 0, fmt.Errorf("fixed position latitude must be between -90 and 90")
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return 0, 0, fmt.Errorf("fixed position longitude must be between -180 and 180")
	}

	lat := int32(math.Round(latitude * 1e7))
	lon := int32(math.Round(longitude * 1e7))

	return lat, lon, nil
}
