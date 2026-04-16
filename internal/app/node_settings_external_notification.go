package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadExternalNotificationSettings(ctx context.Context, target NodeSettingsTarget) (NodeExternalNotificationSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_EXTNOTIF_CONFIG, "get_module_config.external_notification")
	if err != nil {
		return NodeExternalNotificationSettings{}, err
	}
	notification := cfg.GetExternalNotification()
	if notification == nil {
		return NodeExternalNotificationSettings{}, fmt.Errorf("external notification module config payload is empty")
	}
	ringtone, err := s.LoadRingtone(ctx, target)
	if err != nil {
		return NodeExternalNotificationSettings{}, err
	}

	return NodeExternalNotificationSettings{
		NodeID:             strings.TrimSpace(target.NodeID),
		Enabled:            notification.GetEnabled(),
		OutputMS:           notification.GetOutputMs(),
		OutputGPIO:         notification.GetOutput(),
		OutputVibraGPIO:    notification.GetOutputVibra(),
		OutputBuzzerGPIO:   notification.GetOutputBuzzer(),
		OutputActiveHigh:   notification.GetActive(),
		AlertMessageLED:    notification.GetAlertMessage(),
		AlertMessageVibra:  notification.GetAlertMessageVibra(),
		AlertMessageBuzzer: notification.GetAlertMessageBuzzer(),
		AlertBellLED:       notification.GetAlertBell(),
		AlertBellVibra:     notification.GetAlertBellVibra(),
		AlertBellBuzzer:    notification.GetAlertBellBuzzer(),
		UsePWMBuzzer:       notification.GetUsePwm(),
		NagTimeoutSecs:     notification.GetNagTimeout(),
		Ringtone:           ringtone,
		UseI2SAsBuzzer:     notification.GetUseI2SAsBuzzer(),
	}, nil
}

func (s *NodeSettingsService) SaveExternalNotificationSettings(ctx context.Context, target NodeSettingsTarget, settings NodeExternalNotificationSettings) error {
	return s.runEditSettingsWrite(ctx, target, "set_module_config.external_notification", func(saveCtx context.Context, nodeNum uint32) error {
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.external_notification", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetModuleConfig{
				SetModuleConfig: &generated.ModuleConfig{
					PayloadVariant: &generated.ModuleConfig_ExternalNotification{
						ExternalNotification: &generated.ModuleConfig_ExternalNotificationConfig{
							Enabled:            settings.Enabled,
							OutputMs:           settings.OutputMS,
							Output:             settings.OutputGPIO,
							OutputVibra:        settings.OutputVibraGPIO,
							OutputBuzzer:       settings.OutputBuzzerGPIO,
							Active:             settings.OutputActiveHigh,
							AlertMessage:       settings.AlertMessageLED,
							AlertMessageVibra:  settings.AlertMessageVibra,
							AlertMessageBuzzer: settings.AlertMessageBuzzer,
							AlertBell:          settings.AlertBellLED,
							AlertBellVibra:     settings.AlertBellVibra,
							AlertBellBuzzer:    settings.AlertBellBuzzer,
							UsePwm:             settings.UsePWMBuzzer,
							NagTimeout:         settings.NagTimeoutSecs,
							UseI2SAsBuzzer:     settings.UseI2SAsBuzzer,
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("set external notification module config: %w", err)
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_ringtone_message", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetRingtoneMessage{SetRingtoneMessage: settings.Ringtone},
		}); err != nil {
			return fmt.Errorf("set ringtone message: %w", err)
		}

		return nil
	})
}
