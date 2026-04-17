package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadDeviceSettings(ctx context.Context, target NodeSettingsTarget) (NodeDeviceSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_DEVICE_CONFIG, "get_config.device")
	if err != nil {
		return NodeDeviceSettings{}, err
	}
	device := cfg.GetDevice()
	if device == nil {
		return NodeDeviceSettings{}, fmt.Errorf("device config payload is empty")
	}

	return NodeDeviceSettings{
		NodeID:                 strings.TrimSpace(target.NodeID),
		Role:                   int32(device.GetRole()),
		ButtonGPIO:             device.GetButtonGpio(),
		BuzzerGPIO:             device.GetBuzzerGpio(),
		RebroadcastMode:        int32(device.GetRebroadcastMode()),
		NodeInfoBroadcastSecs:  device.GetNodeInfoBroadcastSecs(),
		DoubleTapAsButtonPress: device.GetDoubleTapAsButtonPress(),
		DisableTripleClick:     device.GetDisableTripleClick(),
		Tzdef:                  strings.TrimSpace(device.GetTzdef()),
		LedHeartbeatDisabled:   device.GetLedHeartbeatDisabled(),
		BuzzerMode:             int32(device.GetBuzzerMode()),
	}, nil
}

func (s *NodeSettingsService) SaveDeviceSettings(ctx context.Context, target NodeSettingsTarget, settings NodeDeviceSettings) error {
	return s.saveConfig(ctx, target, "set_config.device", &generated.Config{
		PayloadVariant: &generated.Config_Device{
			Device: &generated.Config_DeviceConfig{
				Role:                   generated.Config_DeviceConfig_Role(settings.Role),
				ButtonGpio:             settings.ButtonGPIO,
				BuzzerGpio:             settings.BuzzerGPIO,
				RebroadcastMode:        generated.Config_DeviceConfig_RebroadcastMode(settings.RebroadcastMode),
				NodeInfoBroadcastSecs:  settings.NodeInfoBroadcastSecs,
				DoubleTapAsButtonPress: settings.DoubleTapAsButtonPress,
				DisableTripleClick:     settings.DisableTripleClick,
				Tzdef:                  strings.TrimSpace(settings.Tzdef),
				LedHeartbeatDisabled:   settings.LedHeartbeatDisabled,
				BuzzerMode:             generated.Config_DeviceConfig_BuzzerMode(settings.BuzzerMode),
			},
		},
	})
}
