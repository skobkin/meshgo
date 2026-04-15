package app

import (
	"context"
	"fmt"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) RebootNode(ctx context.Context, target NodeSettingsTarget) error {
	return s.sendAdminAction(ctx, target, "reboot_seconds", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_RebootSeconds{RebootSeconds: 1},
	})
}

func (s *NodeSettingsService) ShutdownNode(ctx context.Context, target NodeSettingsTarget) error {
	return s.sendAdminAction(ctx, target, "shutdown_seconds", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_ShutdownSeconds{ShutdownSeconds: 1},
	})
}

func (s *NodeSettingsService) FactoryResetNode(ctx context.Context, target NodeSettingsTarget) error {
	return s.sendAdminAction(ctx, target, "factory_reset_config", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_FactoryResetConfig{FactoryResetConfig: 1},
	})
}

func (s *NodeSettingsService) ResetNodeDB(ctx context.Context, target NodeSettingsTarget, preserveFavorites bool) error {
	_ = preserveFavorites

	return s.sendAdminAction(ctx, target, "nodedb_reset", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_NodedbReset{NodedbReset: true},
	})
}

func (s *NodeSettingsService) sendAdminAction(ctx context.Context, target NodeSettingsTarget, action string, message *generated.AdminMessage) error {
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

	actionCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	if err := s.sendAdminAndWaitStatus(actionCtx, nodeNum, action, message); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
