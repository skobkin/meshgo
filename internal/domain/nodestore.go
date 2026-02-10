package domain

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

type NodeStore struct {
	mu      sync.RWMutex
	nodes   map[string]Node
	changes chan struct{}
}

func NewNodeStore() *NodeStore {
	return &NodeStore{
		nodes:   make(map[string]Node),
		changes: make(chan struct{}, 1),
	}
}

func (s *NodeStore) Load(nodes []Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range nodes {
		s.nodes[node.NodeID] = node
	}
	s.notify()
}

func (s *NodeStore) Start(ctx context.Context, b bus.MessageBus) {
	sub := b.Subscribe(connectors.TopicNodeInfo)
	go func() {
		defer b.Unsubscribe(sub, connectors.TopicNodeInfo)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-sub:
				if !ok {
					return
				}
				update, ok := msg.(NodeUpdate)
				if !ok {
					continue
				}
				s.Upsert(update.Node)
			}
		}
	}()
}

func (s *NodeStore) Upsert(node Node) {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.nodes[node.NodeID]
	if ok {
		// Merge sparse updates without wiping cached metadata.
		if node.LongName == "" {
			node.LongName = existing.LongName
		}
		if node.ShortName == "" {
			node.ShortName = existing.ShortName
		}
		if node.RSSI == nil {
			node.RSSI = existing.RSSI
		}
		if node.SNR == nil {
			node.SNR = existing.SNR
		}
		if node.LastHeardAt.IsZero() || existing.LastHeardAt.After(node.LastHeardAt) {
			node.LastHeardAt = existing.LastHeardAt
		}
		if existing.UpdatedAt.After(node.UpdatedAt) {
			node.UpdatedAt = existing.UpdatedAt
		}
	}
	if node.UpdatedAt.IsZero() {
		node.UpdatedAt = time.Now()
	}
	s.nodes[node.NodeID] = node
	s.notify()
}

func (s *NodeStore) SnapshotSorted() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		out = append(out, node)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastHeardAt.After(out[j].LastHeardAt)
	})
	return out
}

func (s *NodeStore) Get(nodeID string) (Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[nodeID]
	return node, ok
}

func (s *NodeStore) Changes() <-chan struct{} {
	return s.changes
}

func (s *NodeStore) notify() {
	select {
	case s.changes <- struct{}{}:
	default:
	}
}
