package protocol

// Protocol implementation inspired by and improved based on research of:
// - lmatte7/meshtastic-go: https://github.com/lmatte7/meshtastic-go
// - Official Meshtastic protocol documentation
// Enhanced with proper null pointer protection and hybrid parsing approach

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"

	"meshgo/internal/core"
	"meshgo/internal/protocol/gomeshproto"
)

type RadioClient struct {
	transport   core.Transport
	events      chan core.Event
	nodeDB      map[string]*core.Node
	nodeDBMu    sync.RWMutex
	channelDB   map[int32]*gomeshproto.Channel // Channel storage by index
	channelDBMu sync.RWMutex
	running     bool
	runningMu   sync.RWMutex
	logger      *slog.Logger
	ownNodeID   uint32
}

func NewRadioClient(logger *slog.Logger) *RadioClient {
	return &RadioClient{
		events:    make(chan core.Event, 100),
		nodeDB:    make(map[string]*core.Node),
		channelDB: make(map[int32]*gomeshproto.Channel),
		logger:    logger,
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

	// Send a startConfig request using proper gomeshproto
	toRadio := &gomeshproto.ToRadio{
		PayloadVariant: &gomeshproto.ToRadio_WantConfigId{
			WantConfigId: 42,
		},
	}

	startConfigBytes, err := proto.Marshal(toRadio)
	if err != nil {
		rc.logger.Error("Failed to marshal startConfig", "error", err)
		return
	}

	if err := rc.transport.WritePacket(ctx, startConfigBytes); err != nil {
		rc.logger.Error("Failed to send startConfig", "error", err)
		return
	}

	rc.logger.Info("Sent startConfig with want_config_id=42")

	// Request channels after a delay to allow config to be received
	go func() {
		time.Sleep(2 * time.Second)
		rc.requestAllChannels(ctx)
	}()
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

	// Create the Data payload using gomeshproto
	data := &gomeshproto.Data{
		Portnum: gomeshproto.PortNum_TEXT_MESSAGE_APP,
		Payload: []byte(text),
	}

	// Create the MeshPacket with decoded payload using gomeshproto
	packet := &gomeshproto.MeshPacket{
		From:     rc.ownNodeID,
		To:       toNode,
		Channel:  0, // Default channel
		Id:       rc.generatePacketID(),
		WantAck:  true,
		Priority: gomeshproto.MeshPacket_DEFAULT,
		PayloadVariant: &gomeshproto.MeshPacket_Decoded{
			Decoded: data,
		},
	}

	// Wrap in ToRadio message using gomeshproto
	toRadio := &gomeshproto.ToRadio{
		PayloadVariant: &gomeshproto.ToRadio_Packet{
			Packet: packet,
		},
	}

	packetData, err := proto.Marshal(toRadio)
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
		PortNum:   int(gomeshproto.PortNum_TEXT_MESSAGE_APP),
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
		From:     rc.ownNodeID,
		To:       node,
		Channel:  0,
		Id:       rc.generatePacketID(),
		WantAck:  true,
		Priority: PriorityDefault,
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
		From:     rc.ownNodeID,
		To:       node,
		Channel:  0,
		Id:       rc.generatePacketID(),
		WantAck:  false, // Traceroute doesn't need ack
		Priority: PriorityDefault,
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

		// Only log every 10th read attempt to reduce spam
		if time.Now().Unix()%10 == 0 {
			rc.logger.Debug("Waiting for Meshtastic packets...")
		}

		// Use longer timeout for Meshtastic devices - they don't send packets frequently
		// Meshtastic devices may only send data every 30+ seconds depending on configuration
		readCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
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

	// Try proper protobuf decoding first with null pointer protection
	// Based on lmatte7/meshtastic-go approach - use official protobuf parsing but safely
	if err := rc.handleFromRadioSafely(data); err != nil {
		rc.logger.Debug("Protobuf decode failed, falling back to manual parsing", "error", err)
		// Only use manual parsing as fallback for partial/corrupted packets
		rc.parsePacketManually(data)
	}
}

func (rc *RadioClient) handleFromRadioSafely(data []byte) error {
	// Use proper protobuf decoding with gomeshproto from lmatte7/goMesh
	// These are properly generated protobuf definitions that work correctly
	if len(data) == 0 {
		return fmt.Errorf("empty packet data")
	}

	rc.logger.Debug("Parsing FromRadio packet", "data_length", len(data))

	// Decode using proper gomeshproto definitions
	fromRadio := &gomeshproto.FromRadio{}
	err := proto.Unmarshal(data, fromRadio)
	if err != nil {
		return fmt.Errorf("failed to decode FromRadio with gomeshproto: %w", err)
	}

	// Process the FromRadio message safely
	rc.handleFromRadioWithGomeshproto(fromRadio)
	return nil
}

func (rc *RadioClient) handleFromRadioWithGomeshproto(msg *gomeshproto.FromRadio) {
	// Handle FromRadio using proper gomeshproto definitions
	if msg == nil {
		rc.logger.Debug("FromRadio message is nil")
		return
	}

	rc.logger.Debug("Received FromRadio message", "id", msg.Id)

	// Handle different FromRadio payload types with null checks
	if packet := msg.GetPacket(); packet != nil {
		rc.logger.Debug("FromRadio contains MeshPacket")
		rc.handleMeshPacketGomeshproto(packet)
	} else if myInfo := msg.GetMyInfo(); myInfo != nil {
		rc.handleMyNodeInfoGomeshproto(myInfo)
	} else if nodeInfo := msg.GetNodeInfo(); nodeInfo != nil {
		rc.handleNodeInfoFromRadioGomeshproto(nodeInfo)
	} else if channel := msg.GetChannel(); channel != nil {
		rc.logger.Debug("FromRadio contains Channel")
		rc.handleChannelGomeshproto(channel)
	} else if configId := msg.GetConfigCompleteId(); configId != 0 {
		rc.logger.Info("Configuration complete", "config_id", configId)
	} else {
		rc.logger.Debug("FromRadio with unknown payload type")
	}
}

func (rc *RadioClient) parsePacketManually(data []byte) {
	if len(data) < 2 {
		return
	}

	// Enhanced protobuf field parsing with better packet type detection
	// Based on official Meshtastic protobuf definitions research
	rc.logger.Debug("Manual packet parsing", "first_bytes", fmt.Sprintf("%02x %02x %02x %02x",
		data[0], getByteAt(data, 1), getByteAt(data, 2), getByteAt(data, 3)))

	if len(data) >= 4 {
		// FromRadio message field analysis:
		// Field 1 (id): 0x08 = varint
		// Field 2 (packet): 0x12 = length-delimited
		// Field 3 (my_info): 0x1a = length-delimited
		// Field 4 (node_info): 0x22 = length-delimited
		// Field 100 (config_complete_id): 0xa0, 0x06 = varint

		switch data[0] {
		case 0x08:
			// Field 1: FromRadio.id (varint)
			rc.logger.Debug("Detected FromRadio with id field")
			rc.parseFromRadioManually(data)
		case 0x1a:
			// Field 3: MyNodeInfo in FromRadio
			rc.logger.Debug("Detected MyNodeInfo packet", "size", len(data))
			rc.parseMyNodeInfoManually(data[1:])
		case 0x22:
			// Field 4: NodeInfo in FromRadio
			rc.logger.Debug("Detected NodeInfo packet", "size", len(data))
			rc.extractNodeInfoSafely(data[1:])
		case 0x12:
			// Field 2: MeshPacket in FromRadio
			rc.logger.Debug("Detected MeshPacket", "size", len(data))
			// Handle MeshPacket parsing carefully
		case 0xa0:
			if len(data) > 1 && data[1] == 0x06 {
				// Field 100: config_complete_id
				rc.logger.Info("Configuration complete detected")
			}
		default:
			rc.logger.Debug("Unknown packet type", "first_byte", fmt.Sprintf("0x%02x", data[0]))
		}
	}
}

func getByteAt(data []byte, index int) byte {
	if index < len(data) {
		return data[index]
	}
	return 0x00
}

func (rc *RadioClient) parseFromRadioManually(data []byte) {
	// Simple FromRadio parsing - look for subsequent fields after the id
	i := 0
	for i < len(data) {
		if i >= len(data) {
			break
		}

		tag := data[i]
		i++

		switch tag {
		case 0x1a: // my_info field
			if i < len(data) {
				length := int(data[i])
				i++
				if i+length <= len(data) {
					rc.parseMyNodeInfoManually(data[i : i+length])
					i += length
				}
			}
		case 0x22: // node_info field
			if i < len(data) {
				length := int(data[i])
				i++
				if i+length <= len(data) {
					rc.extractNodeInfoSafely(data[i : i+length])
					i += length
				}
			}
		default:
			// Skip unknown fields
			if i < len(data) && (tag&0x7) == 2 { // length-delimited
				length := int(data[i])
				i += 1 + length
			} else {
				i++ // Skip varint or other
			}
		}
	}
}

func (rc *RadioClient) parseMyNodeInfoManually(data []byte) {
	// Extract basic MyNodeInfo data manually
	rc.logger.Debug("Parsing MyNodeInfo manually", "size", len(data))

	// Look for node number field (field 1, tag 0x08)
	for i := 0; i < len(data)-4; i++ {
		if data[i] == 0x08 { // varint field 1
			// Extract node number (simplified varint parsing)
			nodeNum := uint32(data[i+1])
			if data[i+1] < 0x80 { // single byte varint
				rc.ownNodeID = nodeNum
				rc.logger.Info("Found own node ID", "id", nodeNum)
				break
			}
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

// New gomeshproto-based handlers using proper protobuf definitions

func (rc *RadioClient) handleMeshPacketGomeshproto(packet *gomeshproto.MeshPacket) {
	// Enhanced null pointer protection with gomeshproto
	if packet == nil {
		rc.logger.Debug("Received nil MeshPacket")
		return
	}

	// Update receive metadata
	if packet.RxTime == 0 {
		now := time.Now().Unix()
		packet.RxTime = uint32(now & 0xFFFFFFFF) // Safe truncation to 32-bit
	}

	// Get the decoded payload
	decodedData := packet.GetDecoded()
	if decodedData == nil {
		// Handle encrypted packets - log and skip for now
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

	rc.logger.Debug("Received MeshPacket", "from", packet.From, "to", packet.To, "portnum", decodedData.Portnum)

	// Handle different packet types based on portnum
	switch decodedData.Portnum {
	case gomeshproto.PortNum_TEXT_MESSAGE_APP:
		rc.handleTextMessageGomeshproto(packet, decodedData)
	case gomeshproto.PortNum_NODEINFO_APP:
		rc.handleNodeInfoGomeshproto(packet, decodedData)
	case gomeshproto.PortNum_POSITION_APP:
		rc.handlePositionGomeshproto(packet, decodedData)
	case gomeshproto.PortNum_ROUTING_APP:
		rc.handleRoutingGomeshproto(packet, decodedData)
	case gomeshproto.PortNum_ADMIN_APP:
		rc.handleAdminMessageGomeshproto(packet, decodedData)
	default:
		rc.logger.Debug("Unhandled packet type", "portnum", decodedData.Portnum)
	}
}

func (rc *RadioClient) handleMyNodeInfoGomeshproto(myInfo *gomeshproto.MyNodeInfo) {
	if myInfo == nil {
		rc.logger.Debug("Received nil MyNodeInfo")
		return
	}

	rc.logger.Info("Received MyNodeInfo via gomeshproto",
		"node_num", myInfo.MyNodeNum,
		"reboot_count", myInfo.RebootCount,
		"min_app_version", myInfo.MinAppVersion)

	// Store our own node ID
	rc.ownNodeID = myInfo.MyNodeNum
}

func (rc *RadioClient) handleNodeInfoFromRadioGomeshproto(nodeInfo *gomeshproto.NodeInfo) {
	if nodeInfo == nil {
		rc.logger.Debug("Received nil NodeInfo")
		return
	}

	rc.logger.Info("Received NodeInfo from config via gomeshproto", "node_num", nodeInfo.Num)

	// Convert to our internal node format
	node := rc.convertGomeshprotoNodeInfo(nodeInfo)
	if node != nil {
		// Emit node updated event
		rc.events <- core.Event{
			Type: core.EventNodeUpdated,
			Data: node,
		}
	}
}

func (rc *RadioClient) handleTextMessageGomeshproto(packet *gomeshproto.MeshPacket, data *gomeshproto.Data) {
	if packet == nil || data == nil {
		rc.logger.Debug("Received nil packet or data in text message")
		return
	}

	// Only process messages from TEXT_MESSAGE_APP port
	if data.Portnum != gomeshproto.PortNum_TEXT_MESSAGE_APP {
		rc.logger.Debug("Skipping non-text message", "portnum", data.Portnum)
		return
	}

	if len(data.Payload) == 0 {
		rc.logger.Debug("Empty text message payload")
		return
	}

	// Validate that payload is valid UTF-8 text
	if !utf8.Valid(data.Payload) {
		rc.logger.Debug("Invalid UTF-8 in text message payload", "payload_hex", fmt.Sprintf("%x", data.Payload[:minInt(20, len(data.Payload))]))
		return
	}

	text := string(data.Payload)

	// Additional validation for reasonable text messages
	if len(text) == 0 || len(text) > 200 {
		rc.logger.Debug("Text message length out of bounds", "length", len(text))
		return
	}

	// Note: Removed unreliable text content filtering - rely on proper Meshtastic protocol port numbers instead

	// Determine chat ID
	chatID := rc.getChatIDFromGomeshproto(packet)
	if chatID == "" {
		rc.logger.Debug("Skipping message with invalid chat ID")
		return
	}

	// Update node encryption status and signal metrics based on the message
	nodeID := fmt.Sprintf("%d", packet.From)
	node := rc.getOrCreateNode(nodeID)
	rc.updateNodeEncryptionFromMessage(node, packet)
	rc.updateNodeMetricsFromGomeshproto(node, packet)

	// Emit node updated event after signal metrics update
	rc.events <- core.Event{
		Type: core.EventNodeUpdated,
		Data: node,
	}

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

	rc.logger.Info("Text message received via gomeshproto",
		"from", packet.From,
		"chat", chatID,
		"text", text)
}

func (rc *RadioClient) handleNodeInfoGomeshproto(packet *gomeshproto.MeshPacket, data *gomeshproto.Data) {
	if packet == nil {
		rc.logger.Debug("Received nil packet in nodeinfo")
		return
	}

	if len(data.Payload) == 0 {
		// Empty payload might be a request - just update node from packet metadata
		nodeID := fmt.Sprintf("%d", packet.From)
		node := rc.getOrCreateNode(nodeID)
		rc.updateNodeMetricsFromGomeshproto(node, packet)

		// Emit node updated event
		rc.events <- core.Event{
			Type: core.EventNodeUpdated,
			Data: node,
		}
		return
	}

	// Try to decode NodeInfo from payload
	nodeInfo := &gomeshproto.NodeInfo{}
	if err := proto.Unmarshal(data.Payload, nodeInfo); err != nil {
		rc.logger.Debug("Failed to decode NodeInfo payload", "error", err)
		return
	}

	node := rc.convertGomeshprotoNodeInfo(nodeInfo)
	if node != nil {
		rc.updateNodeMetricsFromGomeshproto(node, packet)

		// Emit node updated event
		rc.events <- core.Event{
			Type: core.EventNodeUpdated,
			Data: node,
		}
	}

	rc.logger.Info("Node info received via gomeshproto", "node", node.ID)
}

func (rc *RadioClient) handlePositionGomeshproto(packet *gomeshproto.MeshPacket, data *gomeshproto.Data) {
	// Placeholder for position handling
	rc.logger.Debug("Position packet received via gomeshproto", "from", packet.From, "payload_size", len(data.Payload))
}

func (rc *RadioClient) handleRoutingGomeshproto(packet *gomeshproto.MeshPacket, data *gomeshproto.Data) {
	// Placeholder for routing handling
	rc.logger.Debug("Routing packet received via gomeshproto", "from", packet.From, "payload_size", len(data.Payload))
}

func (rc *RadioClient) handleAdminMessageGomeshproto(packet *gomeshproto.MeshPacket, data *gomeshproto.Data) {
	if packet == nil || data == nil {
		rc.logger.Debug("Received nil packet or data in admin message")
		return
	}

	rc.logger.Debug("Admin message received via gomeshproto", "from", packet.From, "payload_size", len(data.Payload))

	// Decode AdminMessage from payload
	adminMsg := &gomeshproto.AdminMessage{}
	if err := proto.Unmarshal(data.Payload, adminMsg); err != nil {
		rc.logger.Debug("Failed to decode AdminMessage payload", "error", err)
		return
	}

	// Handle different admin message types
	if response := adminMsg.GetGetChannelResponse(); response != nil {
		rc.logger.Debug("Received GetChannelResponse")
		rc.handleChannelGomeshproto(response)
	} else {
		// Check if this is a channel request by looking at the payload variant
		switch adminMsg.PayloadVariant.(type) {
		case *gomeshproto.AdminMessage_GetChannelRequest:
			request := adminMsg.GetGetChannelRequest()
			rc.logger.Debug("Received GetChannelRequest", "channel_index", request)
			// This is a request - we don't need to handle it as clients
		default:
			rc.logger.Debug("Received unknown AdminMessage type")
		}
	}
}

// Helper methods

func (rc *RadioClient) convertGomeshprotoNodeInfo(nodeInfo *gomeshproto.NodeInfo) *core.Node {
	if nodeInfo == nil {
		return nil
	}

	nodeID := fmt.Sprintf("%d", nodeInfo.Num)
	node := rc.getOrCreateNode(nodeID)

	// Update from NodeInfo if available
	if nodeInfo.User != nil {
		node.ShortName = nodeInfo.User.ShortName
		node.LongName = nodeInfo.User.LongName
	}

	if nodeInfo.Position != nil {
		node.Position = &core.Position{
			LatitudeI:  nodeInfo.Position.LatitudeI,
			LongitudeI: nodeInfo.Position.LongitudeI,
			Altitude:   nodeInfo.Position.Altitude,
			Time:       nodeInfo.Position.Time,
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
		}
	}

	return node
}

func (rc *RadioClient) updateNodeMetricsFromGomeshproto(node *core.Node, packet *gomeshproto.MeshPacket) {
	if node == nil || packet == nil {
		rc.logger.Debug("updateNodeMetricsFromGomeshproto called with nil node or packet")
		return
	}

	rc.logger.Debug("Updating node signal metrics", "node", node.ID)

	// Update signal metrics
	node.RSSI = int(packet.RxRssi)
	node.SNR = packet.RxSnr
	signalQuality := core.CalculateSignalQuality(node.RSSI, node.SNR)
	node.SignalQuality = int(signalQuality)
	node.LastHeard = time.Unix(int64(packet.RxTime), 0)

	// Log signal quality updates for important events
	if node.RSSI != 0 || node.SNR != 0 {
		rc.logger.Debug("Node signal updated", "node", node.ID, "rssi", node.RSSI, "snr", node.SNR, "quality", signalQuality.String())
	}

	// Store in node database
	rc.nodeDBMu.Lock()
	rc.nodeDB[node.ID] = node
	rc.nodeDBMu.Unlock()
}

func (rc *RadioClient) getChatIDFromGomeshproto(packet *gomeshproto.MeshPacket) string {
	if packet == nil {
		rc.logger.Warn("Attempted to get chat ID from nil packet")
		return "" // Return empty string instead of "unknown" to prevent fake chats
	}

	// Additional validation - ensure packet has valid addressing
	if packet.From == 0 {
		rc.logger.Debug("Packet has invalid From address (0)")
		return ""
	}

	if packet.To == 0xFFFFFFFF {
		return fmt.Sprintf("channel_%d", packet.Channel)
	}
	return fmt.Sprintf("%d", packet.From)
}

// Legacy methods - keeping for backward compatibility

func (rc *RadioClient) handleMeshPacket(packet *MeshPacket) {
	// Enhanced null pointer protection based on lmatte7/meshtastic-go research
	if packet == nil {
		rc.logger.Debug("Received nil MeshPacket")
		return
	}

	// Update receive metadata
	if packet.RxTime == 0 {
		now := time.Now().Unix()
		packet.RxTime = uint32(now & 0xFFFFFFFF) // Safe truncation to 32-bit
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
	now := time.Now().Unix()
	packet := &MeshPacket{
		From:   nodeInfo.Num,
		RxTime: uint32(now & 0xFFFFFFFF), // Safe truncation to 32-bit
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
	if chatID == "" {
		rc.logger.Debug("Skipping message with invalid chat ID (manual parsing)")
		return
	}

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

	rc.logger.Debug("Created new node", "node_id", nodeID)
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
	nano := time.Now().UnixNano()
	return uint32(nano & 0xFFFFFFFF) // Safe truncation to 32-bit
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

func (rc *RadioClient) requestAllChannels(ctx context.Context) {
	rc.logger.Info("Requesting all channels...")

	// Request channels 0-7 (typical Meshtastic supports up to 8 channels)
	for i := uint32(0); i < 8; i++ {
		rc.logger.Debug("Requesting channel", "index", i)

		// Create AdminMessage to request channel
		adminMsg := &gomeshproto.AdminMessage{
			PayloadVariant: &gomeshproto.AdminMessage_GetChannelRequest{
				GetChannelRequest: i,
			},
		}

		// Wrap in Data payload
		data := &gomeshproto.Data{
			Portnum: gomeshproto.PortNum_ADMIN_APP,
			Payload: func() []byte {
				bytes, _ := proto.Marshal(adminMsg)
				return bytes
			}(),
		}

		// Create MeshPacket
		packet := &gomeshproto.MeshPacket{
			From:     rc.ownNodeID,
			To:       rc.ownNodeID, // Send to ourselves (the radio)
			Channel:  0,
			Id:       rc.generatePacketID(),
			WantAck:  false,
			Priority: gomeshproto.MeshPacket_DEFAULT,
			PayloadVariant: &gomeshproto.MeshPacket_Decoded{
				Decoded: data,
			},
		}

		// Wrap in ToRadio
		toRadio := &gomeshproto.ToRadio{
			PayloadVariant: &gomeshproto.ToRadio_Packet{
				Packet: packet,
			},
		}

		packetBytes, err := proto.Marshal(toRadio)
		if err != nil {
			rc.logger.Error("Failed to marshal channel request", "error", err, "channel", i)
			continue
		}

		if err := rc.transport.WritePacket(ctx, packetBytes); err != nil {
			rc.logger.Error("Failed to send channel request", "error", err, "channel", i)
		}

		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
}

func (rc *RadioClient) GetOwnNodeID() uint32 {
	return rc.ownNodeID
}

func minInt(a, b int) int {
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

func (rc *RadioClient) handleChannelGomeshproto(channel *gomeshproto.Channel) {
	if channel == nil {
		rc.logger.Debug("Received nil Channel")
		return
	}

	channelName := rc.getChannelName(channel)
	rc.logger.Info("Received Channel configuration",
		"index", channel.Index,
		"name", channelName)

	// Store channel configuration
	rc.channelDBMu.Lock()
	rc.channelDB[channel.Index] = channel
	rc.channelDBMu.Unlock()

	// Create a chat for this channel if it has a valid name and settings
	if channel.Settings != nil && channelName != "" && channel.Settings.Name != "" {
		// Create a fake message to trigger chat creation
		// This simulates receiving a channel configuration as a "system message"
		chatID := fmt.Sprintf("channel_%d", channel.Index)

		// Determine encryption level from channel settings
		encryption := 0 // Default: unencrypted
		if channel.Settings.Psk != nil {
			psk := channel.Settings.Psk
			if len(psk) == 1 && psk[0] >= 1 && psk[0] <= 10 {
				encryption = 1 // Default key encryption
			} else if len(psk) == 16 || len(psk) == 32 {
				encryption = 2 // Custom key encryption
			}
		}

		rc.logger.Info("Creating chat for discovered channel",
			"chat_id", chatID,
			"channel_name", channelName,
			"channel_index", channel.Index,
			"encryption", encryption)

		// Emit chat updated event directly instead of creating a fake message
		rc.events <- core.Event{
			Type: core.EventChatUpdated,
			Data: map[string]interface{}{
				"chat_id":    chatID,
				"title":      channelName,
				"encryption": encryption,
			},
		}
	}
}

func (rc *RadioClient) getChannelName(channel *gomeshproto.Channel) string {
	if channel == nil || channel.Settings == nil {
		return ""
	}
	return channel.Settings.Name
}

func (rc *RadioClient) GetChannelName(channelIndex int32) string {
	rc.channelDBMu.RLock()
	defer rc.channelDBMu.RUnlock()

	if channel, exists := rc.channelDB[channelIndex]; exists {
		return rc.getChannelName(channel)
	}
	return ""
}

func (rc *RadioClient) updateNodeEncryptionFromMessage(node *core.Node, packet *gomeshproto.MeshPacket) {
	if node == nil || packet == nil {
		return
	}

	// Check if this was an encrypted or unencrypted message
	if packet.GetEncrypted() != nil && len(packet.GetEncrypted()) > 0 {
		// Message was encrypted, but we were able to decrypt it
		// This means we have the same encryption key as the sender

		// Check the channel to determine encryption type
		rc.channelDBMu.RLock()
		channelID := int32(packet.Channel & 0x7FFFFFFF) // Safe conversion to signed int32
		channel, hasChannel := rc.channelDB[channelID]
		rc.channelDBMu.RUnlock()

		if hasChannel && channel.Settings != nil {
			// Determine encryption type based on channel settings
			psk := channel.Settings.Psk
			if len(psk) == 0 {
				// No PSK means unencrypted channel
				node.Unencrypted = true
				node.EncDefaultKey = false
				node.EncCustomKey = false
			} else if len(psk) == 1 && psk[0] >= 1 && psk[0] <= 10 {
				// Single byte 1-10 means default encryption
				node.EncDefaultKey = true
				node.Unencrypted = false
				node.EncCustomKey = false
			} else if len(psk) == 16 || len(psk) == 32 {
				// 16 or 32 bytes means custom key
				node.EncCustomKey = true
				node.Unencrypted = false
				node.EncDefaultKey = false
			} else {
				// Unknown PSK format - assume default
				node.EncDefaultKey = true
				node.Unencrypted = false
				node.EncCustomKey = false
			}
		} else {
			// No channel info available - assume default encryption if we could decrypt
			node.EncDefaultKey = true
			node.Unencrypted = false
			node.EncCustomKey = false
		}

		rc.logger.Debug("Updated node encryption from encrypted message",
			"node", node.ID,
			"channel", packet.Channel,
			"enc_default", node.EncDefaultKey,
			"enc_custom", node.EncCustomKey,
			"unencrypted", node.Unencrypted)
	} else if packet.GetDecoded() != nil {
		// Message was unencrypted (we got decoded data directly)
		node.Unencrypted = true
		node.EncDefaultKey = false
		node.EncCustomKey = false

		rc.logger.Debug("Updated node encryption from unencrypted message",
			"node", node.ID,
			"channel", packet.Channel,
			"unencrypted", true)
	}
}
