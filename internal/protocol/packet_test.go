package protocol

import (
	"reflect"
	"testing"
)

func TestEncodeDecode_MeshPacket(t *testing.T) {
	original := &MeshPacket{
		From:     12345,
		To:       67890,
		Channel:  1,
		ID:       54321,
		PortNum:  PortTextMessageApp,
		Payload:  []byte("Hello, World!"),
		WantAck:  true,
		Priority: Default,
	}

	// Encode
	data, err := EncodeMeshPacket(original)
	if err != nil {
		t.Fatalf("EncodeMeshPacket() error = %v", err)
	}

	// Decode
	decoded, err := DecodeMeshPacket(data)
	if err != nil {
		t.Fatalf("DecodeMeshPacket() error = %v", err)
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
	if decoded.ID != original.ID {
		t.Errorf("ID: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.PortNum != original.PortNum {
		t.Errorf("PortNum: got %d, want %d", decoded.PortNum, original.PortNum)
	}
	if decoded.WantAck != original.WantAck {
		t.Errorf("WantAck: got %t, want %t", decoded.WantAck, original.WantAck)
	}
	if !reflect.DeepEqual(decoded.Payload, original.Payload) {
		t.Errorf("Payload: got %v, want %v", decoded.Payload, original.Payload)
	}
}

func TestEncodeTextMessage(t *testing.T) {
	text := "Hello, Mesh!"
	encoded := EncodeTextMessage(text)
	
	if string(encoded) != text {
		t.Errorf("EncodeTextMessage() = %s, want %s", string(encoded), text)
	}
}

func TestDecodeTextMessage(t *testing.T) {
	text := "Test message"
	payload := []byte(text)
	
	decoded, err := decodePayload(PortTextMessageApp, payload)
	if err != nil {
		t.Fatalf("decodePayload() error = %v", err)
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
	
	decoded, err := decodePayload(PortPositionApp, payload)
	if err != nil {
		t.Fatalf("decodePayload() error = %v", err)
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

func TestIsChannelMessage(t *testing.T) {
	tests := []struct {
		name     string
		packet   *MeshPacket
		expected bool
	}{
		{"Channel broadcast", &MeshPacket{To: 0xFFFFFFFF}, true},
		{"Direct message", &MeshPacket{To: 12345}, false},
		{"Zero destination", &MeshPacket{To: 0}, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsChannelMessage(tt.packet)
			if result != tt.expected {
				t.Errorf("IsChannelMessage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsDMMessage(t *testing.T) {
	tests := []struct {
		name     string
		packet   *MeshPacket
		expected bool
	}{
		{"Direct message", &MeshPacket{To: 12345}, true},
		{"Channel broadcast", &MeshPacket{To: 0xFFFFFFFF}, false},
		{"Zero destination", &MeshPacket{To: 0}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDMMessage(tt.packet)
			if result != tt.expected {
				t.Errorf("IsDMMessage() = %v, want %v", result, tt.expected)
			}
		})
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