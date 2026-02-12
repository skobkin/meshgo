package domain

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/connectors"
)

// NodeStore keeps the latest node snapshots in memory for the UI.
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
		if node.Latitude == nil {
			node.Latitude = existing.Latitude
		}
		if node.Longitude == nil {
			node.Longitude = existing.Longitude
		}
		if node.BatteryLevel == nil {
			node.BatteryLevel = existing.BatteryLevel
		}
		if node.Voltage == nil {
			node.Voltage = existing.Voltage
		}
		if node.Temperature == nil {
			node.Temperature = existing.Temperature
		}
		if node.Humidity == nil {
			node.Humidity = existing.Humidity
		}
		if node.Pressure == nil {
			node.Pressure = existing.Pressure
		}
		if node.AirQualityIndex == nil {
			node.AirQualityIndex = existing.AirQualityIndex
		}
		if node.PowerVoltage == nil {
			node.PowerVoltage = existing.PowerVoltage
		}
		if node.PowerCurrent == nil {
			node.PowerCurrent = existing.PowerCurrent
		}
		if node.BoardModel == "" {
			node.BoardModel = existing.BoardModel
		}
		if node.Role == "" {
			node.Role = existing.Role
		}
		if node.IsUnmessageable == nil {
			node.IsUnmessageable = existing.IsUnmessageable
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

func (s *NodeStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes = make(map[string]Node)
	s.notify()
}

func (s *NodeStore) notify() {
	select {
	case s.changes <- struct{}{}:
	default:
	}
}
