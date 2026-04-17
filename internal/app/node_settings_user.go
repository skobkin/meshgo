package app

import (
	"context"
	"fmt"
	"strings"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func (s *NodeSettingsService) LoadUserSettings(ctx context.Context, target NodeSettingsTarget) (NodeUserSettings, error) {
	user, err := s.loadOwner(ctx, target)
	if err != nil {
		return NodeUserSettings{}, err
	}
	if user == nil {
		return NodeUserSettings{}, fmt.Errorf("owner response is empty")
	}

	return NodeUserSettings{
		NodeID:          strings.TrimSpace(target.NodeID),
		LongName:        strings.TrimSpace(user.GetLongName()),
		ShortName:       strings.TrimSpace(user.GetShortName()),
		HamLicensed:     user.GetIsLicensed(),
		IsUnmessageable: user.GetIsUnmessagable(),
	}, nil
}

func (s *NodeSettingsService) SaveUserSettings(ctx context.Context, target NodeSettingsTarget, settings NodeUserSettings) error {
	return s.saveOwner(ctx, target, &generated.User{
		Id:             strings.TrimSpace(target.NodeID),
		LongName:       strings.TrimSpace(settings.LongName),
		ShortName:      strings.TrimSpace(settings.ShortName),
		IsLicensed:     settings.HamLicensed,
		IsUnmessagable: boolPtr(settings.IsUnmessageable),
	})
}
