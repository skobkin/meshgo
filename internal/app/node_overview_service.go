package app

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const defaultNodeOverviewTelemetryHistoryLimit = 200

type nodeOverviewRadioSender interface {
	SendNodeInfoRequest(to uint32, channel uint32, requester *generated.User) (string, error)
	SendTelemetryRequest(to uint32, channel uint32, kind radio.TelemetryRequestKind) (string, error)
}

// NodeOverviewService provides request and history actions used by node overview UI.
type NodeOverviewService struct {
	radio         nodeOverviewRadioSender
	nodeStore     *domain.NodeStore
	telemetryRepo domain.NodeTelemetryRepository
	connStatus    func() (busmsg.ConnectionStatus, bool)
	logger        *slog.Logger
}

func NewNodeOverviewService(
	radio nodeOverviewRadioSender,
	nodeStore *domain.NodeStore,
	telemetryRepo domain.NodeTelemetryRepository,
	connStatus func() (busmsg.ConnectionStatus, bool),
	logger *slog.Logger,
) *NodeOverviewService {
	if logger == nil {
		logger = slog.Default().With("component", "ui.node_overview")
	}

	return &NodeOverviewService{
		radio:         radio,
		nodeStore:     nodeStore,
		telemetryRepo: telemetryRepo,
		connStatus:    connStatus,
		logger:        logger,
	}
}

func (s *NodeOverviewService) RequestUserInfo(ctx context.Context, targetNodeID string, requester LocalNodeSnapshot) error {
	_ = ctx
	if s == nil || s.radio == nil {
		return fmt.Errorf("node overview service is not initialized")
	}
	if !s.isConnected() {
		return fmt.Errorf("device is not connected")
	}
	targetNodeNum, err := parseNodeIDForOverview(targetNodeID)
	if err != nil {
		return err
	}
	requesterID := strings.TrimSpace(requester.ID)
	if requesterID == "" {
		return fmt.Errorf("local node id is unavailable")
	}
	channel := s.resolveTargetNodeChannel(strings.TrimSpace(targetNodeID))
	requesterUser := nodeOverviewRequesterUser(requester)
	if _, err := s.radio.SendNodeInfoRequest(targetNodeNum, channel, requesterUser); err != nil {
		return err
	}
	s.logger.Info(
		"requested remote user info",
		"target_node_id", strings.TrimSpace(targetNodeID),
		"target_node_num", targetNodeNum,
		"channel", channel,
		"requester_node_id", requesterID,
	)

	return nil
}

func (s *NodeOverviewService) RequestTelemetry(ctx context.Context, targetNodeID string, kind radio.TelemetryRequestKind) error {
	_ = ctx
	if s == nil || s.radio == nil {
		return fmt.Errorf("node overview service is not initialized")
	}
	if !s.isConnected() {
		return fmt.Errorf("device is not connected")
	}
	targetNodeNum, err := parseNodeIDForOverview(targetNodeID)
	if err != nil {
		return err
	}
	channel := s.resolveTargetNodeChannel(strings.TrimSpace(targetNodeID))
	if _, err := s.radio.SendTelemetryRequest(targetNodeNum, channel, kind); err != nil {
		return err
	}
	s.logger.Info(
		"requested remote telemetry",
		"target_node_id", strings.TrimSpace(targetNodeID),
		"target_node_num", targetNodeNum,
		"kind", kind,
		"channel", channel,
	)

	return nil
}

func (s *NodeOverviewService) ListTelemetryHistory(ctx context.Context, nodeID string, limit int) ([]domain.NodeTelemetryHistoryEntry, error) {
	if s == nil || s.telemetryRepo == nil {
		return nil, fmt.Errorf("node overview telemetry repository is not initialized")
	}
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, fmt.Errorf("node id is required")
	}
	if limit <= 0 {
		limit = defaultNodeOverviewTelemetryHistoryLimit
	}

	return s.telemetryRepo.ListHistoryByNodeID(ctx, domain.NodeHistoryQuery{
		NodeID: nodeID,
		Limit:  limit,
		Order:  domain.SortDescending,
	})
}

func (s *NodeOverviewService) isConnected() bool {
	if s == nil || s.connStatus == nil {
		return false
	}
	status, known := s.connStatus()

	return known && status.State == busmsg.ConnectionStateConnected
}

func (s *NodeOverviewService) resolveTargetNodeChannel(nodeID string) uint32 {
	if s == nil || s.nodeStore == nil {
		return 0
	}
	node, ok := s.nodeStore.Get(strings.TrimSpace(nodeID))
	if !ok || node.Channel == nil {
		return 0
	}

	return *node.Channel
}

func nodeOverviewRequesterUser(snapshot LocalNodeSnapshot) *generated.User {
	user := &generated.User{
		Id:        strings.TrimSpace(snapshot.ID),
		LongName:  strings.TrimSpace(snapshot.Node.LongName),
		ShortName: strings.TrimSpace(snapshot.Node.ShortName),
	}
	if len(snapshot.Node.PublicKey) > 0 {
		user.PublicKey = cloneBytes(snapshot.Node.PublicKey)
	}
	if snapshot.Node.IsUnmessageable != nil {
		user.IsUnmessagable = boolPtr(*snapshot.Node.IsUnmessageable)
	}

	return user
}

func parseNodeIDForOverview(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("node id is empty")
	}
	if strings.HasPrefix(raw, "!") {
		v, err := strconv.ParseUint(strings.TrimPrefix(raw, "!"), 16, 32)
		if err != nil {
			return 0, fmt.Errorf("parse node id %q: %w", raw, err)
		}

		return uint32(v), nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "0x") {
		v, err := strconv.ParseUint(raw, 0, 32)
		if err != nil {
			return 0, fmt.Errorf("parse node id %q: %w", raw, err)
		}

		return uint32(v), nil
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse node id %q: %w", raw, err)
	}

	return uint32(v), nil
}
