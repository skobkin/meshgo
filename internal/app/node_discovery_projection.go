package app

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

const nodeDiscoverySourceNodeInfoPacket = "nodeinfo_packet"

// NodeDiscoveryProjection emits TopicNodeDiscovered from post-bootstrap live node info packets.
type NodeDiscoveryProjection struct {
	logger *slog.Logger

	mu               sync.Mutex
	bootstrapReady   bool
	bootstrapCutover time.Time
	knownNodeIDs     map[string]struct{}
	seenNodeIDs      map[string]struct{}
}

func NewNodeDiscoveryProjection(nodeStore *domain.NodeStore, logger *slog.Logger) *NodeDiscoveryProjection {
	if logger == nil {
		logger = slog.Default().With("component", "app.node_discovery")
	}

	return &NodeDiscoveryProjection{
		logger:       logger,
		knownNodeIDs: snapshotNodeIDs(nodeStore),
		seenNodeIDs:  make(map[string]struct{}),
	}
}

func (p *NodeDiscoveryProjection) Start(ctx context.Context, messageBus bus.MessageBus) {
	if p == nil || messageBus == nil {
		return
	}
	nodeSub := messageBus.Subscribe(connectors.TopicNodeInfo)
	radioSub := messageBus.Subscribe(connectors.TopicRadioFrom)
	connSub := messageBus.Subscribe(connectors.TopicConnStatus)

	go func() {
		defer messageBus.Unsubscribe(nodeSub, connectors.TopicNodeInfo)
		defer messageBus.Unsubscribe(radioSub, connectors.TopicRadioFrom)
		defer messageBus.Unsubscribe(connSub, connectors.TopicConnStatus)
		for {
			select {
			case <-ctx.Done():
				return
			case raw, ok := <-connSub:
				if !ok {
					return
				}
				status, ok := raw.(connectors.ConnectionStatus)
				if !ok {
					continue
				}
				p.handleConnStatus(status)
			case raw, ok := <-radioSub:
				if !ok {
					return
				}
				frame, ok := raw.(radio.DecodedFrame)
				if !ok {
					continue
				}
				p.handleRadioFrame(frame)
			case raw, ok := <-nodeSub:
				if !ok {
					return
				}
				update, ok := raw.(domain.NodeUpdate)
				if !ok {
					continue
				}
				event, shouldPublish := p.nodeDiscoveredEvent(update)
				if !shouldPublish {
					continue
				}
				messageBus.Publish(connectors.TopicNodeDiscovered, event)
				p.logger.Info("node discovered", "node_id", event.NodeID, "source", event.Source)
			}
		}
	}()
}

// ResetFromStore updates discovery baseline and clears runtime deduplication.
func (p *NodeDiscoveryProjection) ResetFromStore(nodeStore *domain.NodeStore) {
	if p == nil {
		return
	}
	known := snapshotNodeIDs(nodeStore)
	p.mu.Lock()
	p.bootstrapReady = false
	p.bootstrapCutover = time.Time{}
	p.knownNodeIDs = known
	p.seenNodeIDs = make(map[string]struct{})
	p.mu.Unlock()
	p.logger.Info("node discovery baseline reset", "known_nodes", len(known))
}

func (p *NodeDiscoveryProjection) handleConnStatus(status connectors.ConnectionStatus) {
	if p == nil || status.State == "" {
		return
	}
	if status.State == connectors.ConnectionStateConnected {
		return
	}
	p.mu.Lock()
	p.bootstrapReady = false
	p.bootstrapCutover = time.Time{}
	p.mu.Unlock()
}

func (p *NodeDiscoveryProjection) handleRadioFrame(frame radio.DecodedFrame) {
	if p == nil || !frame.WantConfigReady {
		return
	}
	now := time.Now()
	p.mu.Lock()
	if p.bootstrapReady {
		p.mu.Unlock()

		return
	}
	p.bootstrapReady = true
	p.bootstrapCutover = now
	p.mu.Unlock()
	p.logger.Debug("node discovery armed after initial bootstrap")
}

func (p *NodeDiscoveryProjection) nodeDiscoveredEvent(update domain.NodeUpdate) (domain.NodeDiscovered, bool) {
	if p == nil || update.Type != domain.NodeUpdateTypeNodeInfoPacket {
		return domain.NodeDiscovered{}, false
	}
	nodeID := strings.TrimSpace(update.Node.NodeID)
	if nodeID == "" {
		return domain.NodeDiscovered{}, false
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.bootstrapReady {
		return domain.NodeDiscovered{}, false
	}
	if !nodeUpdateAtOrAfterCutover(update, p.bootstrapCutover) {
		return domain.NodeDiscovered{}, false
	}
	if _, ok := p.knownNodeIDs[nodeID]; ok {
		return domain.NodeDiscovered{}, false
	}
	if _, ok := p.seenNodeIDs[nodeID]; ok {
		return domain.NodeDiscovered{}, false
	}
	p.knownNodeIDs[nodeID] = struct{}{}
	p.seenNodeIDs[nodeID] = struct{}{}

	return domain.NodeDiscovered{
		Node:         update.Node,
		NodeID:       nodeID,
		DiscoveredAt: time.Now(),
		Source:       nodeDiscoverySourceNodeInfoPacket,
	}, true
}

func nodeUpdateAtOrAfterCutover(update domain.NodeUpdate, cutover time.Time) bool {
	if cutover.IsZero() {
		return true
	}
	observedAt := update.Node.UpdatedAt
	if observedAt.IsZero() {
		return true
	}

	return !observedAt.Before(cutover)
}

func snapshotNodeIDs(nodeStore *domain.NodeStore) map[string]struct{} {
	known := make(map[string]struct{})
	if nodeStore == nil {
		return known
	}
	for _, node := range nodeStore.SnapshotSorted() {
		id := strings.TrimSpace(node.NodeID)
		if id == "" {
			continue
		}
		known[id] = struct{}{}
	}

	return known
}
