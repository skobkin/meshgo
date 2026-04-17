package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadBluetoothSettings(ctx context.Context, target NodeSettingsTarget) (NodeBluetoothSettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_BLUETOOTH_CONFIG, "get_config.bluetooth")
	if err != nil {
		return NodeBluetoothSettings{}, err
	}
	bluetooth := cfg.GetBluetooth()
	if bluetooth == nil {
		return NodeBluetoothSettings{}, fmt.Errorf("bluetooth config payload is empty")
	}

	return NodeBluetoothSettings{
		NodeID:   strings.TrimSpace(target.NodeID),
		Enabled:  bluetooth.GetEnabled(),
		Mode:     int32(bluetooth.GetMode()),
		FixedPIN: bluetooth.GetFixedPin(),
	}, nil
}

func (s *NodeSettingsService) SaveBluetoothSettings(ctx context.Context, target NodeSettingsTarget, settings NodeBluetoothSettings) error {
	return s.saveConfig(ctx, target, "set_config.bluetooth", &generated.Config{
		PayloadVariant: &generated.Config_Bluetooth{
			Bluetooth: &generated.Config_BluetoothConfig{
				Enabled:  settings.Enabled,
				Mode:     generated.Config_BluetoothConfig_PairingMode(settings.Mode),
				FixedPin: settings.FixedPIN,
			},
		},
	})
}
