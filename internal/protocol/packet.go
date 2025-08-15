package protocol

import (
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// PortNum represents application port numbers for routing messages
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

// Priority represents message priority levels
type Priority int32

const (
	PriorityUnset      Priority = 0
	PriorityMin        Priority = 1
	PriorityBackground Priority = 10
	PriorityDefault    Priority = 64
	PriorityReliable   Priority = 70
	PriorityAck        Priority = 120
	PriorityMax        Priority = 127
)

// MeshPacket represents a Meshtastic protocol packet
type MeshPacket struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	From     uint32   `protobuf:"fixed32,1,opt,name=from,proto3" json:"from,omitempty"`
	To       uint32   `protobuf:"fixed32,2,opt,name=to,proto3" json:"to,omitempty"`
	Channel  uint32   `protobuf:"varint,3,opt,name=channel,proto3" json:"channel,omitempty"`
	Id       uint32   `protobuf:"fixed32,6,opt,name=id,proto3" json:"id,omitempty"`
	RxTime   uint32   `protobuf:"fixed32,7,opt,name=rx_time,json=rxTime,proto3" json:"rx_time,omitempty"`
	RxSnr    float32  `protobuf:"fixed32,8,opt,name=rx_snr,json=rxSnr,proto3" json:"rx_snr,omitempty"`
	HopLimit uint32   `protobuf:"varint,9,opt,name=hop_limit,json=hopLimit,proto3" json:"hop_limit,omitempty"`
	WantAck  bool     `protobuf:"varint,10,opt,name=want_ack,json=wantAck,proto3" json:"want_ack,omitempty"`
	Priority Priority `protobuf:"varint,11,opt,name=priority,proto3,enum=Priority" json:"priority,omitempty"`
	RxRssi   int32    `protobuf:"varint,12,opt,name=rx_rssi,json=rxRssi,proto3" json:"rx_rssi,omitempty"`

	// PayloadVariant is one of:
	//  - Decoded: unencrypted Data message
	//  - Encrypted: encrypted bytes
	//
	// Types that are assignable to PayloadVariant:
	//	*MeshPacket_Decoded
	//	*MeshPacket_Encrypted
	PayloadVariant isMeshPacket_PayloadVariant `protobuf_oneof:"payload_variant"`
}

func (x *MeshPacket) Reset() {
	*x = MeshPacket{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MeshPacket) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MeshPacket) ProtoMessage() {}

func (x *MeshPacket) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *MeshPacket) GetFrom() uint32 {
	if x != nil {
		return x.From
	}
	return 0
}

func (x *MeshPacket) GetTo() uint32 {
	if x != nil {
		return x.To
	}
	return 0
}

func (x *MeshPacket) GetChannel() uint32 {
	if x != nil {
		return x.Channel
	}
	return 0
}

func (x *MeshPacket) GetDecoded() *Data {
	if x, ok := x.GetPayloadVariant().(*MeshPacket_Decoded); ok {
		return x.Decoded
	}
	return nil
}

func (x *MeshPacket) GetEncrypted() []byte {
	if x, ok := x.GetPayloadVariant().(*MeshPacket_Encrypted); ok {
		return x.Encrypted
	}
	return nil
}

func (x *MeshPacket) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *MeshPacket) GetRxTime() uint32 {
	if x != nil {
		return x.RxTime
	}
	return 0
}

func (x *MeshPacket) GetRxSnr() float32 {
	if x != nil {
		return x.RxSnr
	}
	return 0
}

func (x *MeshPacket) GetHopLimit() uint32 {
	if x != nil {
		return x.HopLimit
	}
	return 0
}

func (x *MeshPacket) GetWantAck() bool {
	if x != nil {
		return x.WantAck
	}
	return false
}

func (x *MeshPacket) GetPriority() Priority {
	if x != nil {
		return x.Priority
	}
	return PriorityUnset
}

func (x *MeshPacket) GetRxRssi() int32 {
	if x != nil {
		return x.RxRssi
	}
	return 0
}

type isMeshPacket_PayloadVariant interface {
	isMeshPacket_PayloadVariant()
}

type MeshPacket_Decoded struct {
	Decoded *Data `protobuf:"bytes,4,opt,name=decoded,proto3,oneof"`
}

type MeshPacket_Encrypted struct {
	Encrypted []byte `protobuf:"bytes,5,opt,name=encrypted,proto3,oneof"`
}

func (*MeshPacket_Decoded) isMeshPacket_PayloadVariant() {}

func (*MeshPacket_Encrypted) isMeshPacket_PayloadVariant() {}

func (x *MeshPacket) GetPayloadVariant() isMeshPacket_PayloadVariant {
	if x != nil {
		return x.PayloadVariant
	}
	return nil
}

// Data represents the decoded application payload
type Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Portnum      PortNum `protobuf:"varint,1,opt,name=portnum,proto3,enum=PortNum" json:"portnum,omitempty"`
	Payload      []byte  `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	WantResponse bool    `protobuf:"varint,3,opt,name=want_response,json=wantResponse,proto3" json:"want_response,omitempty"`
	Dest         uint32  `protobuf:"fixed32,4,opt,name=dest,proto3" json:"dest,omitempty"`
	Source       uint32  `protobuf:"fixed32,5,opt,name=source,proto3" json:"source,omitempty"`
	RequestId    uint32  `protobuf:"fixed32,6,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"`
	ReplyId      uint32  `protobuf:"fixed32,7,opt,name=reply_id,json=replyId,proto3" json:"reply_id,omitempty"`
	Emoji        uint32  `protobuf:"fixed32,8,opt,name=emoji,proto3" json:"emoji,omitempty"`
	Bitfield     uint32  `protobuf:"varint,9,opt,name=bitfield,proto3" json:"bitfield,omitempty"`
}

func (x *Data) Reset() {
	*x = Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Data) ProtoMessage() {}

func (x *Data) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Data) GetPortnum() PortNum {
	if x != nil {
		return x.Portnum
	}
	return PortUnknown
}

func (x *Data) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *Data) GetWantResponse() bool {
	if x != nil {
		return x.WantResponse
	}
	return false
}

// Application-specific message types

// User represents a user/node identity
type User struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	LongName   string `protobuf:"bytes,2,opt,name=long_name,json=longName,proto3" json:"long_name,omitempty"`
	ShortName  string `protobuf:"bytes,3,opt,name=short_name,json=shortName,proto3" json:"short_name,omitempty"`
	Macaddr    []byte `protobuf:"bytes,4,opt,name=macaddr,proto3" json:"macaddr,omitempty"`
	HwModel    int32  `protobuf:"varint,5,opt,name=hw_model,json=hwModel,proto3" json:"hw_model,omitempty"`
	IsLicensed bool   `protobuf:"varint,6,opt,name=is_licensed,json=isLicensed,proto3" json:"is_licensed,omitempty"`
}

func (x *User) Reset() {
	*x = User{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *User) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *User) GetLongName() string {
	if x != nil {
		return x.LongName
	}
	return ""
}

func (x *User) GetShortName() string {
	if x != nil {
		return x.ShortName
	}
	return ""
}

// Position represents geographical location data
type Position struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LatitudeI      int32   `protobuf:"sfixed32,1,opt,name=latitude_i,json=latitudeI,proto3" json:"latitude_i,omitempty"`
	LongitudeI     int32   `protobuf:"sfixed32,2,opt,name=longitude_i,json=longitudeI,proto3" json:"longitude_i,omitempty"`
	Altitude       int32   `protobuf:"sfixed32,3,opt,name=altitude,proto3" json:"altitude,omitempty"`
	Time           uint32  `protobuf:"fixed32,4,opt,name=time,proto3" json:"time,omitempty"`
	LocationSource int32   `protobuf:"varint,5,opt,name=location_source,json=locationSource,proto3" json:"location_source,omitempty"`
	AltitudeSource int32   `protobuf:"varint,6,opt,name=altitude_source,json=altitudeSource,proto3" json:"altitude_source,omitempty"`
	GpsAccuracy    float32 `protobuf:"fixed32,7,opt,name=gps_accuracy,json=gpsAccuracy,proto3" json:"gps_accuracy,omitempty"`
}

func (x *Position) Reset() {
	*x = Position{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Position) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Position) ProtoMessage() {}

func (x *Position) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *Position) GetLatitudeI() int32 {
	if x != nil {
		return x.LatitudeI
	}
	return 0
}

func (x *Position) GetLongitudeI() int32 {
	if x != nil {
		return x.LongitudeI
	}
	return 0
}

func (x *Position) GetAltitude() int32 {
	if x != nil {
		return x.Altitude
	}
	return 0
}

func (x *Position) GetTime() uint32 {
	if x != nil {
		return x.Time
	}
	return 0
}

// Convenience methods for Position
func (p *Position) Latitude() float64 {
	return float64(p.LatitudeI) * 1e-7
}

func (p *Position) Longitude() float64 {
	return float64(p.LongitudeI) * 1e-7
}

// DeviceMetrics represents device telemetry
type DeviceMetrics struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BatteryLevel uint32  `protobuf:"varint,1,opt,name=battery_level,json=batteryLevel,proto3" json:"battery_level,omitempty"`
	Voltage      float32 `protobuf:"fixed32,2,opt,name=voltage,proto3" json:"voltage,omitempty"`
}

func (x *DeviceMetrics) Reset() {
	*x = DeviceMetrics{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeviceMetrics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeviceMetrics) ProtoMessage() {}

func (x *DeviceMetrics) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *DeviceMetrics) GetBatteryLevel() uint32 {
	if x != nil {
		return x.BatteryLevel
	}
	return 0
}

func (x *DeviceMetrics) GetVoltage() float32 {
	if x != nil {
		return x.Voltage
	}
	return 0
}

func (dm *DeviceMetrics) IsCharging() bool {
	return dm.BatteryLevel > 100
}

// NodeInfo represents complete node information
type NodeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Num           uint32         `protobuf:"varint,1,opt,name=num,proto3" json:"num,omitempty"`
	User          *User          `protobuf:"bytes,2,opt,name=user,proto3" json:"user,omitempty"`
	Position      *Position      `protobuf:"bytes,3,opt,name=position,proto3" json:"position,omitempty"`
	Snr           float32        `protobuf:"fixed32,4,opt,name=snr,proto3" json:"snr,omitempty"`
	LastHeard     uint32         `protobuf:"fixed32,5,opt,name=last_heard,json=lastHeard,proto3" json:"last_heard,omitempty"`
	DeviceMetrics *DeviceMetrics `protobuf:"bytes,6,opt,name=device_metrics,json=deviceMetrics,proto3" json:"device_metrics,omitempty"`
}

func (x *NodeInfo) Reset() {
	*x = NodeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NodeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NodeInfo) ProtoMessage() {}

func (x *NodeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *NodeInfo) GetNum() uint32 {
	if x != nil {
		return x.Num
	}
	return 0
}

func (x *NodeInfo) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

func (x *NodeInfo) GetPosition() *Position {
	if x != nil {
		return x.Position
	}
	return nil
}

func (x *NodeInfo) GetSnr() float32 {
	if x != nil {
		return x.Snr
	}
	return 0
}

func (x *NodeInfo) GetLastHeard() uint32 {
	if x != nil {
		return x.LastHeard
	}
	return 0
}

func (x *NodeInfo) GetDeviceMetrics() *DeviceMetrics {
	if x != nil {
		return x.DeviceMetrics
	}
	return nil
}

// ToRadio/FromRadio protocol messages for client-device communication

// ToRadio represents messages sent TO the radio device
type ToRadio struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// PayloadVariant is one of:
	//  - Packet: MeshPacket to send
	//  - WantConfigId: request configuration
	//
	// Types that are assignable to PayloadVariant:
	//	*ToRadio_Packet
	//	*ToRadio_WantConfigId
	PayloadVariant isToRadio_PayloadVariant `protobuf_oneof:"payload_variant"`
}

func (x *ToRadio) Reset() {
	*x = ToRadio{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ToRadio) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ToRadio) ProtoMessage() {}

func (x *ToRadio) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *ToRadio) GetPacket() *MeshPacket {
	if x, ok := x.GetPayloadVariant().(*ToRadio_Packet); ok {
		return x.Packet
	}
	return nil
}

func (x *ToRadio) GetWantConfigId() uint32 {
	if x, ok := x.GetPayloadVariant().(*ToRadio_WantConfigId); ok {
		return x.WantConfigId
	}
	return 0
}

type isToRadio_PayloadVariant interface {
	isToRadio_PayloadVariant()
}

type ToRadio_Packet struct {
	Packet *MeshPacket `protobuf:"bytes,1,opt,name=packet,proto3,oneof"`
}

type ToRadio_WantConfigId struct {
	WantConfigId uint32 `protobuf:"varint,3,opt,name=want_config_id,json=wantConfigId,proto3,oneof"`
}

func (*ToRadio_Packet) isToRadio_PayloadVariant() {}

func (*ToRadio_WantConfigId) isToRadio_PayloadVariant() {}

func (x *ToRadio) GetPayloadVariant() isToRadio_PayloadVariant {
	if x != nil {
		return x.PayloadVariant
	}
	return nil
}

// FromRadio represents messages received FROM the radio device
type FromRadio struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`

	// PayloadVariant is one of:
	//  - Packet: received MeshPacket
	//  - MyInfo: device information
	//  - NodeInfo: node database entry
	//  - ConfigCompleteId: end of config sequence
	//
	// Types that are assignable to PayloadVariant:
	//	*FromRadio_Packet
	//	*FromRadio_MyInfo
	//	*FromRadio_NodeInfo
	//	*FromRadio_ConfigCompleteId
	PayloadVariant isFromRadio_PayloadVariant `protobuf_oneof:"payload_variant"`
}

func (x *FromRadio) Reset() {
	*x = FromRadio{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FromRadio) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FromRadio) ProtoMessage() {}

func (x *FromRadio) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *FromRadio) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *FromRadio) GetPacket() *MeshPacket {
	if x, ok := x.GetPayloadVariant().(*FromRadio_Packet); ok {
		return x.Packet
	}
	return nil
}

func (x *FromRadio) GetMyInfo() *MyNodeInfo {
	if x, ok := x.GetPayloadVariant().(*FromRadio_MyInfo); ok {
		return x.MyInfo
	}
	return nil
}

func (x *FromRadio) GetNodeInfo() *NodeInfo {
	if x, ok := x.GetPayloadVariant().(*FromRadio_NodeInfo); ok {
		return x.NodeInfo
	}
	return nil
}

func (x *FromRadio) GetConfigCompleteId() uint32 {
	if x, ok := x.GetPayloadVariant().(*FromRadio_ConfigCompleteId); ok {
		return x.ConfigCompleteId
	}
	return 0
}

type isFromRadio_PayloadVariant interface {
	isFromRadio_PayloadVariant()
}

type FromRadio_Packet struct {
	Packet *MeshPacket `protobuf:"bytes,2,opt,name=packet,proto3,oneof"`
}

type FromRadio_MyInfo struct {
	MyInfo *MyNodeInfo `protobuf:"bytes,3,opt,name=my_info,json=myInfo,proto3,oneof"`
}

type FromRadio_NodeInfo struct {
	NodeInfo *NodeInfo `protobuf:"bytes,4,opt,name=node_info,json=nodeInfo,proto3,oneof"`
}

type FromRadio_ConfigCompleteId struct {
	ConfigCompleteId uint32 `protobuf:"varint,100,opt,name=config_complete_id,json=configCompleteId,proto3,oneof"`
}

func (*FromRadio_Packet) isFromRadio_PayloadVariant() {}

func (*FromRadio_MyInfo) isFromRadio_PayloadVariant() {}

func (*FromRadio_NodeInfo) isFromRadio_PayloadVariant() {}

func (*FromRadio_ConfigCompleteId) isFromRadio_PayloadVariant() {}

func (x *FromRadio) GetPayloadVariant() isFromRadio_PayloadVariant {
	if x != nil {
		return x.PayloadVariant
	}
	return nil
}

// MyNodeInfo represents information about the local device
type MyNodeInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MyNodeNum          uint32 `protobuf:"varint,1,opt,name=my_node_num,json=myNodeNum,proto3" json:"my_node_num,omitempty"`
	HasGps             bool   `protobuf:"varint,2,opt,name=has_gps,json=hasGps,proto3" json:"has_gps,omitempty"`
	NumChannels        int32  `protobuf:"varint,3,opt,name=num_channels,json=numChannels,proto3" json:"num_channels,omitempty"`
	Region             int32  `protobuf:"varint,4,opt,name=region,proto3" json:"region,omitempty"`
	HwModel            string `protobuf:"bytes,5,opt,name=hw_model,json=hwModel,proto3" json:"hw_model,omitempty"`
	FirmwareVersion    string `protobuf:"bytes,6,opt,name=firmware_version,json=firmwareVersion,proto3" json:"firmware_version,omitempty"`
	ErrorCode          int32  `protobuf:"varint,7,opt,name=error_code,json=errorCode,proto3" json:"error_code,omitempty"`
	ErrorAddress       uint32 `protobuf:"varint,8,opt,name=error_address,json=errorAddress,proto3" json:"error_address,omitempty"`
	ErrorCount         uint32 `protobuf:"varint,9,opt,name=error_count,json=errorCount,proto3" json:"error_count,omitempty"`
	PacketIdBits       uint32 `protobuf:"varint,11,opt,name=packet_id_bits,json=packetIdBits,proto3" json:"packet_id_bits,omitempty"`
	CurrentPacketId    uint32 `protobuf:"varint,12,opt,name=current_packet_id,json=currentPacketId,proto3" json:"current_packet_id,omitempty"`
	NodeNumBits        uint32 `protobuf:"varint,13,opt,name=node_num_bits,json=nodeNumBits,proto3" json:"node_num_bits,omitempty"`
	MessageTimeoutMsec uint32 `protobuf:"varint,14,opt,name=message_timeout_msec,json=messageTimeoutMsec,proto3" json:"message_timeout_msec,omitempty"`
	MinAppVersion      uint32 `protobuf:"varint,15,opt,name=min_app_version,json=minAppVersion,proto3" json:"min_app_version,omitempty"`
	MaxChannels        uint32 `protobuf:"varint,16,opt,name=max_channels,json=maxChannels,proto3" json:"max_channels,omitempty"`
}

func (x *MyNodeInfo) Reset() {
	*x = MyNodeInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_packet_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MyNodeInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MyNodeInfo) ProtoMessage() {}

func (x *MyNodeInfo) ProtoReflect() protoreflect.Message {
	mi := &file_packet_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (x *MyNodeInfo) GetMyNodeNum() uint32 {
	if x != nil {
		return x.MyNodeNum
	}
	return 0
}

func (x *MyNodeInfo) GetFirmwareVersion() string {
	if x != nil {
		return x.FirmwareVersion
	}
	return ""
}

func (x *MyNodeInfo) GetHwModel() string {
	if x != nil {
		return x.HwModel
	}
	return ""
}

// Simple message types

// TextMessage represents a text chat message
type TextMessage struct {
	Text string `json:"text"`
}

// Routing represents routing/traceroute messages
type Routing struct {
	ErrorReason  int32           `json:"error_reason,omitempty"`
	RouteRequest *RouteDiscovery `json:"route_request,omitempty"`
	RouteReply   *RouteDiscovery `json:"route_reply,omitempty"`
}

type RouteDiscovery struct {
	Route []uint32 `json:"route"`
}

// Encoding/Decoding Functions

func EncodeMeshPacket(packet *MeshPacket) ([]byte, error) {
	if packet == nil {
		return nil, errors.New("packet is nil")
	}
	return proto.Marshal(packet)
}

func DecodeMeshPacket(data []byte) (*MeshPacket, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}

	packet := &MeshPacket{}
	err := proto.Unmarshal(data, packet)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MeshPacket: %w", err)
	}

	return packet, nil
}

// Decode the payload based on the Data.Portnum field
func DecodePayload(data *Data) (interface{}, error) {
	if data == nil || len(data.Payload) == 0 {
		return nil, errors.New("empty payload")
	}

	switch data.Portnum {
	case PortTextMessageApp:
		// Text messages are typically just UTF-8 strings
		return &TextMessage{Text: string(data.Payload)}, nil

	case PortNodeInfoApp:
		nodeInfo := &NodeInfo{}
		if err := proto.Unmarshal(data.Payload, nodeInfo); err != nil {
			return nil, fmt.Errorf("failed to decode NodeInfo: %w", err)
		}
		return nodeInfo, nil

	case PortPositionApp:
		position := &Position{}
		if err := proto.Unmarshal(data.Payload, position); err != nil {
			return nil, fmt.Errorf("failed to decode Position: %w", err)
		}
		return position, nil

	case PortRoutingApp:
		// Simplified routing decode - in practice this has sub-messages
		routing := &Routing{}
		// For now, just return a placeholder as routing has complex sub-message types
		return routing, nil

	default:
		// Unknown port number - return raw payload
		return data.Payload, nil
	}
}

// Helper functions for encoding application messages

func EncodeTextMessage(text string) ([]byte, error) {
	data := &Data{
		Portnum: PortTextMessageApp,
		Payload: []byte(text),
	}
	return proto.Marshal(data)
}

func EncodeNodeInfoRequest() ([]byte, error) {
	// NodeInfo requests are typically empty payloads
	data := &Data{
		Portnum: PortNodeInfoApp,
		Payload: []byte{},
	}
	return proto.Marshal(data)
}

func EncodeTracerouteRequest(destination uint32) ([]byte, error) {
	// Simplified traceroute request
	data := &Data{
		Portnum: PortRoutingApp,
		Dest:    destination,
		Payload: []byte{0x01}, // Simple request marker
	}
	return proto.Marshal(data)
}

// Helper functions

func IsChannelMessage(packet *MeshPacket) bool {
	return packet.To == 0xFFFFFFFF // Broadcast to all nodes
}

func IsDMMessage(packet *MeshPacket) bool {
	return packet.To != 0xFFFFFFFF // Direct message to specific node
}

// Encoding/decoding functions for protocol messages

func EncodeToRadio(msg *ToRadio) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("ToRadio message is nil")
	}
	return proto.Marshal(msg)
}

func DecodeFromRadio(data []byte) (*FromRadio, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}

	msg := &FromRadio{}
	err := proto.Unmarshal(data, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal FromRadio: %w", err)
	}

	return msg, nil
}

// CreateStartConfigRequest creates a ToRadio message to request configuration
func CreateStartConfigRequest() *ToRadio {
	return &ToRadio{
		PayloadVariant: &ToRadio_WantConfigId{
			WantConfigId: 42, // Magic number for startConfig
		},
	}
}

// Protobuf message info (required for protobuf implementation)
var (
	file_packet_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
)

func init() {
	// Initialize message infos for protobuf - this is generated code normally
	// but for our hand-crafted structs we need to provide basic implementation
	file_packet_proto_msgTypes[0].GoReflectType = reflect.TypeOf((*MeshPacket)(nil))
	file_packet_proto_msgTypes[1].GoReflectType = reflect.TypeOf((*Data)(nil))
	file_packet_proto_msgTypes[2].GoReflectType = reflect.TypeOf((*User)(nil))
	file_packet_proto_msgTypes[3].GoReflectType = reflect.TypeOf((*Position)(nil))
	file_packet_proto_msgTypes[4].GoReflectType = reflect.TypeOf((*DeviceMetrics)(nil))
	file_packet_proto_msgTypes[5].GoReflectType = reflect.TypeOf((*NodeInfo)(nil))
	file_packet_proto_msgTypes[6].GoReflectType = reflect.TypeOf((*ToRadio)(nil))
	file_packet_proto_msgTypes[7].GoReflectType = reflect.TypeOf((*FromRadio)(nil))
	file_packet_proto_msgTypes[8].GoReflectType = reflect.TypeOf((*MyNodeInfo)(nil))
}
