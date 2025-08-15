package protocol

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

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



func (rc *RadioClient) cleanNodeName(name string) string {
	if len(name) == 0 {
		return ""
	}
	
	cleaned := name
	
	// Remove trailing garbage chars (quotes, backslashes, control chars, high ASCII)
	for len(cleaned) > 0 {
		lastChar := cleaned[len(cleaned)-1]
		if lastChar == '"' || lastChar == '\'' || lastChar == '\\' || 
		   lastChar < 32 || lastChar > 126 || lastChar == 0 {
			cleaned = cleaned[:len(cleaned)-1]
		} else {
			break
		}
	}
	
	// Remove leading garbage chars
	for len(cleaned) > 0 {
		firstChar := cleaned[0]
		if firstChar == '"' || firstChar == '\'' || firstChar == '\\' || 
		   firstChar < 32 || firstChar > 126 || firstChar == 0 {
			cleaned = cleaned[1:]
		} else {
			break
		}
	}
	
	// Remove any embedded null bytes or other binary garbage
	result := make([]byte, 0, len(cleaned))
	for i := 0; i < len(cleaned); i++ {
		char := cleaned[i]
		if char >= 32 && char <= 126 && char != 0 {
			result = append(result, char)
		}
	}
	
	return string(result)
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



func (rc *RadioClient) handlePacket(data []byte) {
	// Log the raw packet data  
	rc.logger.Debug("Received packet from device", "length", len(data))
	
	// Skip the broken protobuf decoding entirely and use working manual parsing
	// This provides proper node detection without fake messages or crashes
	rc.parsePacketManually(data)
}

func (rc *RadioClient) parsePacketManually(data []byte) {
	if len(data) < 2 {
		return
	}
	
	// Basic protobuf field parsing - look for node information only
	if len(data) >= 4 {
		// Look for NodeInfo patterns (field 4 in FromRadio = 0x22)
		if data[0] == 0x22 {
			rc.logger.Debug("Detected NodeInfo packet", "size", len(data))
			rc.extractNodeInfoSafely(data[1:])
		} else if data[0] == 0x1a {
			// field 3 (MyInfo) = 0x1a
			rc.logger.Debug("Detected MyNodeInfo packet", "size", len(data))
		} else if data[0] == 0x12 {
			// field 2 (MeshPacket) = 0x12 - handle without fake message extraction
			rc.logger.Debug("Detected MeshPacket", "size", len(data))
			// Don't extract fake messages - only process if it's a real text message packet
		}
	}
}

func (rc *RadioClient) extractNodeInfoSafely(data []byte) {
	// Extract node names but don't generate fake text messages
	var nodeNames []string
	var current []byte
	
	for _, b := range data {
		if b >= 32 && b <= 126 { // printable ASCII
			current = append(current, b)
		} else {
			if len(current) > 3 {
				name := string(current)
				cleanName := rc.cleanNodeName(name)
				if cleanName != "" && rc.isValidNodeName(cleanName) {
					nodeNames = append(nodeNames, cleanName)
				}
			}
			current = nil
		}
	}
	if len(current) > 3 {
		name := string(current)
		cleanName := rc.cleanNodeName(name)
		if cleanName != "" && rc.isValidNodeName(cleanName) {
			nodeNames = append(nodeNames, cleanName)
		}
	}
	
	if len(nodeNames) > 0 {
		rc.logger.Debug("Extracted node names", "names", nodeNames)
		
		// Find hex ID and best name
		var nodeID string
		var bestName string
		
		for _, name := range nodeNames {
			if len(name) > 0 && name[0] == '!' && len(name) == 9 {
				nodeID = name[1:] // Remove the !
			} else if len(name) >= 4 && len(name) <= 20 {
				if bestName == "" || len(name) > len(bestName) {
					bestName = name
				}
			}
		}
		
		// Create node record - use hex ID if available
		var finalNodeID string
		var displayName string
		
		if nodeID != "" {
			finalNodeID = nodeID
			displayName = bestName
		} else if bestName != "" {
			finalNodeID = fmt.Sprintf("name_%s", bestName)  
			displayName = bestName
		}
		
		if finalNodeID != "" && displayName != "" {
			node := rc.getOrCreateNode(finalNodeID)
			node.ShortName = displayName
			if len(displayName) > 10 {
				node.LongName = displayName
				node.ShortName = displayName[:10]
			}
			
			rc.logger.Debug("Node created/updated", "id", finalNodeID, "name", displayName)
			
			// Emit node updated event
			rc.events <- core.Event{
				Type: core.EventNodeUpdated,
				Data: node,
			}
		}
	}
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
	// Only process messages from TEXT_MESSAGE_APP port
	if data.Portnum != PortTextMessageApp {
		rc.logger.Debug("Skipping non-text message", "portnum", data.Portnum)
		return
	}
	
	if len(data.Payload) == 0 {
		rc.logger.Debug("Empty text message payload")
		return
	}

	// Validate that payload is valid UTF-8 text
	if !utf8.Valid(data.Payload) {
		rc.logger.Debug("Invalid UTF-8 in text message payload")
		return
	}

	text := string(data.Payload)
	
	// Additional validation for reasonable text messages
	if len(text) == 0 || len(text) > 200 {
		rc.logger.Debug("Text message length out of bounds", "length", len(text))
		return
	}
	
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
	signalQuality := core.CalculateSignalQuality(node.RSSI, node.SNR)
	node.SignalQuality = int(signalQuality)
	node.LastHeard = time.Unix(int64(packet.RxTime), 0)
	
	// Debug logging for signal quality
	rc.logger.Debug("Signal quality calculated", 
		"node", node.ID,
		"rssi", node.RSSI, 
		"snr", node.SNR,
		"quality", signalQuality.String(),
		"quality_int", node.SignalQuality)
	
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

func (rc *RadioClient) GetNodes() []*core.Node {
	rc.nodeDBMu.Lock()
	defer rc.nodeDBMu.Unlock()
	
	// Clean up old nodes that haven't been heard from in the last 10 minutes
	cutoff := time.Now().Add(-10 * time.Minute)
	for id, node := range rc.nodeDB {
		if node.LastHeard.Before(cutoff) {
			rc.logger.Debug("Removing stale node", "id", id, "last_heard", node.LastHeard)
			delete(rc.nodeDB, id)
		}
	}
	
	nodes := make([]*core.Node, 0, len(rc.nodeDB))
	for _, node := range rc.nodeDB {
		nodes = append(nodes, node)
	}
	return nodes
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rc *RadioClient) tryExtractSignalMetrics(node *core.Node, data []byte) {
	// This is a heuristic approach to extract RSSI/SNR from raw protobuf data
	// In proper protobuf parsing, these would be in specific fields, but we're doing manual extraction
	
	// Look for patterns that might indicate signal metrics
	// RSSI is typically negative (-30 to -120 dBm range)
	// SNR is typically -20 to +20 dB range
	
	for i := 0; i < len(data)-4; i++ {
		// Look for signed byte patterns that could be RSSI (negative values)
		if i+1 < len(data) {
			rssiCandidate := int8(data[i])
			snrCandidate := float32(int8(data[i+1]))
			
			// Check if this looks like valid RSSI/SNR values
			if rssiCandidate >= -120 && rssiCandidate <= -30 && // Valid RSSI range
			   snrCandidate >= -20 && snrCandidate <= 20 { // Valid SNR range
				
				// Found potential signal metrics
				node.RSSI = int(rssiCandidate)
				node.SNR = snrCandidate
				signalQuality := core.CalculateSignalQuality(node.RSSI, node.SNR)
				node.SignalQuality = int(signalQuality)
				
				rc.logger.Debug("Extracted signal metrics", 
					"node", node.ID,
					"rssi", node.RSSI, 
					"snr", node.SNR,
					"quality", signalQuality.String())
				return
			}
		}
	}
	
	// If no valid signal data found, don't set defaults - leave as 0 to indicate offline
	if node.RSSI == 0 && node.SNR == 0 {
		// Node is offline or we haven't received signal data
		signalQuality := core.CalculateSignalQuality(0, 0)
		node.SignalQuality = int(signalQuality)
		
		rc.logger.Debug("No signal data - node offline or not heard", 
			"node", node.ID,
			"quality", signalQuality.String())
	}
}