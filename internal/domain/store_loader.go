package domain

import (
	"context"
	"fmt"
)

const defaultRecentMessagesLoad = 200

func LoadStoresFromRepositories(
	ctx context.Context,
	nodes *NodeStore,
	chats *ChatStore,
	coreRepo NodeCoreRepository,
	positionRepo NodePositionRepository,
	telemetryRepo NodeTelemetryRepository,
	chatRepo ChatRepository,
	msgRepo MessageRepository,
) error {
	coreItems, err := coreRepo.ListSortedByLastHeard(ctx)
	if err != nil {
		return fmt.Errorf("load node core from db: %w", err)
	}
	positionItems, err := positionRepo.ListLatest(ctx)
	if err != nil {
		return fmt.Errorf("load node position from db: %w", err)
	}
	telemetryItems, err := telemetryRepo.ListLatest(ctx)
	if err != nil {
		return fmt.Errorf("load node telemetry from db: %w", err)
	}
	chatItems, err := chatRepo.ListSortedByLastSentByMe(ctx)
	if err != nil {
		return fmt.Errorf("load chats from db: %w", err)
	}
	messageItems, err := msgRepo.LoadRecentPerChat(ctx, defaultRecentMessagesLoad)
	if err != nil {
		return fmt.Errorf("load messages from db: %w", err)
	}

	nodes.Load(mergeNodeSnapshots(coreItems, positionItems, telemetryItems))
	chats.Load(chatItems, messageItems)

	return nil
}

func mergeNodeSnapshots(coreItems []NodeCore, positionItems []NodePosition, telemetryItems []NodeTelemetry) []Node {
	outByID := make(map[string]Node, len(coreItems))
	for _, core := range coreItems {
		item := nodeFromCore(core)
		outByID[item.NodeID] = item
	}

	for _, position := range positionItems {
		node := outByID[position.NodeID]
		applyNodePosition(&node, position)
		if node.NodeID == "" {
			node.NodeID = position.NodeID
		}
		outByID[position.NodeID] = node
	}

	for _, telemetry := range telemetryItems {
		node := outByID[telemetry.NodeID]
		applyNodeTelemetry(&node, telemetry)
		if node.NodeID == "" {
			node.NodeID = telemetry.NodeID
		}
		outByID[telemetry.NodeID] = node
	}

	out := make([]Node, 0, len(outByID))
	for _, item := range outByID {
		out = append(out, item)
	}

	return out
}

func nodeFromCore(core NodeCore) Node {
	node := Node{
		NodeID:          core.NodeID,
		LongName:        core.LongName,
		ShortName:       core.ShortName,
		PublicKey:       cloneNodePublicKey(core.PublicKey),
		Channel:         core.Channel,
		BoardModel:      core.BoardModel,
		FirmwareVersion: core.FirmwareVersion,
		Role:            core.Role,
		IsFavorite:      core.IsFavorite,
		IsUnmessageable: core.IsUnmessageable,
		LastHeardAt:     core.LastHeardAt,
		RSSI:            core.RSSI,
		SNR:             core.SNR,
		UpdatedAt:       core.UpdatedAt,
	}

	return node
}

func applyNodePosition(node *Node, position NodePosition) {
	if node == nil {
		return
	}
	if node.Channel == nil {
		node.Channel = position.Channel
	}
	node.Latitude = position.Latitude
	node.Longitude = position.Longitude
	node.Altitude = position.Altitude
	node.PositionPrecisionBits = position.PositionPrecisionBits
	if !position.PositionUpdatedAt.IsZero() {
		node.PositionUpdatedAt = position.PositionUpdatedAt
	}
}

func applyNodeTelemetry(node *Node, telemetry NodeTelemetry) {
	if node == nil {
		return
	}
	if node.Channel == nil {
		node.Channel = telemetry.Channel
	}
	node.BatteryLevel = telemetry.BatteryLevel
	node.Voltage = telemetry.Voltage
	node.UptimeSeconds = telemetry.UptimeSeconds
	node.ChannelUtilization = telemetry.ChannelUtilization
	node.AirUtilTx = telemetry.AirUtilTx
	node.Temperature = telemetry.Temperature
	node.Humidity = telemetry.Humidity
	node.Pressure = telemetry.Pressure
	node.SoilTemperature = telemetry.SoilTemperature
	node.SoilMoisture = telemetry.SoilMoisture
	node.GasResistance = telemetry.GasResistance
	node.Lux = telemetry.Lux
	node.UVLux = telemetry.UVLux
	node.Radiation = telemetry.Radiation
	node.AirQualityIndex = telemetry.AirQualityIndex
	node.PowerVoltage = telemetry.PowerVoltage
	node.PowerCurrent = telemetry.PowerCurrent
}
