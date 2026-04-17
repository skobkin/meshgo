package app

import (
	"context"
	"fmt"
	"math"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadPositionSettings(ctx context.Context, target NodeSettingsTarget) (NodePositionSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_POSITION_CONFIG, "get_config.position")
	if err != nil {
		return NodePositionSettings{}, err
	}
	position := cfg.GetPosition()
	if position == nil {
		return NodePositionSettings{}, fmt.Errorf("position config payload is empty")
	}

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
	return s.runEditSettingsWrite(ctx, target, "set_config.position", func(saveCtx context.Context, nodeNum uint32) error {
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
				return fmt.Errorf("set fixed position: %w", err)
			}
		} else if settings.RemoveFixedPosition {
			if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "remove_fixed_position", &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_RemoveFixedPosition{RemoveFixedPosition: true},
			}); err != nil {
				return fmt.Errorf("remove fixed position: %w", err)
			}
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.position", &generated.AdminMessage{
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
		}); err != nil {
			return fmt.Errorf("set position config: %w", err)
		}

		return nil
	})
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
