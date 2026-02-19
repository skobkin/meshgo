package app

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const (
	// NodeChannelMaxSlots is the currently supported channel slot count.
	// Keep this as a single constant so raising channel limits is a small update.
	NodeChannelMaxSlots = 8

	nodeSettingsChannelsReadTimeout = nodeSettingsOpTimeout + 8*time.Second
	nodeSettingsChannelsSaveTimeout = nodeSettingsOpTimeout + 20*time.Second
)

func (s *NodeSettingsService) LoadChannelSettings(ctx context.Context, target NodeSettingsTarget) (NodeChannelSettingsList, error) {
	if s == nil || s.bus == nil || s.radio == nil {
		return NodeChannelSettingsList{}, fmt.Errorf("node settings service is not initialized")
	}

	nodeNum, parseErr := parseNodeID(target.NodeID)
	if parseErr != nil {
		return NodeChannelSettingsList{}, parseErr
	}
	s.logger.Info("requesting node channel settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	loadCtx, cancel := context.WithTimeout(ctx, nodeSettingsChannelsReadTimeout)
	defer cancel()

	slots := make([]*NodeChannelSettings, NodeChannelMaxSlots)
	for requestIndex := uint32(1); requestIndex <= uint32(NodeChannelMaxSlots); requestIndex++ {
		action := fmt.Sprintf("get_channel.%d", requestIndex)

		var (
			resp *generated.AdminMessage
			err  error
		)
		for attempt := 0; attempt <= nodeSettingsReadRetry; attempt++ {
			resp, err = s.sendAdminAndWaitResponse(loadCtx, nodeNum, action, &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_GetChannelRequest{GetChannelRequest: requestIndex},
			})
			if err == nil {
				break
			}
			if attempt >= nodeSettingsReadRetry || !isRetriableReadError(err) {
				s.logger.Warn(
					"requesting node channel settings failed",
					"node_id", strings.TrimSpace(target.NodeID),
					"request_index", requestIndex,
					"error", err,
				)

				return NodeChannelSettingsList{}, fmt.Errorf("load channel %d: %w", requestIndex, err)
			}
			s.logger.Warn(
				"requesting node channel settings timed out, retrying",
				"node_id", strings.TrimSpace(target.NodeID),
				"request_index", requestIndex,
				"attempt", attempt+1,
				"max_attempts", nodeSettingsReadRetry+1,
				"error", err,
			)
		}

		channel := resp.GetGetChannelResponse()
		if channel == nil {
			s.logger.Warn(
				"requesting node channel settings returned empty channel payload",
				"node_id", strings.TrimSpace(target.NodeID),
				"request_index", requestIndex,
			)

			return NodeChannelSettingsList{}, fmt.Errorf("channel %d response is empty", requestIndex)
		}

		index := int(channel.GetIndex())
		if index < 0 || index >= NodeChannelMaxSlots {
			s.logger.Warn(
				"requesting node channel settings returned out-of-range index",
				"node_id", strings.TrimSpace(target.NodeID),
				"request_index", requestIndex,
				"index", index,
			)

			continue
		}
		if channel.GetRole() == generated.Channel_DISABLED {
			slots[index] = nil

			continue
		}

		settings := nodeChannelSettingsFromProto(channel.GetSettings())
		slots[index] = &settings
	}

	loaded := NodeChannelSettingsList{
		NodeID:   strings.TrimSpace(target.NodeID),
		MaxSlots: NodeChannelMaxSlots,
		Channels: make([]NodeChannelSettings, 0, NodeChannelMaxSlots),
	}
	for index := 0; index < NodeChannelMaxSlots; index++ {
		if slots[index] == nil {
			continue
		}
		loaded.Channels = append(loaded.Channels, cloneNodeChannelSettings(*slots[index]))
	}

	s.logger.Info(
		"received node channel settings response",
		"node_id", strings.TrimSpace(target.NodeID),
		"channel_count", len(loaded.Channels),
		"max_slots", loaded.MaxSlots,
	)

	return loaded, nil
}

func (s *NodeSettingsService) SaveChannelSettings(ctx context.Context, target NodeSettingsTarget, settings NodeChannelSettingsList) error {
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
	s.logger.Info("saving node channel settings", "node_id", strings.TrimSpace(target.NodeID), "node_num", nodeNum)

	release, err := s.beginSave()
	if err != nil {
		return err
	}
	defer release()

	saveCtx, cancel := context.WithTimeout(ctx, nodeSettingsChannelsSaveTimeout)
	defer cancel()

	current, err := s.LoadChannelSettings(saveCtx, target)
	if err != nil {
		s.logger.Warn("loading current channel settings before save failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("load current channels: %w", err)
	}

	targetMaxSlots := settings.MaxSlots
	if targetMaxSlots <= 0 {
		targetMaxSlots = NodeChannelMaxSlots
	}
	if targetMaxSlots > NodeChannelMaxSlots {
		targetMaxSlots = NodeChannelMaxSlots
	}

	changes := diffNodeChannels(current.Channels, settings.Channels, targetMaxSlots)
	if len(changes) == 0 {
		s.logger.Info("saving node channel settings skipped: no changes detected", "node_id", strings.TrimSpace(target.NodeID))

		return nil
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "begin_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_BeginEditSettings{BeginEditSettings: true},
	}); err != nil {
		s.logger.Warn("begin edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("begin edit settings: %w", err)
	}

	for index, change := range changes {
		action := fmt.Sprintf("set_channel.%d", change.GetIndex())
		if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, action, &generated.AdminMessage{
			PayloadVariant: &generated.AdminMessage_SetChannel{SetChannel: change},
		}); err != nil {
			s.logger.Warn(
				"set channel failed",
				"node_id", strings.TrimSpace(target.NodeID),
				"channel_index", change.GetIndex(),
				"change", index+1,
				"change_total", len(changes),
				"error", err,
			)

			return fmt.Errorf(
				"set channel %d/%d (index %d): %w",
				index+1,
				len(changes),
				change.GetIndex(),
				err,
			)
		}
	}

	if err := s.sendAdminAndWaitStatus(saveCtx, nodeNum, "commit_edit_settings", &generated.AdminMessage{
		PayloadVariant: &generated.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}); err != nil {
		s.logger.Warn("commit edit settings failed", "node_id", strings.TrimSpace(target.NodeID), "error", err)

		return fmt.Errorf("commit edit settings: %w", err)
	}

	s.logger.Info(
		"saved node channel settings",
		"node_id", strings.TrimSpace(target.NodeID),
		"change_count", len(changes),
	)

	return nil
}

func diffNodeChannels(current []NodeChannelSettings, desired []NodeChannelSettings, maxSlots int) []*generated.Channel {
	if maxSlots <= 0 {
		maxSlots = NodeChannelMaxSlots
	}
	if maxSlots > NodeChannelMaxSlots {
		maxSlots = NodeChannelMaxSlots
	}
	maxSlots32, ok := safeIntToInt32(maxSlots)
	if !ok {
		maxSlots32 = int32(NodeChannelMaxSlots)
	}

	changes := make([]*generated.Channel, 0, maxSlots)
	for index := int32(0); index < maxSlots32; index++ {
		slot := int(index)
		currentSettings, hasCurrent := nodeChannelAt(current, slot)
		desiredSettings, hasDesired := nodeChannelAt(desired, slot)
		if hasCurrent == hasDesired && (!hasCurrent || nodeChannelSettingsEqual(currentSettings, desiredSettings)) {
			continue
		}

		channel := &generated.Channel{Index: index}
		if hasDesired {
			if slot == 0 {
				channel.Role = generated.Channel_PRIMARY
			} else {
				channel.Role = generated.Channel_SECONDARY
			}
			channel.Settings = nodeChannelSettingsToProto(desiredSettings)
		} else {
			channel.Role = generated.Channel_DISABLED
			channel.Settings = &generated.ChannelSettings{}
		}
		changes = append(changes, channel)
	}

	return changes
}

func nodeChannelAt(channels []NodeChannelSettings, index int) (NodeChannelSettings, bool) {
	if index < 0 || index >= len(channels) {
		return NodeChannelSettings{}, false
	}

	return cloneNodeChannelSettings(channels[index]), true
}

func nodeChannelSettingsFromProto(settings *generated.ChannelSettings) NodeChannelSettings {
	if settings == nil {
		return NodeChannelSettings{}
	}

	module := settings.GetModuleSettings()

	return NodeChannelSettings{
		Name:              strings.TrimSpace(settings.GetName()),
		PSK:               cloneBytes(settings.GetPsk()),
		ID:                settings.GetId(),
		UplinkEnabled:     settings.GetUplinkEnabled(),
		DownlinkEnabled:   settings.GetDownlinkEnabled(),
		PositionPrecision: module.GetPositionPrecision(),
		Muted:             module.GetIsMuted(),
	}
}

func nodeChannelSettingsToProto(settings NodeChannelSettings) *generated.ChannelSettings {
	module := &generated.ModuleSettings{
		PositionPrecision: settings.PositionPrecision,
		IsMuted:           settings.Muted,
	}

	return &generated.ChannelSettings{
		Psk:             cloneBytes(settings.PSK),
		Name:            strings.TrimSpace(settings.Name),
		Id:              settings.ID,
		UplinkEnabled:   settings.UplinkEnabled,
		DownlinkEnabled: settings.DownlinkEnabled,
		ModuleSettings:  module,
	}
}

func nodeChannelSettingsEqual(left, right NodeChannelSettings) bool {
	return strings.TrimSpace(left.Name) == strings.TrimSpace(right.Name) &&
		left.ID == right.ID &&
		left.UplinkEnabled == right.UplinkEnabled &&
		left.DownlinkEnabled == right.DownlinkEnabled &&
		left.PositionPrecision == right.PositionPrecision &&
		left.Muted == right.Muted &&
		bytes.Equal(left.PSK, right.PSK)
}

func cloneNodeChannelSettings(value NodeChannelSettings) NodeChannelSettings {
	return NodeChannelSettings{
		Name:              strings.TrimSpace(value.Name),
		PSK:               cloneBytes(value.PSK),
		ID:                value.ID,
		UplinkEnabled:     value.UplinkEnabled,
		DownlinkEnabled:   value.DownlinkEnabled,
		PositionPrecision: value.PositionPrecision,
		Muted:             value.Muted,
	}
}

func safeIntToInt32(value int) (int32, bool) {
	if value > math.MaxInt32 || value < math.MinInt32 {
		return 0, false
	}

	return int32(value), true
}
