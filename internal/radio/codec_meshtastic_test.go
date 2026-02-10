package radio

import (
	"testing"

	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
	"google.golang.org/protobuf/proto"
)

func TestMeshtasticCodec_EncodeTextIncludesDeviceMessageID(t *testing.T) {
	codec := NewMeshtasticCodec()
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

func TestMeshtasticCodec_DecodeFromRadioQueueStatusSuccess(t *testing.T) {
	codec := NewMeshtasticCodec()

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
	codec := NewMeshtasticCodec()

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
	codec := NewMeshtasticCodec()

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
	codec := NewMeshtasticCodec()

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
	codec := NewMeshtasticCodec()

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
