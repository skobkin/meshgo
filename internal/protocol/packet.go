package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type PortNum int32

const (
	PortUnknown         PortNum = 0
	PortTextMessageApp  PortNum = 1
	PortRemoteHardware  PortNum = 2
	PortPositionApp     PortNum = 3
	PortNodeInfoApp     PortNum = 4
	PortRoutingApp      PortNum = 10
	PortAdminApp        PortNum = 11
	PortReplyApp        PortNum = 32
	PortIpTunnelApp     PortNum = 33
	PortPaxCounterApp   PortNum = 34
	PortSerialApp       PortNum = 64
	PortStoreForwardApp PortNum = 65
	PortRangeTestApp    PortNum = 66
	PortTelemetryApp    PortNum = 67
	PortZpsApp          PortNum = 68
	PortSimulatorApp    PortNum = 69
	PortTracerouteApp   PortNum = 70
	PortNeighborInfoApp PortNum = 71
	PortAudioApp        PortNum = 72
	PortDetectionSensor PortNum = 73
)

type MeshPacket struct {
	From       uint32      `json:"from"`
	To         uint32      `json:"to"`
	Channel    uint8       `json:"channel"`
	ID         uint32      `json:"id"`
	RxTime     uint32      `json:"rx_time"`
	RxSNR      float32     `json:"rx_snr"`
	RxRSSI     int32       `json:"rx_rssi"`
	HopLimit   uint32      `json:"hop_limit"`
	WantAck    bool        `json:"want_ack"`
	Priority   Priority    `json:"priority"`
	Payload    []byte      `json:"payload"`
	PortNum    PortNum     `json:"portnum"`
	Decoded    interface{} `json:"decoded,omitempty"`
}

type Priority int32

const (
	Unset Priority = iota
	Min
	Background
	Default
	Reliable
	Ack
	Max
)

type NodeInfo struct {
	Num           uint32         `json:"num"`
	User          *User          `json:"user,omitempty"`
	Position      *Position      `json:"position,omitempty"`
	SNR           float32        `json:"snr"`
	LastHeard     uint32         `json:"last_heard"`
	DeviceMetrics *DeviceMetrics `json:"device_metrics,omitempty"`
}

type User struct {
	ID         string `json:"id"`
	LongName   string `json:"long_name"`
	ShortName  string `json:"short_name"`
	MacAddr    []byte `json:"mac_addr"`
	HWModel    int32  `json:"hw_model"`
	IsLicensed bool   `json:"is_licensed"`
}

type Position struct {
	LatitudeI      int32   `json:"latitude_i"`
	LongitudeI     int32   `json:"longitude_i"`
	Altitude       int32   `json:"altitude"`
	Time           uint32  `json:"time"`
	LocationSource int32   `json:"location_source"`
	AltitudeSource int32   `json:"altitude_source"`
	GPSAccuracy    float32 `json:"gps_accuracy"`
}

func (p *Position) Latitude() float64 {
	return float64(p.LatitudeI) * 1e-7
}

func (p *Position) Longitude() float64 {
	return float64(p.LongitudeI) * 1e-7
}

type DeviceMetrics struct {
	BatteryLevel uint32  `json:"battery_level"`
	Voltage      float32 `json:"voltage"`
}

func (dm *DeviceMetrics) IsCharging() bool {
	return dm.BatteryLevel > 100
}

type TextMessage struct {
	Text string `json:"text"`
}

type ChannelSettings struct {
	PSK             []byte `json:"psk"`
	Name            string `json:"name"`
	ID              uint32 `json:"id"`
	UplinkEnabled   bool   `json:"uplink_enabled"`
	DownlinkEnabled bool   `json:"downlink_enabled"`
}

type RouteDiscovery struct {
	Route []uint32 `json:"route"`
}

type Routing struct {
	ErrorReason   int32           `json:"error_reason,omitempty"`
	RouteRequest  *RouteDiscovery `json:"route_request,omitempty"`
	RouteReply    *RouteDiscovery `json:"route_reply,omitempty"`
}

func EncodeMeshPacket(packet *MeshPacket) ([]byte, error) {
	if packet == nil {
		return nil, errors.New("packet is nil")
	}

	// Simple encoding for demonstration
	// In a real implementation, this would use protobuf
	buf := make([]byte, 0, 256)
	
	// Header: from, to, channel, id
	header := make([]byte, 20)
	binary.LittleEndian.PutUint32(header[0:4], packet.From)
	binary.LittleEndian.PutUint32(header[4:8], packet.To)
	header[8] = packet.Channel
	binary.LittleEndian.PutUint32(header[9:13], packet.ID)
	binary.LittleEndian.PutUint32(header[13:17], uint32(packet.PortNum))
	
	if packet.WantAck {
		header[17] = 1
	}
	binary.LittleEndian.PutUint16(header[18:20], uint16(len(packet.Payload)))
	
	buf = append(buf, header...)
	buf = append(buf, packet.Payload...)
	
	return buf, nil
}

func DecodeMeshPacket(data []byte) (*MeshPacket, error) {
	if len(data) < 20 {
		return nil, errors.New("packet too short")
	}

	packet := &MeshPacket{}
	
	// Parse header
	packet.From = binary.LittleEndian.Uint32(data[0:4])
	packet.To = binary.LittleEndian.Uint32(data[4:8])
	packet.Channel = data[8]
	packet.ID = binary.LittleEndian.Uint32(data[9:13])
	packet.PortNum = PortNum(binary.LittleEndian.Uint32(data[13:17]))
	packet.WantAck = data[17] == 1
	payloadLen := binary.LittleEndian.Uint16(data[18:20])
	
	if len(data) < 20+int(payloadLen) {
		return nil, errors.New("payload length mismatch")
	}
	
	packet.Payload = data[20 : 20+payloadLen]
	
	// Decode payload based on port number
	var err error
	packet.Decoded, err = decodePayload(packet.PortNum, packet.Payload)
	if err != nil {
		// Don't fail on decode errors, just log and continue
		packet.Decoded = nil
	}
	
	return packet, nil
}

func decodePayload(portNum PortNum, payload []byte) (interface{}, error) {
	switch portNum {
	case PortTextMessageApp:
		if len(payload) == 0 {
			return nil, errors.New("empty text message payload")
		}
		return &TextMessage{Text: string(payload)}, nil
		
	case PortNodeInfoApp:
		return decodeNodeInfo(payload)
		
	case PortPositionApp:
		return decodePosition(payload)
		
	case PortRoutingApp:
		return decodeRouting(payload)
		
	default:
		return nil, fmt.Errorf("unsupported port number: %d", portNum)
	}
}

func decodeNodeInfo(payload []byte) (*NodeInfo, error) {
	// Simplified NodeInfo decoding
	if len(payload) < 8 {
		return nil, errors.New("invalid NodeInfo payload")
	}
	
	nodeInfo := &NodeInfo{
		Num:       binary.LittleEndian.Uint32(payload[0:4]),
		LastHeard: binary.LittleEndian.Uint32(payload[4:8]),
	}
	
	// In real implementation, would properly decode User and other fields
	return nodeInfo, nil
}

func decodePosition(payload []byte) (*Position, error) {
	// Simplified Position decoding
	if len(payload) < 12 {
		return nil, errors.New("invalid Position payload")
	}
	
	position := &Position{
		LatitudeI:  int32(binary.LittleEndian.Uint32(payload[0:4])),
		LongitudeI: int32(binary.LittleEndian.Uint32(payload[4:8])),
		Altitude:   int32(binary.LittleEndian.Uint32(payload[8:12])),
	}
	
	if len(payload) >= 16 {
		position.Time = binary.LittleEndian.Uint32(payload[12:16])
	}
	
	return position, nil
}

func decodeRouting(payload []byte) (*Routing, error) {
	// Simplified Routing decoding
	if len(payload) < 4 {
		return nil, errors.New("invalid Routing payload")
	}
	
	routing := &Routing{}
	
	// This would be much more complex in a real implementation
	// For now, just return a placeholder
	return routing, nil
}

func EncodeTextMessage(text string) []byte {
	return []byte(text)
}

func EncodeNodeInfoRequest() []byte {
	// In real implementation, would encode proper NodeInfo request protobuf
	return []byte{0x01} // Simple request marker
}

func EncodeTracerouteRequest(destination uint32) []byte {
	// In real implementation, would encode proper traceroute protobuf
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, destination)
	return buf
}

func IsChannelMessage(packet *MeshPacket) bool {
	return packet.To == 0xFFFFFFFF // Broadcast to all nodes
}

func IsDMMessage(packet *MeshPacket) bool {
	return packet.To != 0xFFFFFFFF // Direct message to specific node
}