package protocol

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"meshgo/internal/core"
)

type RadioClient struct {
	transport   core.Transport
	events      chan core.Event
	nodeDB      map[string]*core.Node
	nodeDBMu    sync.RWMutex
	running     bool
	runningMu   sync.RWMutex
	logger      *slog.Logger
	ownNodeID   uint32
}

func NewRadioClient(logger *slog.Logger) *RadioClient {
	return &RadioClient{
		events:  make(chan core.Event, 100),
		nodeDB:  make(map[string]*core.Node),
		logger:  logger,
	}
}

func (rc *RadioClient) Start(ctx context.Context, transport core.Transport) error {
	rc.runningMu.Lock()
	defer rc.runningMu.Unlock()

	if rc.running {
		return fmt.Errorf("radio client already running")
	}

	rc.transport = transport
	rc.running = true

	// Start read loop
	go rc.readLoop(ctx)

	rc.logger.Info("Radio client started", "endpoint", transport.Endpoint())
	return nil
}

func (rc *RadioClient) Stop() error {
	rc.runningMu.Lock()
	defer rc.runningMu.Unlock()

	if !rc.running {
		return nil
	}

	rc.running = false
	close(rc.events)

	rc.logger.Info("Radio client stopped")
	return nil
}

func (rc *RadioClient) Events() <-chan core.Event {
	return rc.events
}

func (rc *RadioClient) SendText(ctx context.Context, chatID string, toNode uint32, text string) error {
	if !rc.isRunning() {
		return fmt.Errorf("radio client not running")
	}

	packet := &MeshPacket{
		From:     rc.ownNodeID,
		To:       toNode,
		Channel:  0, // Default channel
		ID:       rc.generatePacketID(),
		PortNum:  PortTextMessageApp,
		Payload:  EncodeTextMessage(text),
		WantAck:  true,
		Priority: Default,
	}

	data, err := EncodeMeshPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to encode packet: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, data); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	// Create message record for local storage
	msg := &core.Message{
		ChatID:    chatID,
		SenderID:  fmt.Sprintf("%d", rc.ownNodeID),
		PortNum:   int(PortTextMessageApp),
		Text:      text,
		Timestamp: time.Now(),
		IsUnread:  false, // Own messages are not unread
	}

	// Emit message sent event
	rc.events <- core.Event{
		Type: core.EventMessageReceived,
		Data: msg,
	}

	rc.logger.Debug("Text message sent", 
		"to", toNode, 
		"chat", chatID, 
		"length", len(text))

	return nil
}

func (rc *RadioClient) SendExchangeUserInfo(ctx context.Context, node uint32) error {
	if !rc.isRunning() {
		return fmt.Errorf("radio client not running")
	}

	packet := &MeshPacket{
		From:     rc.ownNodeID,
		To:       node,
		Channel:  0,
		ID:       rc.generatePacketID(),
		PortNum:  PortNodeInfoApp,
		Payload:  EncodeNodeInfoRequest(),
		WantAck:  true,
		Priority: Default,
	}

	data, err := EncodeMeshPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to encode user info request: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, data); err != nil {
		return fmt.Errorf("failed to send user info request: %w", err)
	}

	rc.logger.Debug("User info exchange requested", "node", node)
	return nil
}

func (rc *RadioClient) SendTraceroute(ctx context.Context, node uint32) error {
	if !rc.isRunning() {
		return fmt.Errorf("radio client not running")
	}

	packet := &MeshPacket{
		From:     rc.ownNodeID,
		To:       node,
		Channel:  0,
		ID:       rc.generatePacketID(),
		PortNum:  PortRoutingApp,
		Payload:  EncodeTracerouteRequest(node),
		WantAck:  false, // Traceroute doesn't need ack
		Priority: Default,
	}

	data, err := EncodeMeshPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to encode traceroute request: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, data); err != nil {
		return fmt.Errorf("failed to send traceroute request: %w", err)
	}

	rc.logger.Debug("Traceroute requested", "destination", node)
	return nil
}

func (rc *RadioClient) readLoop(ctx context.Context) {
	rc.logger.Info("Starting packet read loop")
	
	for rc.isRunning() {
		select {
		case <-ctx.Done():
			rc.logger.Info("Read loop cancelled by context")
			return
		default:
		}

		if !rc.transport.IsConnected() {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Read packet with timeout
		readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		data, err := rc.transport.ReadPacket(readCtx)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return // Context cancelled
			}
			rc.logger.Debug("Read packet error", "error", err)
			continue
		}

		if len(data) == 0 {
			continue
		}

		rc.handlePacket(data)
	}

	rc.logger.Info("Read loop stopped")
}

func (rc *RadioClient) handlePacket(data []byte) {
	packet, err := DecodeMeshPacket(data)
	if err != nil {
		rc.logger.Debug("Failed to decode packet", "error", err, "length", len(data))
		return
	}

	rc.logger.Debug("Received packet",
		"from", packet.From,
		"to", packet.To,
		"port", packet.PortNum,
		"channel", packet.Channel,
		"id", packet.ID)

	// Update receive metadata
	if packet.RxTime == 0 {
		packet.RxTime = uint32(time.Now().Unix())
	}

	// Handle different packet types
	switch packet.PortNum {
	case PortTextMessageApp:
		rc.handleTextMessage(packet)
	case PortNodeInfoApp:
		rc.handleNodeInfo(packet)
	case PortPositionApp:
		rc.handlePosition(packet)
	case PortRoutingApp:
		rc.handleRouting(packet)
	default:
		rc.logger.Debug("Unhandled packet type", "port", packet.PortNum)
	}
}

func (rc *RadioClient) handleTextMessage(packet *MeshPacket) {
	if packet.Decoded == nil {
		rc.logger.Debug("No decoded text message")
		return
	}

	textMsg, ok := packet.Decoded.(*TextMessage)
	if !ok {
		rc.logger.Debug("Invalid text message type")
		return
	}

	// Determine chat ID
	chatID := rc.getChatID(packet)
	
	rxrssi := int(packet.RxRSSI)
	msg := &core.Message{
		ChatID:    chatID,
		SenderID:  fmt.Sprintf("%d", packet.From),
		PortNum:   int(packet.PortNum),
		Text:      textMsg.Text,
		RXSNR:     &packet.RxSNR,
		RXRSSI:    &rxrssi,
		Timestamp: time.Unix(int64(packet.RxTime), 0),
		IsUnread:  true,
	}

	// Emit message received event
	rc.events <- core.Event{
		Type: core.EventMessageReceived,
		Data: msg,
	}

	rc.logger.Info("Text message received",
		"from", packet.From,
		"chat", chatID,
		"text", textMsg.Text)
}

func (rc *RadioClient) handleNodeInfo(packet *MeshPacket) {
	if packet.Decoded == nil {
		return
	}

	nodeInfo, ok := packet.Decoded.(*NodeInfo)
	if !ok {
		return
	}

	node := rc.updateNodeFromPacket(packet, nodeInfo)
	
	// Emit node updated event
	rc.events <- core.Event{
		Type: core.EventNodeUpdated,
		Data: node,
	}

	rc.logger.Info("Node info received", "node", node.ID)
}

func (rc *RadioClient) handlePosition(packet *MeshPacket) {
	if packet.Decoded == nil {
		return
	}

	position, ok := packet.Decoded.(*Position)
	if !ok {
		return
	}

	nodeID := fmt.Sprintf("%d", packet.From)
	node := rc.getOrCreateNode(nodeID)
	
	// Update position
	node.Position = &core.Position{
		LatitudeI:      position.LatitudeI,
		LongitudeI:     position.LongitudeI,
		Altitude:       position.Altitude,
		Time:           position.Time,
		LocationSource: int(position.LocationSource),
		AltitudeSource: int(position.AltitudeSource),
		GPSAccuracy:    position.GPSAccuracy,
	}
	
	rc.updateNodeMetrics(node, packet)
	
	// Emit node updated event
	rc.events <- core.Event{
		Type: core.EventNodeUpdated,
		Data: node,
	}

	rc.logger.Debug("Position received", "node", nodeID, 
		"lat", position.Latitude(), "lon", position.Longitude())
}

func (rc *RadioClient) handleRouting(packet *MeshPacket) {
	if packet.Decoded == nil {
		return
	}

	// Handle traceroute responses and route discovery
	rc.logger.Debug("Routing packet received", "from", packet.From)
}

func (rc *RadioClient) updateNodeFromPacket(packet *MeshPacket, nodeInfo *NodeInfo) *core.Node {
	nodeID := fmt.Sprintf("%d", packet.From)
	node := rc.getOrCreateNode(nodeID)

	// Update from NodeInfo if available
	if nodeInfo.User != nil {
		node.ShortName = nodeInfo.User.ShortName
		node.LongName = nodeInfo.User.LongName
	}

	if nodeInfo.Position != nil {
		node.Position = &core.Position{
			LatitudeI:      nodeInfo.Position.LatitudeI,
			LongitudeI:     nodeInfo.Position.LongitudeI,
			Altitude:       nodeInfo.Position.Altitude,
			Time:           nodeInfo.Position.Time,
			LocationSource: int(nodeInfo.Position.LocationSource),
			AltitudeSource: int(nodeInfo.Position.AltitudeSource),
			GPSAccuracy:    nodeInfo.Position.GPSAccuracy,
		}
	}

	if nodeInfo.DeviceMetrics != nil {
		node.DeviceMetrics = &core.DeviceMetrics{
			BatteryLevel: nodeInfo.DeviceMetrics.BatteryLevel,
			Voltage:      nodeInfo.DeviceMetrics.Voltage,
		}
		
		// Update battery info
		if nodeInfo.DeviceMetrics.BatteryLevel <= 100 {
			level := int(nodeInfo.DeviceMetrics.BatteryLevel)
			node.BatteryLevel = &level
			charging := nodeInfo.DeviceMetrics.IsCharging()
			node.IsCharging = &charging
		}
	}

	rc.updateNodeMetrics(node, packet)
	return node
}

func (rc *RadioClient) updateNodeMetrics(node *core.Node, packet *MeshPacket) {
	// Update signal metrics
	node.RSSI = int(packet.RxRSSI)
	node.SNR = packet.RxSNR
	node.SignalQuality = int(core.CalculateSignalQuality(node.RSSI, node.SNR))
	node.LastHeard = time.Unix(int64(packet.RxTime), 0)
	
	// Store in node database
	rc.nodeDBMu.Lock()
	rc.nodeDB[node.ID] = node
	rc.nodeDBMu.Unlock()
}

func (rc *RadioClient) getOrCreateNode(nodeID string) *core.Node {
	rc.nodeDBMu.RLock()
	node, exists := rc.nodeDB[nodeID]
	rc.nodeDBMu.RUnlock()

	if exists {
		return node
	}

	// Create new node
	node = &core.Node{
		ID:        nodeID,
		LastHeard: time.Now(),
	}

	rc.nodeDBMu.Lock()
	rc.nodeDB[nodeID] = node
	rc.nodeDBMu.Unlock()

	return node
}

func (rc *RadioClient) getChatID(packet *MeshPacket) string {
	if IsChannelMessage(packet) {
		return fmt.Sprintf("channel_%d", packet.Channel)
	}
	return fmt.Sprintf("%d", packet.From)
}

func (rc *RadioClient) generatePacketID() uint32 {
	// Simple packet ID generation - in real app would be more sophisticated
	return uint32(time.Now().UnixNano() & 0xFFFFFFFF)
}

func (rc *RadioClient) isRunning() bool {
	rc.runningMu.RLock()
	defer rc.runningMu.RUnlock()
	return rc.running
}