package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadCannedMessageSettings(ctx context.Context, target NodeSettingsTarget) (NodeCannedMessageSettings, error) {
	cfg, err := s.loadModuleConfig(ctx, target, generated.AdminMessage_CANNEDMSG_CONFIG, "get_module_config.canned_message")
	if err != nil {
		return NodeCannedMessageSettings{}, err
	}
	canned := cfg.GetCannedMessage()
	if canned == nil {
		return NodeCannedMessageSettings{}, fmt.Errorf("canned message module config payload is empty")
	}
	enabled := canned.GetEnabled()                   //nolint:staticcheck // device protocol still exposes deprecated fields
	allowInputSource := canned.GetAllowInputSource() //nolint:staticcheck // device protocol still exposes deprecated fields
	messages, err := s.LoadCannedMessages(ctx, target)
	if err != nil {
		return NodeCannedMessageSettings{}, err
	}

	return NodeCannedMessageSettings{
		NodeID:                strings.TrimSpace(target.NodeID),
		Rotary1Enabled:        canned.GetRotary1Enabled(),
		InputBrokerPinA:       canned.GetInputbrokerPinA(),
		InputBrokerPinB:       canned.GetInputbrokerPinB(),
		InputBrokerPinPress:   canned.GetInputbrokerPinPress(),
		InputBrokerEventCW:    int32(canned.GetInputbrokerEventCw()),
		InputBrokerEventCCW:   int32(canned.GetInputbrokerEventCcw()),
		InputBrokerEventPress: int32(canned.GetInputbrokerEventPress()),
		UpDown1Enabled:        canned.GetUpdown1Enabled(),
		Enabled:               enabled,
		AllowInputSource:      allowInputSource,
		SendBell:              canned.GetSendBell(),
		Messages:              messages,
	}, nil
}

func (s *NodeSettingsService) SaveCannedMessageSettings(ctx context.Context, target NodeSettingsTarget, settings NodeCannedMessageSettings) error {
	return s.runEditSettingsWrite(ctx, target, "set_module_config.canned_message", func(saveCtx context.Context, nodeNum uint32) error {
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_module_config.canned_message", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetModuleConfig{
				SetModuleConfig: &generated.ModuleConfig{
					PayloadVariant: &generated.ModuleConfig_CannedMessage{
						CannedMessage: &generated.ModuleConfig_CannedMessageConfig{
							Rotary1Enabled:        settings.Rotary1Enabled,
							InputbrokerPinA:       settings.InputBrokerPinA,
							InputbrokerPinB:       settings.InputBrokerPinB,
							InputbrokerPinPress:   settings.InputBrokerPinPress,
							InputbrokerEventCw:    generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventCW),
							InputbrokerEventCcw:   generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventCCW),
							InputbrokerEventPress: generated.ModuleConfig_CannedMessageConfig_InputEventChar(settings.InputBrokerEventPress),
							Updown1Enabled:        settings.UpDown1Enabled,
							Enabled:               settings.Enabled,
							AllowInputSource:      settings.AllowInputSource,
							SendBell:              settings.SendBell,
						},
					},
				},
			},
		}); err != nil {
			return fmt.Errorf("set canned message module config: %w", err)
		}

		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_canned_message_module_messages", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetCannedMessageModuleMessages{SetCannedMessageModuleMessages: settings.Messages},
		}); err != nil {
			return fmt.Errorf("set canned messages: %w", err)
		}

		return nil
	})
}
