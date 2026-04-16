package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadRemoteHardwareSettings(ctx context.Context, target NodeSettingsTarget) (NodeRemoteHardwareSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_REMOTEHARDWARE_CONFIG, "get_module_config.remote_hardware")
	if err != nil {
		return NodeRemoteHardwareSettings{}, err
	}
	hardware := cfg.GetRemoteHardware()
	if hardware == nil {
		return NodeRemoteHardwareSettings{}, fmt.Errorf("remote hardware module config payload is empty")
	}

	return NodeRemoteHardwareSettings{
		NodeID:                  strings.TrimSpace(target.NodeID),
		Enabled:                 hardware.GetEnabled(),
		AllowUndefinedPinAccess: hardware.GetAllowUndefinedPinAccess(),
		AvailablePins: func() []uint32 {
			pins := hardware.GetAvailablePins()
			out := make([]uint32, 0, len(pins))
			for _, pin := range pins {
				if pin == nil {
					continue
				}
				out = append(out, pin.GetGpioPin())
			}

			return out
		}(),
	}, nil
}

func (s *NodeSettingsService) SaveRemoteHardwareSettings(ctx context.Context, target NodeSettingsTarget, settings NodeRemoteHardwareSettings) error {
	availablePins := make([]*generated.RemoteHardwarePin, 0, len(settings.AvailablePins))
	for _, pin := range settings.AvailablePins {
		availablePins = append(availablePins, &generated.RemoteHardwarePin{GpioPin: pin})
	}

	return s.saveModuleConfig(ctx, target, "set_module_config.remote_hardware", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_RemoteHardware{
			RemoteHardware: &generated.ModuleConfig_RemoteHardwareConfig{
				Enabled:                 settings.Enabled,
				AllowUndefinedPinAccess: settings.AllowUndefinedPinAccess,
				AvailablePins:           availablePins,
			},
		},
	})
}
