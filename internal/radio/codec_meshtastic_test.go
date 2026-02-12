package radio

import (
	"math"
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

func mustNewMeshtasticCodec(t *testing.T) *MeshtasticCodec {
	t.Helper()

	codec, err := NewMeshtasticCodec()
	if err != nil {
		t.Fatalf("initialize codec: %v", err)
	}

	return codec
}

func TestMeshtasticCodec_EncodeTextIncludesDeviceMessageID(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)
	encoded, err := codec.EncodeText("dm:!1234abcd", "hello")
	if err != nil {
		t.Fatalf("encode text: %v", err)
	}
	if encoded.DeviceMessageID == "" {
		t.Fatalf("expected non-empty device message id")
	}
	if len(encoded.Payload) == 0 {
		t.Fatalf("expected non-empty payload")
	}
	if !encoded.WantAck {
		t.Fatalf("expected want_ack for direct message")
	}
}

func TestMeshtasticCodec_DecodeFromRadioTelemetryEnvironmentPacket(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	telemetryPayload, err := proto.Marshal(&generated.Telemetry{
		Variant: &generated.Telemetry_EnvironmentMetrics{
			EnvironmentMetrics: &generated.EnvironmentMetrics{
				Temperature:        proto.Float32(22.7),
				RelativeHumidity:   proto.Float32(47.3),
				BarometricPressure: proto.Float32(1008.6),
				Iaq:                proto.Uint32(92),
				Voltage:            proto.Float32(4.12),
				Current:            proto.Float32(0.137),
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal telemetry: %v", err)
	}

	raw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				From:   0x1234abcd,
				RxTime: 1_735_123_456,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum: generated.PortNum_TELEMETRY_APP,
						Payload: telemetryPayload,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode telemetry packet: %v", err)
	}
	if frame.NodeUpdate == nil {
		t.Fatalf("expected node update")
	}
	node := frame.NodeUpdate.Node
	if node.NodeID != "!1234abcd" {
		t.Fatalf("unexpected node id: %q", node.NodeID)
	}
	assertFloatPtr(t, node.Temperature, 22.7, "temperature")
	assertFloatPtr(t, node.Humidity, 47.3, "humidity")
	assertFloatPtr(t, node.Pressure, 1008.6, "pressure")
	assertFloatPtr(t, node.AirQualityIndex, 92.0, "air quality index")
	assertFloatPtr(t, node.PowerVoltage, 4.12, "power voltage")
	assertFloatPtr(t, node.PowerCurrent, 0.137, "power current")
}

func TestMeshtasticCodec_DecodeFromRadioTelemetryPowerPacket(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	telemetryPayload, err := proto.Marshal(&generated.Telemetry{
		Variant: &generated.Telemetry_PowerMetrics{
			PowerMetrics: &generated.PowerMetrics{
				Ch1Voltage: proto.Float32(12.34),
				Ch1Current: proto.Float32(1.25),
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal telemetry: %v", err)
	}

	raw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				From: 0x7654dcba,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum: generated.PortNum_TELEMETRY_APP,
						Payload: telemetryPayload,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode telemetry packet: %v", err)
	}
	if frame.NodeUpdate == nil {
		t.Fatalf("expected node update")
	}
	node := frame.NodeUpdate.Node
	assertFloatPtr(t, node.PowerVoltage, 12.34, "power voltage")
	assertFloatPtr(t, node.PowerCurrent, 1.25, "power current")
	assertFloatPtr(t, node.Voltage, 12.34, "voltage")
}

func TestMeshtasticCodec_DecodeFromRadioPositionPacket(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	positionPayload, err := proto.Marshal(&generated.Position{
		LatitudeI:  proto.Int32(37_774_9000),
		LongitudeI: proto.Int32(-122_419_4000),
	})
	if err != nil {
		t.Fatalf("marshal position: %v", err)
	}

	raw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				From:   0x1234abcd,
				RxTime: 1_735_123_456,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum: generated.PortNum_POSITION_APP,
						Payload: positionPayload,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode position packet: %v", err)
	}
	if frame.NodeUpdate == nil {
		t.Fatalf("expected node update")
	}
	assertFloatPtr(t, frame.NodeUpdate.Node.Latitude, 37.7749, "latitude")
	assertFloatPtr(t, frame.NodeUpdate.Node.Longitude, -122.4194, "longitude")
}

func TestMeshtasticCodec_DecodeFromRadioPositionPacketInvalidCoordinatesIgnored(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	positionPayload, err := proto.Marshal(&generated.Position{
		LatitudeI:  proto.Int32(910_000_000),
		LongitudeI: proto.Int32(0),
	})
	if err != nil {
		t.Fatalf("marshal position: %v", err)
	}

	raw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				From: 0x1234abcd,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum: generated.PortNum_POSITION_APP,
						Payload: positionPayload,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode position packet: %v", err)
	}
	if frame.NodeUpdate != nil {
		t.Fatalf("expected no node update for invalid coordinates")
	}
}

func TestMeshtasticCodec_DecodeFromRadioNodeInfoIncludesStaticPosition(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	raw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_NodeInfo{
			NodeInfo: &generated.NodeInfo{
				Num:       0x1234abcd,
				LastHeard: 1_735_123_456,
				User: &generated.User{
					LongName:  "Alpha",
					ShortName: "ALPH",
				},
				Position: &generated.Position{
					LatitudeI:  proto.Int32(37_774_9000),
					LongitudeI: proto.Int32(-122_419_4000),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode node info frame: %v", err)
	}
	if frame.NodeUpdate == nil {
		t.Fatalf("expected node update")
	}
	assertFloatPtr(t, frame.NodeUpdate.Node.Latitude, 37.7749, "latitude")
	assertFloatPtr(t, frame.NodeUpdate.Node.Longitude, -122.4194, "longitude")
}

func assertFloatPtr(t *testing.T, got *float64, want float64, field string) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected %s", field)
	}
	if math.Abs(*got-want) > 0.0001 {
		t.Fatalf("unexpected %s value: got %v want %v", field, *got, want)
	}
}

func TestMeshtasticCodec_DecodeFromRadioQueueStatusSuccess(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	wire := &generated.FromRadio{
		PayloadVariant: &generated.FromRadio_QueueStatus{
			QueueStatus: &generated.QueueStatus{
				MeshPacketId: 42,
				Res:          int32(generated.Routing_NONE),
			},
		},
	}
	raw, err := proto.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode fromradio: %v", err)
	}
	if frame.MessageStatus == nil {
		t.Fatalf("expected message status update")
	}
	if frame.MessageStatus.DeviceMessageID != "42" {
		t.Fatalf("unexpected device id: %q", frame.MessageStatus.DeviceMessageID)
	}
	if frame.MessageStatus.Status != domain.MessageStatusSent {
		t.Fatalf("unexpected status: %v", frame.MessageStatus.Status)
	}
}

func TestMeshtasticCodec_DecodeFromRadioQueueStatusFailure(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	wire := &generated.FromRadio{
		PayloadVariant: &generated.FromRadio_QueueStatus{
			QueueStatus: &generated.QueueStatus{
				MeshPacketId: 42,
				Res:          int32(generated.Routing_NO_ROUTE),
			},
		},
	}
	raw, err := proto.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode fromradio: %v", err)
	}
	if frame.MessageStatus == nil {
		t.Fatalf("expected message status update")
	}
	if frame.MessageStatus.DeviceMessageID != "42" {
		t.Fatalf("unexpected device id: %q", frame.MessageStatus.DeviceMessageID)
	}
	if frame.MessageStatus.Status != domain.MessageStatusFailed {
		t.Fatalf("unexpected status: %v", frame.MessageStatus.Status)
	}
	if frame.MessageStatus.Reason != "NO_ROUTE" {
		t.Fatalf("unexpected reason: %q", frame.MessageStatus.Reason)
	}
}

func TestMeshtasticCodec_DecodeFromRadioAckPacket(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	wire := &generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				Priority: generated.MeshPacket_ACK,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum:   generated.PortNum_TEXT_MESSAGE_APP,
						RequestId: 777,
					},
				},
			},
		},
	}
	raw, err := proto.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode fromradio: %v", err)
	}
	if frame.MessageStatus == nil {
		t.Fatalf("expected message status update")
	}
	if frame.MessageStatus.DeviceMessageID != "777" {
		t.Fatalf("unexpected device id: %q", frame.MessageStatus.DeviceMessageID)
	}
	if frame.MessageStatus.Status != domain.MessageStatusAcked {
		t.Fatalf("unexpected status: %v", frame.MessageStatus.Status)
	}
}

func TestMeshtasticCodec_DecodeFromRadioRoutingError(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	routingPayload, err := proto.Marshal(&generated.Routing{
		Variant: &generated.Routing_ErrorReason{ErrorReason: generated.Routing_NO_ROUTE},
	})
	if err != nil {
		t.Fatalf("marshal routing: %v", err)
	}

	wire := &generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum:   generated.PortNum_ROUTING_APP,
						RequestId: 9001,
						Payload:   routingPayload,
					},
				},
			},
		},
	}
	raw, err := proto.Marshal(wire)
	if err != nil {
		t.Fatalf("marshal fromradio: %v", err)
	}

	frame, err := codec.DecodeFromRadio(raw)
	if err != nil {
		t.Fatalf("decode fromradio: %v", err)
	}
	if frame.MessageStatus == nil {
		t.Fatalf("expected message status update")
	}
	if frame.MessageStatus.DeviceMessageID != "9001" {
		t.Fatalf("unexpected device id: %q", frame.MessageStatus.DeviceMessageID)
	}
	if frame.MessageStatus.Status != domain.MessageStatusFailed {
		t.Fatalf("unexpected status: %v", frame.MessageStatus.Status)
	}
	if frame.MessageStatus.Reason != "NO_ROUTE" {
		t.Fatalf("unexpected reason: %q", frame.MessageStatus.Reason)
	}
}

func TestMeshtasticCodec_DecodeFromRadioLocalEchoIsPendingWhenWantAck(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	myInfoRaw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_MyInfo{
			MyInfo: &generated.MyNodeInfo{MyNodeNum: 123},
		},
	})
	if err != nil {
		t.Fatalf("marshal myinfo: %v", err)
	}
	if _, err := codec.DecodeFromRadio(myInfoRaw); err != nil {
		t.Fatalf("decode myinfo: %v", err)
	}

	packetRaw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Packet{
			Packet: &generated.MeshPacket{
				From:    123,
				To:      456,
				WantAck: true,
				PayloadVariant: &generated.MeshPacket_Decoded{
					Decoded: &generated.Data{
						Portnum: generated.PortNum_TEXT_MESSAGE_APP,
						Payload: []byte("hello"),
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal packet: %v", err)
	}

	frame, err := codec.DecodeFromRadio(packetRaw)
	if err != nil {
		t.Fatalf("decode packet: %v", err)
	}
	if frame.TextMessage == nil {
		t.Fatalf("expected text message")
	}
	if frame.TextMessage.Direction != domain.MessageDirectionOut {
		t.Fatalf("expected outgoing direction, got %v", frame.TextMessage.Direction)
	}
	if frame.TextMessage.Status != domain.MessageStatusPending {
		t.Fatalf("expected pending status, got %v", frame.TextMessage.Status)
	}
}

func TestDecodeChannelInfo_EmptyPrimaryUsesDefaultTitle(t *testing.T) {
	channel := &generated.Channel{
		Index: 1,
		Role:  generated.Channel_PRIMARY,
		Settings: &generated.ChannelSettings{
			Name: "",
		},
	}

	channels, _, ok := decodeChannelInfo(channel, "LongFast")
	if !ok {
		t.Fatalf("expected channel to be decoded")
	}
	if len(channels.Items) != 1 {
		t.Fatalf("expected one channel item, got %d", len(channels.Items))
	}
	if channels.Items[0].Title != "LongFast" {
		t.Fatalf("expected primary fallback title LongFast, got %q", channels.Items[0].Title)
	}
}

func TestDecodeChannelInfo_EmptySecondaryUsesDefaultTitle(t *testing.T) {
	channel := &generated.Channel{
		Index: 2,
		Role:  generated.Channel_SECONDARY,
		Settings: &generated.ChannelSettings{
			Name: "",
			Psk:  []byte{1},
		},
	}

	channels, _, ok := decodeChannelInfo(channel, "LongFast")
	if !ok {
		t.Fatalf("expected channel to be decoded")
	}
	if len(channels.Items) != 1 {
		t.Fatalf("expected one channel item, got %d", len(channels.Items))
	}
	if channels.Items[0].Title != "LongFast" {
		t.Fatalf("expected secondary fallback title LongFast, got %q", channels.Items[0].Title)
	}
}

func TestMeshtasticCodec_DecodeFromRadioConfigPresetAffectsEmptyPrimaryName(t *testing.T) {
	codec := mustNewMeshtasticCodec(t)

	configRaw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Config{
			Config: &generated.Config{
				PayloadVariant: &generated.Config_Lora{
					Lora: &generated.Config_LoRaConfig{
						ModemPreset: generated.Config_LoRaConfig_MEDIUM_FAST,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal config frame: %v", err)
	}
	if _, err := codec.DecodeFromRadio(configRaw); err != nil {
		t.Fatalf("decode config frame: %v", err)
	}

	channelRaw, err := proto.Marshal(&generated.FromRadio{
		PayloadVariant: &generated.FromRadio_Channel{
			Channel: &generated.Channel{
				Index: 3,
				Role:  generated.Channel_PRIMARY,
				Settings: &generated.ChannelSettings{
					Name: "",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal channel frame: %v", err)
	}

	frame, err := codec.DecodeFromRadio(channelRaw)
	if err != nil {
		t.Fatalf("decode channel frame: %v", err)
	}
	if frame.Channels == nil || len(frame.Channels.Items) != 1 {
		t.Fatalf("expected one decoded channel")
	}
	if frame.Channels.Items[0].Title != "MediumFast" {
		t.Fatalf("expected MediumFast title, got %q", frame.Channels.Items[0].Title)
	}
}
