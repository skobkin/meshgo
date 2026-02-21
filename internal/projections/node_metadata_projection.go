package projections

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

// NodeMetadataProjection maps admin metadata responses into NodeUpdate events.
type NodeMetadataProjection struct{}

func NewNodeMetadataProjection() *NodeMetadataProjection {
	return &NodeMetadataProjection{}
}

func (p *NodeMetadataProjection) Start(ctx context.Context, messageBus bus.MessageBus) {
	if p == nil || messageBus == nil {
		return
	}

	adminSub := messageBus.Subscribe(bus.TopicAdminMessage)
	go func() {
		defer messageBus.Unsubscribe(adminSub, bus.TopicAdminMessage)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-adminSub:
				if !ok {
					return
				}
				event, ok := raw.(busmsg.AdminMessageEvent)
				if !ok || event.Message == nil || event.From == 0 {
					continue
				}
				metadata := event.Message.GetGetDeviceMetadataResponse()
				if metadata == nil {
					continue
				}
				nodeID := fmt.Sprintf("!%08x", event.From)
				now := time.Now()
				node := domain.NodeCore{
					NodeID:          nodeID,
					FirmwareVersion: strings.TrimSpace(metadata.GetFirmwareVersion()),
					LastHeardAt:     now,
					UpdatedAt:       now,
				}
				if model := metadata.GetHwModel(); model != generated.HardwareModel_UNSET {
					node.BoardModel = model.String()
				}
				if role := strings.TrimSpace(metadata.GetRole().String()); role != "" {
					node.Role = role
				}
				messageBus.Publish(bus.TopicNodeCore, domain.NodeCoreUpdate{
					Core:       node,
					FromPacket: true,
					Type:       domain.NodeUpdateTypeMetadata,
				})
			}
		}
	}()
}
