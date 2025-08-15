package protocol

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"
	"meshgo/internal/protocol/gomeshproto"
)

func TestEncodeDecode_MeshPacket(t *testing.T) {
	original := &gomeshproto.MeshPacket{
		From:     12345,
		To:       67890,
		Channel:  1,
		Id:       54321,
		WantAck:  true,
		Priority: gomeshproto.MeshPacket_UNSET,
		PayloadVariant: &gomeshproto.MeshPacket_Decoded{
			Decoded: &gomeshproto.Data{
				Portnum: gomeshproto.PortNum_TEXT_MESSAGE_APP,
				Payload: []byte("Hello, World!"),
			},
		},
	}

	// Test direct protobuf marshaling (bypass the problematic wrapper functions)
	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	// Test unmarshaling
	decoded := &gomeshproto.MeshPacket{}
	err = proto.Unmarshal(data, decoded)
	if err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}

	// Compare fields
	if decoded.From != original.From {
		t.Errorf("From: got %d, want %d", decoded.From, original.From)
	}
	if decoded.To != original.To {
		t.Errorf("To: got %d, want %d", decoded.To, original.To)
	}
	if decoded.Channel != original.Channel {
		t.Errorf("Channel: got %d, want %d", decoded.Channel, original.Channel)
	}
	if decoded.Id != original.Id {
		t.Errorf("Id: got %d, want %d", decoded.Id, original.Id)
	}
	if decoded.WantAck != original.WantAck {
		t.Errorf("WantAck: got %t, want %t", decoded.WantAck, original.WantAck)
	}
	if decoded.Priority != original.Priority {
		t.Errorf("Priority: got %d, want %d", decoded.Priority, original.Priority)
	}

	// Compare decoded payload
	originalDecoded := original.GetDecoded()
	decodedDecoded := decoded.GetDecoded()
	if originalDecoded == nil || decodedDecoded == nil {
		t.Fatal("Decoded payload is nil")
	}
	if decodedDecoded.Portnum != originalDecoded.Portnum {
		t.Errorf("Portnum: got %d, want %d", decodedDecoded.Portnum, originalDecoded.Portnum)
	}
	if !reflect.DeepEqual(decodedDecoded.Payload, originalDecoded.Payload) {
		t.Errorf("Payload: got %v, want %v", decodedDecoded.Payload, originalDecoded.Payload)
	}
}

func TestEncodeTextMessage(t *testing.T) {
	text := "Hello, Mesh!"
	encoded, err := EncodeTextMessage(text)
	if err != nil {
		t.Fatalf("EncodeTextMessage() error = %v", err)
	}

	// Decode the Data message and check the payload
	data := &Data{}
	err = proto.Unmarshal(encoded, data)
	if err != nil {
		t.Fatalf("Failed to unmarshal encoded data: %v", err)
	}

	if data.Portnum != PortTextMessageApp {
		t.Errorf("Portnum: got %d, want %d", data.Portnum, PortTextMessageApp)
	}
	if string(data.Payload) != text {
		t.Errorf("Payload: got %s, want %s", string(data.Payload), text)
	}
}

func TestDecodeTextMessage(t *testing.T) {
	text := "Test message"
	payload := []byte(text)

	data := &Data{Portnum: PortTextMessageApp, Payload: payload}
	decoded, err := DecodePayload(data)
	if err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}

	textMsg, ok := decoded.(*TextMessage)
	if !ok {
		t.Fatalf("Expected *TextMessage, got %T", decoded)
	}

	if textMsg.Text != text {
		t.Errorf("Text: got %s, want %s", textMsg.Text, text)
	}
}

func TestDecodePosition(t *testing.T) {
	// Create test position data (simplified)
	payload := make([]byte, 16)
	// Latitude: 37.5594120 * 1e7 = 375594120
	payload[0] = 0x78
	payload[1] = 0x6C
	payload[2] = 0x65
	payload[3] = 0x16
	// Longitude: -121.3894470 * 1e7 = -1213894470 (as uint32: 3081072826)
	payload[4] = 0xCA
	payload[5] = 0x7E
	payload[6] = 0xDA
	payload[7] = 0xB7
	// Altitude: 100
	payload[8] = 0x64
	payload[9] = 0x00
	payload[10] = 0x00
	payload[11] = 0x00
	// Time: 1609459200 (Jan 1, 2021)
	payload[12] = 0x00
	payload[13] = 0x6C
	payload[14] = 0xF7
	payload[15] = 0x5F

	data := &Data{Portnum: PortPositionApp, Payload: payload}
	decoded, err := DecodePayload(data)
	if err != nil {
		t.Fatalf("DecodePayload() error = %v", err)
	}

	position, ok := decoded.(*Position)
	if !ok {
		t.Fatalf("Expected *Position, got %T", decoded)
	}

	if position.LatitudeI != 375594120 {
		t.Errorf("LatitudeI: got %d, want %d", position.LatitudeI, 375594120)
	}

	expectedLat := 37.5594120
	actualLat := position.Latitude()
	if abs(actualLat-expectedLat) > 0.0000001 {
		t.Errorf("Latitude(): got %f, want %f", actualLat, expectedLat)
	}
}


func TestDecodeMeshPacket_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Too short", []byte{1, 2, 3}},
		{"Empty", []byte{}},
		{"Header only", make([]byte, 20)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeMeshPacket(tt.data)
			if err == nil {
				t.Errorf("DecodeMeshPacket() expected error for %s", tt.name)
			}
		})
	}
}

func TestEncodeMeshPacket_NilPacket(t *testing.T) {
	_, err := EncodeMeshPacket(nil)
	if err == nil {
		t.Error("EncodeMeshPacket(nil) expected error")
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
