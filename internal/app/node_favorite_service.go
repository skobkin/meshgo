package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const nodeFavoriteAdminChannel = 0

type nodeFavoriteRadioSender interface {
	SendAdmin(to uint32, channel uint32, wantResponse bool, payload *generated.AdminMessage) (string, error)
}

// NodeFavoriteService toggles Meshtastic node favorites and keeps local state in sync.
type NodeFavoriteService struct {
	radio       nodeFavoriteRadioSender
	nodeStore   *domain.NodeStore
	messageBus  bus.MessageBus
	localNodeID func() string
	connStatus  func() (busmsg.ConnectionStatus, bool)
	logger      *slog.Logger
}

func NewNodeFavoriteService(
	radio nodeFavoriteRadioSender,
	nodeStore *domain.NodeStore,
	messageBus bus.MessageBus,
	localNodeID func() string,
	connStatus func() (busmsg.ConnectionStatus, bool),
	logger *slog.Logger,
) *NodeFavoriteService {
	if logger == nil {
		logger = slog.Default().With("component", "ui.node_favorite")
	}

	return &NodeFavoriteService{
		radio:       radio,
		nodeStore:   nodeStore,
		messageBus:  messageBus,
		localNodeID: localNodeID,
		connStatus:  connStatus,
		logger:      logger,
	}
}

func (s *NodeFavoriteService) SetFavorite(ctx context.Context, targetNodeID string, favorite bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.radio == nil {
		return fmt.Errorf("node favorite service is not initialized")
	}
	if !s.isConnected() {
		return fmt.Errorf("device is not connected")
	}

	targetNodeID = strings.TrimSpace(targetNodeID)
	if targetNodeID == "" {
		return fmt.Errorf("target node id is empty")
	}
	targetNodeNum, err := parseNodeID(targetNodeID)
	if err != nil {
		return err
	}

	localNodeNum, err := s.resolveLocalNodeNum()
	if err != nil {
		return err
	}

	adminPayload := &generated.AdminMessage{}
	if favorite {
		adminPayload.PayloadVariant = &generated.AdminMessage_SetFavoriteNode{SetFavoriteNode: targetNodeNum}
	} else {
		adminPayload.PayloadVariant = &generated.AdminMessage_RemoveFavoriteNode{RemoveFavoriteNode: targetNodeNum}
	}
	if _, err := s.radio.SendAdmin(localNodeNum, nodeFavoriteAdminChannel, false, adminPayload); err != nil {
		return err
	}

	s.publishLocalFavoriteUpdate(targetNodeID, favorite)
	s.logger.Info(
		"node favorite toggled",
		"target_node_id", targetNodeID,
		"target_node_num", targetNodeNum,
		"value", favorite,
	)

	return nil
}

func (s *NodeFavoriteService) isConnected() bool {
	if s == nil || s.connStatus == nil {
		return false
	}
	status, known := s.connStatus()

	return known && status.State == busmsg.ConnectionStateConnected
}

func (s *NodeFavoriteService) resolveLocalNodeNum() (uint32, error) {
	if s == nil || s.localNodeID == nil {
		return 0, fmt.Errorf("local node id is unavailable")
	}
	localNodeID := strings.TrimSpace(s.localNodeID())
	if localNodeID == "" {
		return 0, fmt.Errorf("local node id is unavailable")
	}

	return parseNodeID(localNodeID)
}

func (s *NodeFavoriteService) publishLocalFavoriteUpdate(nodeID string, favorite bool) {
	update := domain.NodeCoreUpdate{
		Core: domain.NodeCore{
			NodeID:     strings.TrimSpace(nodeID),
			IsFavorite: boolPtr(favorite),
			UpdatedAt:  time.Now(),
		},
		FromPacket: false,
		Type:       domain.NodeUpdateTypeUnknown,
	}

	if s.messageBus != nil {
		s.messageBus.Publish(bus.TopicNodeCore, update)

		return
	}
	if s.nodeStore != nil {
		s.nodeStore.Upsert(domain.Node{
			NodeID:     update.Core.NodeID,
			IsFavorite: update.Core.IsFavorite,
			UpdatedAt:  update.Core.UpdatedAt,
		})
	}
}
