package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadDisplaySettings(ctx context.Context, target NodeSettingsTarget) (NodeDisplaySettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeDisplaySettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeDisplaySettings{}, parseErr
	}
	s.logger.Info("requesting node display settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_config.display", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: generated.AdminMessage_DISPLAY_CONFIG},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			s.logger.Warn("requesting node display settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeDisplaySettings{}, err
		}
		s.logger.Warn(
			"requesting node display settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		s.logger.Warn("requesting node display settings returned empty config response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeDisplaySettings{}, fmt.Errorf("display config response is empty")
	}
	display := cfg.GetDisplay()
	if display == nil {
		s.logger.Warn("requesting node display settings returned empty display payload", "node_id", strings.TrimSpace(target.NodeID))

		return NodeDisplaySettings{}, fmt.Errorf("display config payload is empty")
	}
	//nolint:staticcheck // Kept for Android parity while this proto field remains present upstream.
	compassNorthTop := display.GetCompassNorthTop()
	s.logger.Info("received node display settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeDisplaySettings{
		NodeID:                 strings.TrimSpace(target.NodeID),
		ScreenOnSecs:           display.GetScreenOnSecs(),
		AutoScreenCarouselSecs: display.GetAutoScreenCarouselSecs(),
		CompassNorthTop:        compassNorthTop,
		FlipScreen:             display.GetFlipScreen(),
		Units:                  int32(display.GetUnits()),
		Oled:                   int32(display.GetOled()),
		DisplayMode:            int32(display.GetDisplaymode()),
		HeadingBold:            display.GetHeadingBold(),
		WakeOnTapOrMotion:      display.GetWakeOnTapOrMotion(),
		CompassOrientation:     int32(display.GetCompassOrientation()),
		Use12HClock:            display.GetUse_12HClock(),
	}, nil
}

func (s *NodeSettingsService) SaveDisplaySettings(ctx context.Context, target NodeSettingsTarget, settings NodeDisplaySettings) error {
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
	s.logger.Info("saving node display settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
		PayloadVariant: &generated.AdminMessage_SetConfig{
			SetConfig: &generated.Config{
				PayloadVariant: &generated.Config_Display{
					Display: &generated.Config_DisplayConfig{
						ScreenOnSecs:           settings.ScreenOnSecs,
						AutoScreenCarouselSecs: settings.AutoScreenCarouselSecs,
						CompassNorthTop:        settings.CompassNorthTop,
						FlipScreen:             settings.FlipScreen,
						Units:                  generated.Config_DisplayConfig_DisplayUnits(settings.Units),
						Oled:                   generated.Config_DisplayConfig_OledType(settings.Oled),
						Displaymode:            generated.Config_DisplayConfig_DisplayMode(settings.DisplayMode),
						HeadingBold:            settings.HeadingBold,
						WakeOnTapOrMotion:      settings.WakeOnTapOrMotion,
						CompassOrientation:     generated.Config_DisplayConfig_CompassOrientation(settings.CompassOrientation),
						Use_12HClock:           settings.Use12HClock,
					},
				},
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_config.display", admin); err != nil {
		s.logger.Warn("set display config failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set display config: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node display settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
