package radio

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

const broadcastNodeNum = ^uint32(0)
const meshtasticPositionScale = 1e-7

// MeshtasticCodec implements Codec for Meshtastic protobuf frames.
type MeshtasticCodec struct {
	wantConfigID atomic.Uint32
	packetID     atomic.Uint32
	localNodeNum atomic.Uint32
	modemPreset  atomic.Int32
}

func NewMeshtasticCodec() (*MeshtasticCodec, error) {
	var seedRaw [4]byte
	if _, err := rand.Read(seedRaw[:]); err != nil {
		return nil, fmt.Errorf("seed meshtastic codec packet id: %w", err)
	}
	seed := binary.BigEndian.Uint32(seedRaw[:])
	c := &MeshtasticCodec{}
	c.packetID.Store(seed)
	c.modemPreset.Store(int32(generated.Config_LoRaConfig_LONG_FAST))

	return c, nil
}

func (c *MeshtasticCodec) LocalNodeID() string {
	localNodeNum := c.localNodeNum.Load()
	if localNodeNum == 0 {
		return ""
	}

	return formatNodeNum(localNodeNum)
}

func (c *MeshtasticCodec) EncodeWantConfig() ([]byte, error) {
	id := c.nextNonZeroID()
	wire := &generated.ToRadio{PayloadVariant: &generated.ToRadio_WantConfigId{WantConfigId: id}}
	payload, err := proto.Marshal(wire)
	if err != nil {
		return nil, err
	}
	c.wantConfigID.Store(id)

	return payload, nil
}

func (c *MeshtasticCodec) EncodeHeartbeat() ([]byte, error) {
	wire := &generated.ToRadio{PayloadVariant: &generated.ToRadio_Heartbeat{Heartbeat: &generated.Heartbeat{}}}

	return proto.Marshal(wire)
}

func (c *MeshtasticCodec) EncodeText(chatKey, text string) (EncodedText, error) {
	to, channel, err := parseChatTarget(chatKey)
	if err != nil {
		return EncodedText{}, err
	}
	packetID := c.nextNonZeroID()

	packet := &generated.MeshPacket{
		To:      to,
		Channel: channel,
		Id:      packetID,
		WantAck: to != broadcastNodeNum,
		PayloadVariant: &generated.MeshPacket_Decoded{Decoded: &generated.Data{
			Portnum: generated.PortNum_TEXT_MESSAGE_APP,
			Payload: []byte(text),
		}},
	}
	wire := &generated.ToRadio{PayloadVariant: &generated.ToRadio_Packet{Packet: packet}}
	payload, err := proto.Marshal(wire)
	if err != nil {
		return EncodedText{}, err
	}

	return EncodedText{
		Payload:         payload,
		DeviceMessageID: strconv.FormatUint(uint64(packetID), 10),
		WantAck:         packet.GetWantAck(),
	}, nil
}

func (c *MeshtasticCodec) EncodeAdmin(
	to uint32,
	channel uint32,
	wantResponse bool,
	payload *generated.AdminMessage,
) (EncodedAdmin, error) {
	if payload == nil {
		return EncodedAdmin{}, fmt.Errorf("admin payload is required")
	}
	encodedAdmin, err := proto.Marshal(payload)
	if err != nil {
		return EncodedAdmin{}, fmt.Errorf("marshal admin payload: %w", err)
	}
	packetID := c.nextNonZeroID()
	packet := &generated.MeshPacket{
		To:       to,
		Channel:  channel,
		Id:       packetID,
		WantAck:  true,
		Priority: generated.MeshPacket_RELIABLE,
		PayloadVariant: &generated.MeshPacket_Decoded{Decoded: &generated.Data{
			Portnum:      generated.PortNum_ADMIN_APP,
			Payload:      encodedAdmin,
			WantResponse: wantResponse,
		}},
	}
	wire := &generated.ToRadio{PayloadVariant: &generated.ToRadio_Packet{Packet: packet}}
	encoded, err := proto.Marshal(wire)
	if err != nil {
		return EncodedAdmin{}, fmt.Errorf("marshal admin packet: %w", err)
	}

	return EncodedAdmin{
		Payload:         encoded,
		DeviceMessageID: strconv.FormatUint(uint64(packetID), 10),
	}, nil
}

func (c *MeshtasticCodec) EncodeTraceroute(to uint32, channel uint32) (EncodedTraceroute, error) {
	packetID := c.nextNonZeroID()
	packet := &generated.MeshPacket{
		To:      to,
		Channel: channel,
		Id:      packetID,
		WantAck: true,
		PayloadVariant: &generated.MeshPacket_Decoded{Decoded: &generated.Data{
			Portnum:      generated.PortNum_TRACEROUTE_APP,
			WantResponse: true,
		}},
	}
	wire := &generated.ToRadio{PayloadVariant: &generated.ToRadio_Packet{Packet: packet}}
	encoded, err := proto.Marshal(wire)
	if err != nil {
		return EncodedTraceroute{}, fmt.Errorf("marshal traceroute packet: %w", err)
	}

	return EncodedTraceroute{
		Payload:         encoded,
		DeviceMessageID: strconv.FormatUint(uint64(packetID), 10),
	}, nil
}

func (c *MeshtasticCodec) DecodeFromRadio(payload []byte) (DecodedFrame, error) {
	out := DecodedFrame{Raw: payload}

	var wire generated.FromRadio
	if err := proto.Unmarshal(payload, &wire); err != nil {
		return out, fmt.Errorf("decode fromradio protobuf: %w", err)
	}

	now := time.Now()
	if my := wire.GetMyInfo(); my != nil && my.GetMyNodeNum() != 0 {
		c.localNodeNum.Store(my.GetMyNodeNum())
	}
	if cfg := wire.GetConfig(); cfg != nil {
		c.updateModemPresetFromConfig(cfg)
	}

	if configID := wire.GetConfigCompleteId(); configID != 0 {
		out.ConfigCompleteID = configID
		expected := c.wantConfigID.Load()
		if expected != 0 && configID == expected {
			out.WantConfigReady = true
		}
	}

	if nodeInfo := wire.GetNodeInfo(); nodeInfo != nil {
		nodeUpdate := decodeNodeInfo(nodeInfo, now)
		out.NodeUpdate = &nodeUpdate
	}

	if channelInfo := wire.GetChannel(); channelInfo != nil {
		defaultTitle := c.defaultPresetChannelTitle()
		if channelList, snapshot, ok := decodeChannelInfo(channelInfo, defaultTitle); ok {
			out.Channels = &channelList
			out.ConfigSnapshot = &snapshot
		}
	}
	if queueStatus := wire.GetQueueStatus(); queueStatus != nil {
		if status, ok := decodeQueueStatus(queueStatus); ok {
			out.MessageStatus = &status
		}
	}

	if packet := wire.GetPacket(); packet != nil {
		decodePacket(packet, now, c.localNodeNum.Load(), &out)
	}

	return out, nil
}

func decodePacket(packet *generated.MeshPacket, now time.Time, localNode uint32, out *DecodedFrame) {
	decoded := packet.GetDecoded()
	if decoded == nil {
		return
	}
	if status, ok := decodePacketStatus(packet, decoded); ok {
		out.MessageStatus = &status
	}

	switch decoded.GetPortnum() {
	case generated.PortNum_TEXT_MESSAGE_APP, generated.PortNum_TEXT_MESSAGE_COMPRESSED_APP, generated.PortNum_DETECTION_SENSOR_APP, generated.PortNum_ALERT_APP:
		text := strings.TrimSpace(string(decoded.GetPayload()))
		if text == "" {
			return
		}

		direction := domain.MessageDirectionIn
		if localNode != 0 && packet.GetFrom() == localNode {
			direction = domain.MessageDirectionOut
		}
		status := domain.MessageStatusSent
		if direction == domain.MessageDirectionOut && packet.GetWantAck() {
			status = domain.MessageStatusPending
		}

		msg := domain.ChatMessage{
			ChatKey:   chatKeyForPacket(packet, direction),
			Direction: direction,
			Body:      text,
			Status:    status,
			At:        packetTimestamp(packet.GetRxTime(), now),
			MetaJSON:  packetMetaJSON(decoded.GetPortnum(), packet),
		}
		if packet.GetId() != 0 {
			msg.DeviceMessageID = strconv.FormatUint(uint64(packet.GetId()), 10)
		}
		out.TextMessage = &msg
	case generated.PortNum_NODEINFO_APP:
		if nodeUpdate, ok := decodeNodeFromPacketPayload(packet, decoded.GetPayload(), now); ok {
			out.NodeUpdate = &nodeUpdate
		}
	case generated.PortNum_TELEMETRY_APP:
		if nodeUpdate, ok := decodeNodeTelemetryFromPacket(packet, decoded.GetPayload(), now); ok {
			out.NodeUpdate = &nodeUpdate
		}
	case generated.PortNum_POSITION_APP:
		if nodeUpdate, ok := decodeNodePositionFromPacket(packet, decoded.GetPayload(), now); ok {
			out.NodeUpdate = &nodeUpdate
		}
	case generated.PortNum_ADMIN_APP:
		var admin generated.AdminMessage
		if err := proto.Unmarshal(decoded.GetPayload(), &admin); err != nil {
			return
		}
		out.AdminMessage = &AdminMessageEvent{
			From:      packet.GetFrom(),
			To:        packet.GetTo(),
			PacketID:  packet.GetId(),
			RequestID: decoded.GetRequestId(),
			ReplyID:   decoded.GetReplyId(),
			Message:   &admin,
		}
	case generated.PortNum_TRACEROUTE_APP:
		if event, ok := decodeTracerouteEvent(packet, decoded); ok {
			out.Traceroute = &event
		}
	}
}

func decodeTracerouteEvent(packet *generated.MeshPacket, decoded *generated.Data) (connectors.TracerouteEvent, bool) {
	if decoded == nil || decoded.GetWantResponse() {
		return connectors.TracerouteEvent{}, false
	}
	var routeDiscovery generated.RouteDiscovery
	if err := proto.Unmarshal(decoded.GetPayload(), &routeDiscovery); err != nil {
		return connectors.TracerouteEvent{}, false
	}

	destinationID := decoded.GetDest()
	if destinationID == 0 {
		destinationID = packet.GetTo()
	}
	sourceID := decoded.GetSource()
	if sourceID == 0 {
		sourceID = packet.GetFrom()
	}

	fullRoute := make([]uint32, 0, len(routeDiscovery.GetRoute())+2)
	if destinationID != 0 {
		fullRoute = append(fullRoute, destinationID)
	}
	fullRoute = append(fullRoute, routeDiscovery.GetRoute()...)
	if sourceID != 0 {
		fullRoute = append(fullRoute, sourceID)
	}

	routeBack := routeDiscovery.GetRouteBack()
	fullRouteBack := make([]uint32, 0, len(routeBack)+2)
	if (packet.GetHopStart() > 0 || decoded.GetBitfield() != 0) && len(routeDiscovery.GetSnrBack()) > 0 {
		if sourceID != 0 {
			fullRouteBack = append(fullRouteBack, sourceID)
		}
		fullRouteBack = append(fullRouteBack, routeBack...)
		if destinationID != 0 {
			fullRouteBack = append(fullRouteBack, destinationID)
		}
	} else {
		fullRouteBack = append(fullRouteBack, routeBack...)
	}

	requestID := decoded.GetRequestId()
	if requestID == 0 {
		requestID = decoded.GetReplyId()
	}

	return connectors.TracerouteEvent{
		From:       packet.GetFrom(),
		To:         packet.GetTo(),
		PacketID:   packet.GetId(),
		RequestID:  requestID,
		ReplyID:    decoded.GetReplyId(),
		Route:      fullRoute,
		SnrTowards: append([]int32(nil), routeDiscovery.GetSnrTowards()...),
		RouteBack:  fullRouteBack,
		SnrBack:    append([]int32(nil), routeDiscovery.GetSnrBack()...),
		IsComplete: len(fullRoute) > 0 && len(fullRouteBack) > 0,
	}, true
}

func decodeQueueStatus(queueStatus *generated.QueueStatus) (domain.MessageStatusUpdate, bool) {
	packetID := queueStatus.GetMeshPacketId()
	if packetID == 0 {
		return domain.MessageStatusUpdate{}, false
	}

	update := domain.MessageStatusUpdate{
		DeviceMessageID: strconv.FormatUint(uint64(packetID), 10),
		Status:          domain.MessageStatusSent,
	}
	if res := generated.Routing_Error(queueStatus.GetRes()); res != generated.Routing_NONE {
		update.Status = domain.MessageStatusFailed
		update.Reason = res.String()
	}

	return update, true
}

func decodePacketStatus(packet *generated.MeshPacket, decoded *generated.Data) (domain.MessageStatusUpdate, bool) {
	requestID := decoded.GetRequestId()
	if requestID == 0 {
		return domain.MessageStatusUpdate{}, false
	}

	isRouting := decoded.GetPortnum() == generated.PortNum_ROUTING_APP
	isAck := packet.GetPriority() == generated.MeshPacket_ACK
	if !isRouting && !isAck {
		return domain.MessageStatusUpdate{}, false
	}

	update := domain.MessageStatusUpdate{
		DeviceMessageID: strconv.FormatUint(uint64(requestID), 10),
		Status:          domain.MessageStatusAcked,
	}

	if isRouting {
		var routing generated.Routing
		if err := proto.Unmarshal(decoded.GetPayload(), &routing); err == nil {
			if reason := routing.GetErrorReason(); reason != generated.Routing_NONE {
				update.Status = domain.MessageStatusFailed
				update.Reason = reason.String()
			}
		}
	}

	return update, true
}

func decodeNodeInfo(nodeInfo *generated.NodeInfo, now time.Time) domain.NodeUpdate {
	user := nodeInfo.GetUser()
	node := domain.Node{
		NodeID:      formatNodeNum(nodeInfo.GetNum()),
		LongName:    strings.TrimSpace(user.GetLongName()),
		ShortName:   strings.TrimSpace(user.GetShortName()),
		LastHeardAt: packetTimestamp(nodeInfo.GetLastHeard(), now),
		UpdatedAt:   now,
	}
	if model := user.GetHwModel(); model != generated.HardwareModel_UNSET {
		node.BoardModel = model.String()
	}
	if role := strings.TrimSpace(user.GetRole().String()); role != "" {
		node.Role = role
	}
	if user != nil && user.IsUnmessagable != nil {
		v := user.GetIsUnmessagable()
		node.IsUnmessageable = &v
	}
	applyPositionCoordinates(&node, nodeInfo.GetPosition())
	applyDeviceMetrics(&node, nodeInfo.GetDeviceMetrics())

	if snr := nodeInfo.GetSnr(); snr != 0 {
		snrVal := float64(snr)
		node.SNR = &snrVal
	}

	return domain.NodeUpdate{
		Node:       node,
		LastHeard:  node.LastHeardAt,
		FromPacket: true,
		Type:       domain.NodeUpdateTypeNodeInfoSnapshot,
	}
}

func decodeNodeTelemetryFromPacket(packet *generated.MeshPacket, payload []byte, now time.Time) (domain.NodeUpdate, bool) {
	if packet.GetFrom() == 0 {
		return domain.NodeUpdate{}, false
	}

	var telemetry generated.Telemetry
	if err := proto.Unmarshal(payload, &telemetry); err != nil {
		return domain.NodeUpdate{}, false
	}

	node := domain.Node{
		NodeID:      formatNodeNum(packet.GetFrom()),
		Channel:     uint32Ptr(packet.GetChannel()),
		LastHeardAt: packetTimestamp(packet.GetRxTime(), now),
		UpdatedAt:   now,
	}

	applyDeviceMetrics(&node, telemetry.GetDeviceMetrics())
	applyEnvironmentMetrics(&node, telemetry.GetEnvironmentMetrics())
	applyPowerMetrics(&node, telemetry.GetPowerMetrics())
	applyAirQualityMetrics(&node, telemetry.GetAirQualityMetrics())

	if telemetry.GetEnvironmentMetrics() == nil && telemetry.GetPowerMetrics() == nil && telemetry.GetAirQualityMetrics() == nil && telemetry.GetDeviceMetrics() == nil {
		return domain.NodeUpdate{}, false
	}
	if rssi := packet.GetRxRssi(); rssi != 0 {
		rssiVal := int(rssi)
		node.RSSI = &rssiVal
	}
	if snr := packet.GetRxSnr(); snr != 0 {
		snrVal := float64(snr)
		node.SNR = &snrVal
	}

	return domain.NodeUpdate{
		Node:       node,
		LastHeard:  node.LastHeardAt,
		FromPacket: true,
		Type:       domain.NodeUpdateTypeTelemetryPacket,
	}, true
}

func decodeChannelInfo(channelInfo *generated.Channel, defaultPresetTitle string) (domain.ChannelList, connectors.ConfigSnapshot, bool) {
	if channelInfo.GetRole() == generated.Channel_DISABLED {
		return domain.ChannelList{}, connectors.ConfigSnapshot{}, false
	}
	idx := int(channelInfo.GetIndex())
	if idx < 0 {
		return domain.ChannelList{}, connectors.ConfigSnapshot{}, false
	}

	title := strings.TrimSpace(channelInfo.GetSettings().GetName())
	if title == "" {
		title = strings.TrimSpace(defaultPresetTitle)
		if title == "" {
			title = fmt.Sprintf("Channel %d", idx)
		}
	}

	list := domain.ChannelList{Items: []domain.ChannelInfo{{Index: idx, Title: title}}}
	snapshot := connectors.ConfigSnapshot{ChannelTitles: []string{title}}

	return list, snapshot, true
}

func (c *MeshtasticCodec) updateModemPresetFromConfig(cfg *generated.Config) {
	if cfg == nil {
		return
	}
	lora := cfg.GetLora()
	if lora == nil {
		return
	}
	c.modemPreset.Store(int32(lora.GetModemPreset()))
}

func (c *MeshtasticCodec) defaultPresetChannelTitle() string {
	preset := generated.Config_LoRaConfig_ModemPreset(c.modemPreset.Load())

	return modemPresetTitle(preset)
}

func modemPresetTitle(preset generated.Config_LoRaConfig_ModemPreset) string {
	switch preset.String() {
	case "LONG_FAST":
		return "LongFast"
	case "LONG_SLOW":
		return "LongSlow"
	case "VERY_LONG_SLOW":
		return "VeryLongSlow"
	case "MEDIUM_SLOW":
		return "MediumSlow"
	case "MEDIUM_FAST":
		return "MediumFast"
	case "SHORT_SLOW":
		return "ShortSlow"
	case "SHORT_FAST":
		return "ShortFast"
	case "LONG_MODERATE":
		return "LongModerate"
	case "SHORT_TURBO":
		return "ShortTurbo"
	case "LONG_TURBO":
		return "LongTurbo"
	default:
		return "LongFast"
	}
}

func decodeNodeFromPacketPayload(packet *generated.MeshPacket, payload []byte, now time.Time) (domain.NodeUpdate, bool) {
	if packet.GetFrom() == 0 {
		return domain.NodeUpdate{}, false
	}

	var user generated.User
	if err := proto.Unmarshal(payload, &user); err != nil {
		return domain.NodeUpdate{}, false
	}

	node := domain.Node{
		NodeID:      formatNodeNum(packet.GetFrom()),
		Channel:     uint32Ptr(packet.GetChannel()),
		LongName:    strings.TrimSpace(user.GetLongName()),
		ShortName:   strings.TrimSpace(user.GetShortName()),
		LastHeardAt: packetTimestamp(packet.GetRxTime(), now),
		UpdatedAt:   now,
	}
	if model := user.GetHwModel(); model != generated.HardwareModel_UNSET {
		node.BoardModel = model.String()
	}
	if role := strings.TrimSpace(user.GetRole().String()); role != "" {
		node.Role = role
	}
	if user.IsUnmessagable != nil {
		v := user.GetIsUnmessagable()
		node.IsUnmessageable = &v
	}
	if rssi := packet.GetRxRssi(); rssi != 0 {
		rssiVal := int(rssi)
		node.RSSI = &rssiVal
	}
	if snr := packet.GetRxSnr(); snr != 0 {
		snrVal := float64(snr)
		node.SNR = &snrVal
	}

	return domain.NodeUpdate{
		Node:       node,
		LastHeard:  node.LastHeardAt,
		FromPacket: true,
		Type:       domain.NodeUpdateTypeNodeInfoPacket,
	}, true
}

func decodeNodePositionFromPacket(packet *generated.MeshPacket, payload []byte, now time.Time) (domain.NodeUpdate, bool) {
	if packet.GetFrom() == 0 {
		return domain.NodeUpdate{}, false
	}

	var position generated.Position
	if err := proto.Unmarshal(payload, &position); err != nil {
		return domain.NodeUpdate{}, false
	}
	node := domain.Node{
		NodeID:      formatNodeNum(packet.GetFrom()),
		Channel:     uint32Ptr(packet.GetChannel()),
		LastHeardAt: packetTimestamp(packet.GetRxTime(), now),
		UpdatedAt:   now,
	}
	if !applyPositionCoordinates(&node, &position) {
		return domain.NodeUpdate{}, false
	}
	if rssi := packet.GetRxRssi(); rssi != 0 {
		rssiVal := int(rssi)
		node.RSSI = &rssiVal
	}
	if snr := packet.GetRxSnr(); snr != 0 {
		snrVal := float64(snr)
		node.SNR = &snrVal
	}

	return domain.NodeUpdate{
		Node:       node,
		LastHeard:  node.LastHeardAt,
		FromPacket: true,
		Type:       domain.NodeUpdateTypePositionPacket,
	}, true
}

func isValidNodeCoordinate(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func applyPositionCoordinates(node *domain.Node, position *generated.Position) bool {
	if node == nil || position == nil || position.LatitudeI == nil || position.LongitudeI == nil {
		return false
	}

	lat := float64(position.GetLatitudeI()) * meshtasticPositionScale
	lon := float64(position.GetLongitudeI()) * meshtasticPositionScale
	if !isValidNodeCoordinate(lat, lon) {
		return false
	}

	node.Latitude = &lat
	node.Longitude = &lon

	return true
}

func applyDeviceMetrics(node *domain.Node, dm *generated.DeviceMetrics) {
	if dm == nil || node == nil {
		return
	}
	if dm.BatteryLevel != nil {
		v := dm.GetBatteryLevel()
		node.BatteryLevel = &v
	}
	if dm.Voltage != nil {
		v := float64(dm.GetVoltage())
		node.Voltage = &v
	}
}

func applyEnvironmentMetrics(node *domain.Node, env *generated.EnvironmentMetrics) {
	if env == nil || node == nil {
		return
	}
	if env.Temperature != nil {
		v := float64(env.GetTemperature())
		node.Temperature = &v
	}
	if env.RelativeHumidity != nil {
		v := float64(env.GetRelativeHumidity())
		node.Humidity = &v
	}
	if env.BarometricPressure != nil {
		v := float64(env.GetBarometricPressure())
		node.Pressure = &v
	}
	if env.Iaq != nil {
		v := float64(env.GetIaq())
		node.AirQualityIndex = &v
	}
	// Some older telemetry reports power metrics in environment payload.
	if env.Voltage != nil {
		v := float64(env.GetVoltage())
		node.PowerVoltage = &v
		if node.Voltage == nil {
			node.Voltage = &v
		}
	}
	if env.Current != nil {
		v := float64(env.GetCurrent())
		node.PowerCurrent = &v
	}
}

func applyPowerMetrics(node *domain.Node, power *generated.PowerMetrics) {
	if power == nil || node == nil {
		return
	}
	if power.Ch1Voltage != nil {
		v := float64(power.GetCh1Voltage())
		node.PowerVoltage = &v
		if node.Voltage == nil {
			node.Voltage = &v
		}
	}
	if power.Ch1Current != nil {
		v := float64(power.GetCh1Current())
		node.PowerCurrent = &v
	}
}

func applyAirQualityMetrics(node *domain.Node, aq *generated.AirQualityMetrics) {
	if aq == nil || node == nil {
		return
	}
	if node.AirQualityIndex == nil && aq.PmVocIdx != nil {
		v := float64(aq.GetPmVocIdx())
		node.AirQualityIndex = &v
	}
}

func parseChatTarget(chatKey string) (to uint32, channel uint32, err error) {
	chatKey = strings.TrimSpace(chatKey)
	switch {
	case strings.HasPrefix(chatKey, "channel:"):
		idx, parseErr := strconv.ParseUint(strings.TrimPrefix(chatKey, "channel:"), 10, 32)
		if parseErr != nil || idx > math.MaxUint32 {
			return 0, 0, fmt.Errorf("invalid channel chat key: %q", chatKey)
		}

		return broadcastNodeNum, uint32(idx), nil
	case strings.HasPrefix(chatKey, "dm:"):
		nodeNum, parseErr := parseNodeNum(strings.TrimPrefix(chatKey, "dm:"))
		if parseErr != nil {
			return 0, 0, fmt.Errorf("invalid dm chat key: %q", chatKey)
		}

		return nodeNum, 0, nil
	default:
		return 0, 0, fmt.Errorf("unsupported chat key: %q", chatKey)
	}
}

func parseNodeNum(raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("node id is empty")
	}
	if strings.HasPrefix(raw, "!") {
		v, err := strconv.ParseUint(strings.TrimPrefix(raw, "!"), 16, 32)
		if err != nil {
			return 0, err
		}

		return uint32(v), nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "0x") {
		v, err := strconv.ParseUint(raw, 0, 32)
		if err != nil {
			return 0, err
		}

		return uint32(v), nil
	}
	if strings.IndexFunc(raw, func(r rune) bool {
		return (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
	}) >= 0 {
		v, err := strconv.ParseUint(raw, 16, 32)
		if err != nil {
			return 0, err
		}

		return uint32(v), nil
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(v), nil
}

func chatKeyForPacket(packet *generated.MeshPacket, direction domain.MessageDirection) string {
	if packet.GetTo() == broadcastNodeNum {
		return domain.ChatKeyForChannel(int(packet.GetChannel()))
	}
	if direction == domain.MessageDirectionOut {
		if packet.GetTo() != 0 {
			return domain.ChatKeyForDM(formatNodeNum(packet.GetTo()))
		}
	}
	if packet.GetFrom() != 0 {
		return domain.ChatKeyForDM(formatNodeNum(packet.GetFrom()))
	}
	if packet.GetTo() != 0 {
		return domain.ChatKeyForDM(formatNodeNum(packet.GetTo()))
	}

	return domain.ChatKeyForDM("unknown")
}

func formatNodeNum(num uint32) string {
	if num == 0 {
		return "unknown"
	}

	return fmt.Sprintf("!%08x", num)
}

func packetTimestamp(epochSec uint32, fallback time.Time) time.Time {
	if epochSec == 0 {
		return fallback
	}

	return time.Unix(int64(epochSec), 0)
}

func packetMetaJSON(port generated.PortNum, packet *generated.MeshPacket) string {
	meta := map[string]any{
		"codec":     "meshtastic-proto",
		"portnum":   port.String(),
		"from":      formatNodeNum(packet.GetFrom()),
		"to":        formatNodeNum(packet.GetTo()),
		"channel":   packet.GetChannel(),
		"packet_id": packet.GetId(),
		"transport": packet.GetTransportMechanism().String(),
	}
	if hops, ok := packetHops(packet); ok {
		meta["hops"] = hops
	}
	if hopStart := packet.GetHopStart(); hopStart != 0 {
		meta["hop_start"] = hopStart
	}
	if hopLimit := packet.GetHopLimit(); hopLimit != 0 {
		meta["hop_limit"] = hopLimit
	}
	if rxRssi := packet.GetRxRssi(); rxRssi != 0 {
		meta["rx_rssi"] = int(rxRssi)
	}
	if rxSnr := packet.GetRxSnr(); rxSnr != 0 {
		meta["rx_snr"] = float64(rxSnr)
	}
	if packet.GetViaMqtt() {
		meta["via_mqtt"] = true
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return ""
	}

	return string(raw)
}

func packetHops(packet *generated.MeshPacket) (int, bool) {
	hopStart := packet.GetHopStart()
	hopLimit := packet.GetHopLimit()
	if hopStart == 0 && hopLimit == 0 {
		return 0, false
	}
	if hopStart < hopLimit {
		return 0, false
	}

	return int(hopStart - hopLimit), true
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

func (c *MeshtasticCodec) nextNonZeroID() uint32 {
	for {
		id := c.packetID.Add(1)
		if id != 0 {
			return id
		}
	}
}
