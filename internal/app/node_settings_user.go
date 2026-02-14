package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadUserSettings(ctx context.Context, target NodeSettingsTarget) (NodeUserSettings, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeUserSettings{}, fmt.Errorf("node settings service is not initialized")
	}
	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeUserSettings{}, parseErr
	}
	s.logger.Info("requesting node user settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
			s.logger.Warn("requesting node user settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

			return NodeUserSettings{}, err
		}
		s.logger.Warn(
			"requesting node user settings timed out, retrying",
			"node_id", strings.TrimSpace(target.NodeID),
			"attempt", attempt+1,
			"max_attempts", nodeSettingsReadRetry+1,
			"error", err,
		)
	}
	user := resp.GetGetOwnerResponse()
	if user == nil {
		s.logger.Warn("requesting node user settings returned empty response", "node_id", strings.TrimSpace(target.NodeID))

		return NodeUserSettings{}, fmt.Errorf("owner response is empty")
	}
	s.logger.Info("received node user settings response", "node_id", strings.TrimSpace(target.NodeID))

	return NodeUserSettings{
		NodeID:          strings.TrimSpace(target.NodeID),
		LongName:        strings.TrimSpace(user.GetLongName()),
		ShortName:       strings.TrimSpace(user.GetShortName()),
		HamLicensed:     user.GetIsLicensed(),
		IsUnmessageable: user.GetIsUnmessagable(),
	}, nil
}

func (s *NodeSettingsService) SaveUserSettings(ctx context.Context, target NodeSettingsTarget, settings NodeUserSettings) error {
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
	s.logger.Info("saving node user settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

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
		s.logger.Warn("begin edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("begin edit settings: %w", err)
	}

	admin := &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_SetOwner{
			SetOwner: &generated.User{
				Id:             strings.TrimSpace(target.NodeID),
				LongName:       strings.TrimSpace(settings.LongName),
				ShortName:      strings.TrimSpace(settings.ShortName),
				IsLicensed:     settings.HamLicensed,
				IsUnmessagable: boolPtr(settings.IsUnmessageable),
			},
		},
	}
	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "set_owner", admin); err != nil {
		s.logger.Warn("set owner failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("set owner: %w", err)
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}
	s.logger.Info("saved node user settings", "node_id", strings.TrimSpace(target.NodeID))

	return nil
}
