package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadSecuritySettings(ctx context.Context, target NodeSettingsTarget) (NodeSecuritySettings, error) {
	cfg, err := s.loadConfig(ctx, target, generated.AdminMessage_SECURITY_CONFIG, "get_config.security")
	if err != nil {
		return NodeSecuritySettings{}, err
	}
	security := cfg.GetSecurity()
	if security == nil {
		return NodeSecuritySettings{}, fmt.Errorf("security config payload is empty")
	}

	return NodeSecuritySettings{
		NodeID:              strings.TrimSpace(target.NodeID),
		PublicKey:           cloneBytes(security.GetPublicKey()),
		PrivateKey:          cloneBytes(security.GetPrivateKey()),
		AdminKeys:           cloneBytesList(security.GetAdminKey()),
		IsManaged:           security.GetIsManaged(),
		SerialEnabled:       security.GetSerialEnabled(),
		DebugLogAPIEnabled:  security.GetDebugLogApiEnabled(),
		AdminChannelEnabled: security.GetAdminChannelEnabled(),
	}, nil
}

func (s *NodeSettingsService) SaveSecuritySettings(ctx context.Context, target NodeSettingsTarget, settings NodeSecuritySettings) error {
	return s.saveConfig(ctx, target, "set_config.security", &generated.Config{
		PayloadVariant: &generated.Config_Security{
			Security: &generated.Config_SecurityConfig{
				PublicKey:           cloneBytes(settings.PublicKey),
				PrivateKey:          cloneBytes(settings.PrivateKey),
				AdminKey:            cloneBytesList(settings.AdminKeys),
				IsManaged:           settings.IsManaged,
				SerialEnabled:       settings.SerialEnabled,
				DebugLogApiEnabled:  settings.DebugLogAPIEnabled,
				AdminChannelEnabled: settings.AdminChannelEnabled,
			},
		},
	})
}
