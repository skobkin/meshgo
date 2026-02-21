package domain

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/skobkin/meshgo/internal/bus"
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
	coreSub := b.Subscribe(bus.TopicNodeCore)
	positionSub := b.Subscribe(bus.TopicNodePosition)
	telemetrySub := b.Subscribe(bus.TopicNodeTelemetry)
	go func() {
		defer b.Unsubscribe(coreSub, bus.TopicNodeCore)
		defer b.Unsubscribe(positionSub, bus.TopicNodePosition)
		defer b.Unsubscribe(telemetrySub, bus.TopicNodeTelemetry)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-coreSub:
				if !ok {
					return
				}
				update, ok := msg.(NodeCoreUpdate)
				if !ok {
					continue
				}
				s.Upsert(nodeFromCore(update.Core))
			case msg, ok := <-positionSub:
				if !ok {
					return
				}
				update, ok := msg.(NodePositionUpdate)
				if !ok {
					continue
				}
				s.Upsert(nodeFromPosition(update.Position))
			case msg, ok := <-telemetrySub:
				if !ok {
					return
				}
				update, ok := msg.(NodeTelemetryUpdate)
				if !ok {
					continue
				}
				s.Upsert(nodeFromTelemetry(update.Telemetry))
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
		if len(node.PublicKey) == 0 {
			node.PublicKey = cloneNodePublicKey(existing.PublicKey)
		}
		if node.Channel == nil {
			node.Channel = existing.Channel
		}
		if node.Latitude == nil {
			node.Latitude = existing.Latitude
		}
		if node.Longitude == nil {
			node.Longitude = existing.Longitude
		}
		if node.Altitude == nil {
			node.Altitude = existing.Altitude
		}
		if node.PositionPrecisionBits == nil {
			node.PositionPrecisionBits = existing.PositionPrecisionBits
		}
		if node.BatteryLevel == nil {
			node.BatteryLevel = existing.BatteryLevel
		}
		if node.Voltage == nil {
			node.Voltage = existing.Voltage
		}
		if node.UptimeSeconds == nil {
			node.UptimeSeconds = existing.UptimeSeconds
		}
		if node.ChannelUtilization == nil {
			node.ChannelUtilization = existing.ChannelUtilization
		}
		if node.AirUtilTx == nil {
			node.AirUtilTx = existing.AirUtilTx
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
		if node.FirmwareVersion == "" {
			node.FirmwareVersion = existing.FirmwareVersion
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
		if node.PositionUpdatedAt.IsZero() || existing.PositionUpdatedAt.After(node.PositionUpdatedAt) {
			node.PositionUpdatedAt = existing.PositionUpdatedAt
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

func cloneNodePublicKey(key []byte) []byte {
	if len(key) == 0 {
		return nil
	}

	out := make([]byte, len(key))
	copy(out, key)

	return out
}

func nodeFromPosition(position NodePosition) Node {
	return Node{
		NodeID:                position.NodeID,
		Channel:               position.Channel,
		Latitude:              position.Latitude,
		Longitude:             position.Longitude,
		Altitude:              position.Altitude,
		PositionPrecisionBits: position.PositionPrecisionBits,
		PositionUpdatedAt:     position.PositionUpdatedAt,
		UpdatedAt:             position.UpdatedAt,
	}
}

func nodeFromTelemetry(telemetry NodeTelemetry) Node {
	return Node{
		NodeID:             telemetry.NodeID,
		Channel:            telemetry.Channel,
		BatteryLevel:       telemetry.BatteryLevel,
		Voltage:            telemetry.Voltage,
		UptimeSeconds:      telemetry.UptimeSeconds,
		ChannelUtilization: telemetry.ChannelUtilization,
		AirUtilTx:          telemetry.AirUtilTx,
		Temperature:        telemetry.Temperature,
		Humidity:           telemetry.Humidity,
		Pressure:           telemetry.Pressure,
		AirQualityIndex:    telemetry.AirQualityIndex,
		PowerVoltage:       telemetry.PowerVoltage,
		PowerCurrent:       telemetry.PowerCurrent,
		UpdatedAt:          telemetry.UpdatedAt,
	}
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
