package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

func (s *NodeSettingsService) loadConfig(
	ctx context.Context,
	target NodeSettingsTarget,
	configType generated.AdminMessage_ConfigType,
	action string,
) (*generated.Config, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return nil, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return nil, parseErr
	}
	s.logger.Info("requesting node config", "action", action, "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, action, &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetConfigRequest{GetConfigRequest: configType},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			return nil, err
		}
	}

	cfg := resp.GetGetConfigResponse()
	if cfg == nil {
		return nil, fmt.Errorf("%s response is empty", action)
	}

	return cfg, nil
}

func (s *NodeSettingsService) loadModuleConfig(
	ctx context.Context,
	target NodeSettingsTarget,
	configType generated.AdminMessage_ModuleConfigType,
	action string,
) (*generated.ModuleConfig, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return nil, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return nil, parseErr
	}
	s.logger.Info("requesting node module config", "action", action, "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, action, &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetModuleConfigRequest{
				GetModuleConfigRequest: configType,
			},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			return nil, err
		}
	}

	cfg := resp.GetGetModuleConfigResponse()
	if cfg == nil {
		return nil, fmt.Errorf("%s response is empty", action)
	}

	return cfg, nil
}

func (s *NodeSettingsService) saveConfig(
	ctx context.Context,
	target NodeSettingsTarget,
	action string,
	config *generated.Config,
) error {
	if config == nil {
		return fmt.Errorf("%s config payload is empty", action)
	}

	return s.runEditSettingsWrite(ctx, target, action, func(saveCtx context.Context, nodeNum uint32) error {
		return s.sendAdminAndWaitStatus(saveCtx, nodeNum, action, &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetConfig{SetConfig: config},
		})
	})
}

func (s *NodeSettingsService) saveModuleConfig(
	ctx context.Context,
	target NodeSettingsTarget,
	action string,
	config *generated.ModuleConfig,
) error {
	if config == nil {
		return fmt.Errorf("%s module config payload is empty", action)
	}

	return s.runEditSettingsWrite(ctx, target, action, func(saveCtx context.Context, nodeNum uint32) error {
		return s.sendAdminAndWaitStatus(saveCtx, nodeNum, action, &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetModuleConfig{SetModuleConfig: config},
		})
	})
}

func (s *NodeSettingsService) runEditSettingsWrite(
	ctx context.Context,
	target NodeSettingsTarget,
	_ string,
	write func(context.Context, uint32) error,
) error {
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

	release, err := s.beginSave()
	if err != nil {
		return err
	}
	defer release()

	saveCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "begin_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_BeginEditSettings{BeginEditSettings: true},
	}); err != nil {
		return fmt.Errorf("begin edit settings: %w", err)
	}

	if err := write(saveCtx, nodeNum); err != nil {
		return err
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		return fmt.Errorf("commit edit settings: %w", err)
	}

	return nil
}

func (s *NodeSettingsService) loadAdminString(
	ctx context.Context,
	target NodeSettingsTarget,
	action string,
	request *generated.AdminMessage,
	extract func(*generated.AdminMessage) string,
) (string, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return "", fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return "", parseErr
	}

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	resp, err := s.sendAdminAndWaitResponse(loadCtx, nodeNum, action, request)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(extract(resp)), nil
}

func (s *NodeSettingsService) saveAdminString(
	ctx context.Context,
	target NodeSettingsTarget,
	action string,
	writeMessage func(string) *generated.AdminMessage,
	value string,
) error {
	return s.runEditSettingsWrite(ctx, target, action, func(saveCtx context.Context, nodeNum uint32) error {
		return s.sendAdminAndWaitStatus(saveCtx, nodeNum, action, writeMessage(value))
	})
}

func (s *NodeSettingsService) loadOwner(ctx context.Context, target NodeSettingsTarget) (*generated.User, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return nil, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return nil, parseErr
	}
	s.logger.Info("requesting node owner settings", "action", "get_owner", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsOpTimeout)
	defer cancel()

	var (
		resp *generated.AdminMessage
		err  error
	)
	for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
		resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, "get_owner", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_GetOwnerRequest{GetOwnerRequest: true},
		})
		if err == nil {
			break
		}
		if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
			return nil, err
		}
	}

	owner := resp.GetGetOwnerResponse()
	if owner == nil {
		return nil, fmt.Errorf("owner response is empty")
	}

	return owner, nil
}

func (s *NodeSettingsService) saveOwner(ctx context.Context, target NodeSettingsTarget, owner *generated.User) error {
	if owner == nil {
		return fmt.Errorf("set_owner owner payload is empty")
	}

	return s.runEditSettingsWrite(ctx, target, "set_owner", func(saveCtx context.Context, nodeNum uint32) error {
		return s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_owner", &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetOwner{SetOwner: owner},
		})
	})
}

func cloneProtoMessage[T proto.Message](in T) T {
	if any(in) == nil {
		var zero T

		return zero
	}

	return proto.Clone(in).(T)
}

func cloneUint32Slice(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}

	out := make([]uint32, len(values))
	copy(out, values)

	return out
}
