package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadPowerSettings(ctx context.Context, target NodeSettingsTarget) (NodePowerSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_POWER_CONFIG, "get_config.power")
	if err != nil {
		return NodePowerSettings{}, err
	}
	power := cfg.GetPower()
	if power == nil {
		return NodePowerSettings{}, fmt.Errorf("power config payload is empty")
	}

	return NodePowerSettings{
		NodeID:                     strings.TrimSpace(target.NodeID),
		IsPowerSaving:              power.GetIsPowerSaving(),
		OnBatteryShutdownAfterSecs: power.GetOnBatteryShutdownAfterSecs(),
		AdcMultiplierOverride:      power.GetAdcMultiplierOverride(),
		WaitBluetoothSecs:          power.GetWaitBluetoothSecs(),
		SdsSecs:                    power.GetSdsSecs(),
		LsSecs:                     power.GetLsSecs(),
		MinWakeSecs:                power.GetMinWakeSecs(),
		DeviceBatteryInaAddress:    power.GetDeviceBatteryInaAddress(),
		PowermonEnables:            power.GetPowermonEnables(),
	}, nil
}

func (s *NodeSettingsService) SavePowerSettings(ctx context.Context, target NodeSettingsTarget, settings NodePowerSettings) error {
	return s.saveConfig(ctx, target, "set_config.power", &generated.Config{
		PayloadVariant: &generated.Config_Power{
			Power: &generated.Config_PowerConfig{
				IsPowerSaving:              settings.IsPowerSaving,
				OnBatteryShutdownAfterSecs: settings.OnBatteryShutdownAfterSecs,
				AdcMultiplierOverride:      settings.AdcMultiplierOverride,
				WaitBluetoothSecs:          settings.WaitBluetoothSecs,
				SdsSecs:                    settings.SdsSecs,
				LsSecs:                     settings.LsSecs,
				MinWakeSecs:                settings.MinWakeSecs,
				DeviceBatteryInaAddress:    settings.DeviceBatteryInaAddress,
				PowermonEnables:            settings.PowermonEnables,
			},
		},
	})
}
