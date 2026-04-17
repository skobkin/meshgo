package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadDisplaySettings(ctx context.Context, target NodeSettingsTarget) (NodeDisplaySettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_DISPLAY_CONFIG, "get_config.display")
	if err != nil {
		return NodeDisplaySettings{}, err
	}
	display := cfg.GetDisplay()
	if display == nil {
		return NodeDisplaySettings{}, fmt.Errorf("display config payload is empty")
	}
	//nolint:staticcheck // Kept for Android parity while this proto field remains present upstream.
	compassNorthTop := display.GetCompassNorthTop()

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
	return s.saveConfig(ctx, target, "set_config.display", &generated.Config{
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
	})
}
