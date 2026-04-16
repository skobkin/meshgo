package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadSerialSettings(ctx context.Context, target NodeSettingsTarget) (NodeSerialSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_SERIAL_CONFIG, "get_module_config.serial")
	if err != nil {
		return NodeSerialSettings{}, err
	}
	serial := cfg.GetSerial()
	if serial == nil {
		return NodeSerialSettings{}, fmt.Errorf("serial module config payload is empty")
	}

	return NodeSerialSettings{
		NodeID:                    strings.TrimSpace(target.NodeID),
		Enabled:                   serial.GetEnabled(),
		EchoEnabled:               serial.GetEcho(),
		RXGPIO:                    serial.GetRxd(),
		TXGPIO:                    serial.GetTxd(),
		Baud:                      int32(serial.GetBaud()),
		Timeout:                   serial.GetTimeout(),
		Mode:                      int32(serial.GetMode()),
		OverrideConsoleSerialPort: serial.GetOverrideConsoleSerialPort(),
	}, nil
}

func (s *NodeSettingsService) SaveSerialSettings(ctx context.Context, target NodeSettingsTarget, settings NodeSerialSettings) error {
	return s.saveModuleConfig(ctx, target, "set_module_config.serial", &generated.ModuleConfig{
		PayloadVariant: &generated.ModuleConfig_Serial{
			Serial: &generated.ModuleConfig_SerialConfig{
				Enabled:                   settings.Enabled,
				Echo:                      settings.EchoEnabled,
				Rxd:                       settings.RXGPIO,
				Txd:                       settings.TXGPIO,
				Baud:                      generated.ModuleConfig_SerialConfig_Serial_Baud(settings.Baud),
				Timeout:                   settings.Timeout,
				Mode:                      generated.ModuleConfig_SerialConfig_Serial_Mode(settings.Mode),
				OverrideConsoleSerialPort: settings.OverrideConsoleSerialPort,
			},
		},
	})
}
