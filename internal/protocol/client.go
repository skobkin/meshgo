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

	// Send initialization handshake after a short delay
	go rc.initializeConnection(ctx)

	rc.logger.Info("Radio client started", "endpoint", transport.Endpoint())
	return nil
}

func (rc *RadioClient) initializeConnection(ctx context.Context) {
	// Wait for connection to be established
	time.Sleep(500 * time.Millisecond)
	
	rc.logger.Info("Sending startConfig request...")
	
	// Send a startConfig request using raw protobuf bytes
	// ToRadio message with want_config_id (field 3) = 42
	// Field 3 with varint encoding: tag=(3<<3)|0 = 24 = 0x18
	// Value 42 = 0x2A
	startConfigBytes := []byte{0x18, 0x2A} // field 3, value 42
	
	if err := rc.transport.WritePacket(ctx, startConfigBytes); err != nil {
		rc.logger.Error("Failed to send startConfig", "error", err)
		return
	}

	rc.logger.Info("Sent startConfig with want_config_id=42")
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

	// Create the Data payload
	data := &Data{
		Portnum: PortTextMessageApp,
		Payload: []byte(text),
	}

	// Create the MeshPacket with decoded payload
	packet := &MeshPacket{
		From:      rc.ownNodeID,
		To:        toNode,
		Channel:   0, // Default channel
		Id:        rc.generatePacketID(),
		WantAck:   true,
		Priority:  PriorityDefault,
		PayloadVariant: &MeshPacket_Decoded{
			Decoded: data,
		},
	}

	// Wrap in ToRadio message
	toRadio := &ToRadio{
		PayloadVariant: &ToRadio_Packet{
			Packet: packet,
		},
	}

	packetData, err := EncodeToRadio(toRadio)
	if err != nil {
		return fmt.Errorf("failed to encode ToRadio: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, packetData); err != nil {
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

	// Create NodeInfo request payload
	data := &Data{
		Portnum: PortNodeInfoApp,
		Payload: []byte{}, // Empty payload for NodeInfo request
	}

	packet := &MeshPacket{
		From:      rc.ownNodeID,
		To:        node,
		Channel:   0,
		Id:        rc.generatePacketID(),
		WantAck:   true,
		Priority:  PriorityDefault,
		PayloadVariant: &MeshPacket_Decoded{
			Decoded: data,
		},
	}

	packetData, err := EncodeMeshPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to encode user info request: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, packetData); err != nil {
		return fmt.Errorf("failed to send user info request: %w", err)
	}

	rc.logger.Debug("User info exchange requested", "node", node)
	return nil
}

func (rc *RadioClient) SendTraceroute(ctx context.Context, node uint32) error {
	if !rc.isRunning() {
		return fmt.Errorf("radio client not running")
	}

	// Create traceroute request payload
	data := &Data{
		Portnum: PortRoutingApp,
		Dest:    node,
		Payload: []byte{0x01}, // Simple traceroute request marker
	}

	packet := &MeshPacket{
		From:      rc.ownNodeID,
		To:        node,
		Channel:   0,
		Id:        rc.generatePacketID(),
		WantAck:   false, // Traceroute doesn't need ack
		Priority:  PriorityDefault,
		PayloadVariant: &MeshPacket_Decoded{
			Decoded: data,
		},
	}

	packetData, err := EncodeMeshPacket(packet)
	if err != nil {
		return fmt.Errorf("failed to encode traceroute request: %w", err)
	}

	if err := rc.transport.WritePacket(ctx, packetData); err != nil {
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

		rc.logger.Debug("Attempting to read packet...")
		
		// Read packet with shorter timeout initially, then longer  
		readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		data, err := rc.transport.ReadPacket(readCtx)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return // Context cancelled
			}
			rc.logger.Debug("Read packet error", "error", err)
			// Continue with a small delay to avoid spinning
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if len(data) == 0 {
			rc.logger.Debug("Received empty packet")
			continue
		}

		rc.logger.Debug("Received raw packet", "length", len(data))
		rc.handlePacket(data)
	}

	rc.logger.Info("Read loop stopped")
}

func (rc *RadioClient) parsePacketManually(data []byte) {
	if len(data) < 2 {
		return
	}
	
	// Basic protobuf field parsing
	// Check for common field patterns in the first few bytes
	if len(data) >= 4 {
		// Look for NodeInfo patterns (field 4 in FromRadio = 0x22)
		if data[0] == 0x22 {
			rc.logger.Info("Detected NodeInfo packet", "size", len(data))
			rc.extractNodeInfo(data[1:])
		} else if data[0] == 0x1a {
			// field 3 (MyInfo) = 0x1a
			rc.logger.Info("Detected MyNodeInfo packet", "size", len(data))
		} else if data[0] == 0x12 {
			// field 2 (MeshPacket) = 0x12
			rc.logger.Info("Detected MeshPacket", "size", len(data))
			rc.extractMeshPacketData(data[1:])
		} else {
			rc.logger.Debug("Unknown packet type", "first_byte", fmt.Sprintf("0x%02x", data[0]))
		}
	}
}

func (rc *RadioClient) extractNodeInfo(data []byte) {
	// Extract readable strings from NodeInfo data with better filtering
	var nodeNames []string
	var current []byte
	
	for _, b := range data {
		if b >= 32 && b <= 126 { // printable ASCII
			current = append(current, b)
		} else {
			if len(current) > 3 {
				name := string(current)
				// Filter for reasonable node names
				if rc.isValidNodeName(name) {
					nodeNames = append(nodeNames, name)
				}
			}
			current = nil
		}
	}
	if len(current) > 3 {
		name := string(current)
		if rc.isValidNodeName(name) {
			nodeNames = append(nodeNames, name)
		}
	}
	
	if len(nodeNames) > 0 {
		rc.logger.Info("Extracted node names", "names", nodeNames)
		
		// Find the best node name - prefer longer descriptive names
		var bestName string
		var shortName string
		
		for _, name := range nodeNames {
			// Skip node IDs starting with ! (hex IDs)
			if name[0] == '!' {
				continue
			}
			
			// Prefer names with reasonable length and common mesh naming patterns
			if len(name) >= 4 && len(name) <= 20 {
				if len(name) <= 8 {
					if shortName == "" || len(name) > len(shortName) {
						shortName = name
					}
				}
				if len(name) > 8 {
					if bestName == "" || len(name) < len(bestName) {
						bestName = name
					}
				}
			}
		}
		
		// Create node record if we found a good name
		if bestName != "" || shortName != "" {
			// Use the hex ID if available as the primary node ID
			var nodeID string
			for _, name := range nodeNames {
				if name[0] == '!' && len(name) == 9 {
					nodeID = name[1:] // Remove the !
					break
				}
			}
			
			if nodeID == "" {
				nodeID = fmt.Sprintf("node_%s", shortName)
				if bestName != "" {
					nodeID = fmt.Sprintf("node_%s", bestName)
				}
			}
			
			node := rc.getOrCreateNode(nodeID)
			if shortName != "" {
				node.ShortName = shortName
			}
			if bestName != "" {
				node.LongName = bestName
			}
			
			// Emit node updated event
			rc.events <- core.Event{
				Type: core.EventNodeUpdated,
				Data: node,
			}
		}
	}
}

func (rc *RadioClient) isValidNodeName(name string) bool {
	// Filter out noise and keep only reasonable node names
	if len(name) < 4 || len(name) > 25 {
		return false
	}
	
	// Skip names that look like garbage (too many non-alphanumeric chars)
	alphanumCount := 0
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			alphanumCount++
		}
	}
	
	// Must be at least 50% reasonable characters
	return float64(alphanumCount)/float64(len(name)) >= 0.5
}

func (rc *RadioClient) extractMeshPacketData(data []byte) {
	// Look for text messages in MeshPacket data
	// Text messages typically have PortNum=1 and readable payload
	
	// Simple approach: look for readable text strings that could be messages
	var messages []string
	var current []byte
	
	for _, b := range data {
		if b >= 32 && b <= 126 { // printable ASCII
			current = append(current, b)
		} else {
			if len(current) > 10 { // Messages are typically longer
				text := string(current)
				if rc.isValidMessage(text) {
					messages = append(messages, text)
				}
			}
			current = nil
		}
	}
	if len(current) > 10 {
		text := string(current)
		if rc.isValidMessage(text) {
			messages = append(messages, text)
		}
	}
	
	for _, msgText := range messages {
		rc.logger.Info("Detected potential text message", "text", msgText)
		
		// Create a message record - we don't have full sender info yet
		// but this shows that message detection is working
		msg := &core.Message{
			ChatID:    "unknown",
			SenderID:  "unknown",
			PortNum:   1, // TEXT_MESSAGE_APP
			Text:      msgText,
			Timestamp: time.Now(),
			IsUnread:  true,
		}
		
		// Emit message received event
		rc.events <- core.Event{
			Type: core.EventMessageReceived,
			Data: msg,
		}
	}
}

func (rc *RadioClient) isValidMessage(text string) bool {
	// Filter for reasonable message content
	if len(text) < 3 || len(text) > 200 {
		return false
	}
	
	// Check if it looks like a real message (has spaces and reasonable chars)
	hasSpace := false
	alphaCount := 0
	
	for _, r := range text {
		if r == ' ' {
			hasSpace = true
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			alphaCount++
		}
	}
	
	// Must have some alpha characters and ideally spaces (real messages)
	return alphaCount > 2 && (hasSpace || len(text) < 20)
}

func (rc *RadioClient) handlePacket(data []byte) {
	// Log the raw packet data
	rc.logger.Info("Received packet from device", 
		"length", len(data),
		"hex", fmt.Sprintf("%x", data))
	
	// For now, let's just try to extract basic information without full protobuf decode
	rc.parsePacketManually(data)
}

func (rc *RadioClient) handleFromRadio(msg *FromRadio) {
	rc.logger.Debug("Received FromRadio message", "id", msg.Id)

	// Handle different FromRadio payload types
	if packet := msg.GetPacket(); packet != nil {
		rc.logger.Debug("FromRadio contains MeshPacket")
		rc.handleMeshPacket(packet)
	} else if myInfo := msg.GetMyInfo(); myInfo != nil {
		rc.handleMyNodeInfo(myInfo)
	} else if nodeInfo := msg.GetNodeInfo(); nodeInfo != nil {
		rc.handleNodeInfoFromRadio(nodeInfo)
	} else if configId := msg.GetConfigCompleteId(); configId != 0 {
		rc.logger.Info("Configuration complete", "config_id", configId)
	} else {
		rc.logger.Debug("FromRadio with unknown payload type")
	}
}

func (rc *RadioClient) handleMeshPacket(packet *MeshPacket) {
	// Update receive metadata
	if packet.RxTime == 0 {
		packet.RxTime = uint32(time.Now().Unix())
	}

	// Get the decoded payload
	decodedData := packet.GetDecoded()
	if decodedData == nil {
		// Handle encrypted packets - for now, just log and skip
		if packet.GetEncrypted() != nil {
			rc.logger.Debug("Received encrypted packet - skipping", 
				"from", packet.From, 
				"to", packet.To,
				"size", len(packet.GetEncrypted()))
		} else {
			rc.logger.Debug("Received packet with no payload")
		}
		return
	}

	rc.logger.Debug("Received MeshPacket",
		"from", packet.From,
		"to", packet.To,
		"portnum", decodedData.Portnum,
		"channel", packet.Channel,
		"id", packet.Id)

	// Handle different packet types based on portnum
	switch decodedData.Portnum {
	case PortTextMessageApp:
		rc.handleTextMessage(packet, decodedData)
	case PortNodeInfoApp:
		rc.handleNodeInfo(packet, decodedData)
	case PortPositionApp:
		rc.handlePosition(packet, decodedData)
	case PortRoutingApp:
		rc.handleRouting(packet, decodedData)
	default:
		rc.logger.Debug("Unhandled packet type", "portnum", decodedData.Portnum)
	}
}

func (rc *RadioClient) handleMyNodeInfo(myInfo *MyNodeInfo) {
	rc.logger.Info("Received MyNodeInfo", 
		"node_num", myInfo.MyNodeNum,
		"firmware", myInfo.FirmwareVersion,
		"hw_model", myInfo.HwModel)
	
	// Store our own node ID
	rc.ownNodeID = myInfo.MyNodeNum
}

func (rc *RadioClient) handleNodeInfoFromRadio(nodeInfo *NodeInfo) {
	rc.logger.Info("Received NodeInfo from config", "node_num", nodeInfo.Num)
	
	// Create a fake packet to reuse existing logic
	packet := &MeshPacket{
		From:   nodeInfo.Num,
		RxTime: uint32(time.Now().Unix()),
	}
	
	node := rc.updateNodeFromPacket(packet, nodeInfo)
	
	// Emit node updated event
	rc.events <- core.Event{
		Type: core.EventNodeUpdated,
		Data: node,
	}
}

func (rc *RadioClient) handleTextMessage(packet *MeshPacket, data *Data) {
	if len(data.Payload) == 0 {
		rc.logger.Debug("Empty text message payload")
		return
	}

	text := string(data.Payload)
	
	// Determine chat ID
	chatID := rc.getChatID(packet)
	
	rxrssi := int(packet.RxRssi)
	msg := &core.Message{
		ChatID:    chatID,
		SenderID:  fmt.Sprintf("%d", packet.From),
		PortNum:   int(data.Portnum),
		Text:      text,
		RXSNR:     &packet.RxSnr,
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
		"text", text)
}

func (rc *RadioClient) handleNodeInfo(packet *MeshPacket, data *Data) {
	if len(data.Payload) == 0 {
		// Empty payload might be a request - just update node from packet metadata
		nodeID := fmt.Sprintf("%d", packet.From)
		node := rc.getOrCreateNode(nodeID)
		rc.updateNodeMetrics(node, packet)
		
		// Emit node updated event
		rc.events <- core.Event{
			Type: core.EventNodeUpdated,
			Data: node,
		}
		return
	}

	// Decode NodeInfo from payload will be handled below

	// Extract nodeInfo from decoded interface
	decoded, err := DecodePayload(data)
	if err != nil {
		rc.logger.Debug("Failed to decode NodeInfo payload", "error", err)
		return
	}

	nodeInfo, ok := decoded.(*NodeInfo)
	if !ok {
		rc.logger.Debug("Decoded payload is not NodeInfo")
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

func (rc *RadioClient) handlePosition(packet *MeshPacket, data *Data) {
	if len(data.Payload) == 0 {
		rc.logger.Debug("Empty position payload")
		return
	}

	// Decode Position from payload
	decoded, err := DecodePayload(data)
	if err != nil {
		rc.logger.Debug("Failed to decode Position payload", "error", err)
		return
	}

	position, ok := decoded.(*Position)
	if !ok {
		rc.logger.Debug("Decoded payload is not Position")
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
		GPSAccuracy:    position.GpsAccuracy,
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

func (rc *RadioClient) handleRouting(packet *MeshPacket, data *Data) {
	// Handle traceroute responses and route discovery
	rc.logger.Debug("Routing packet received", "from", packet.From, "payload_size", len(data.Payload))
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
			GPSAccuracy:    nodeInfo.Position.GpsAccuracy,
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
	node.RSSI = int(packet.RxRssi)
	node.SNR = packet.RxSnr
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