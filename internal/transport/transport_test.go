package transport

import (
	"context"
	"testing"
	"time"
)

func TestSerialTransport_Endpoint(t *testing.T) {
	transport := NewSerialTransport("/dev/ttyUSB0", 115200)
	
	expected := "serial:///dev/ttyUSB0@115200"
	actual := transport.Endpoint()
	
	if actual != expected {
		t.Errorf("Endpoint() = %s, want %s", actual, expected)
	}
}

func TestSerialTransport_DefaultBaudRate(t *testing.T) {
	transport := NewSerialTransport("/dev/ttyUSB0", 0)
	
	expected := "serial:///dev/ttyUSB0@115200"
	actual := transport.Endpoint()
	
	if actual != expected {
		t.Errorf("Default baud rate not applied. Endpoint() = %s, want %s", actual, expected)
	}
}

func TestTCPTransport_Endpoint(t *testing.T) {
	transport := NewTCPTransport("192.168.1.100", 4403)
	
	expected := "tcp://192.168.1.100:4403"
	actual := transport.Endpoint()
	
	if actual != expected {
		t.Errorf("Endpoint() = %s, want %s", actual, expected)
	}
}

func TestTCPTransport_DefaultPort(t *testing.T) {
	transport := NewTCPTransport("192.168.1.100", 0)
	
	expected := "tcp://192.168.1.100:4403"
	actual := transport.Endpoint()
	
	if actual != expected {
		t.Errorf("Default port not applied. Endpoint() = %s, want %s", actual, expected)
	}
}

func TestTransport_InitialState(t *testing.T) {
	serialTransport := NewSerialTransport("/dev/ttyUSB0", 115200)
	tcpTransport := NewTCPTransport("localhost", 4403)
	
	if serialTransport.IsConnected() {
		t.Error("Serial transport should not be connected initially")
	}
	
	if tcpTransport.IsConnected() {
		t.Error("TCP transport should not be connected initially")
	}
}

func TestTransport_WriteWithoutConnection(t *testing.T) {
	ctx := context.Background()
	serialTransport := NewSerialTransport("/dev/ttyUSB0", 115200)
	tcpTransport := NewTCPTransport("localhost", 4403)
	
	testData := []byte("test data")
	
	err := serialTransport.WritePacket(ctx, testData)
	if err == nil {
		t.Error("WritePacket should fail when not connected (serial)")
	}
	
	err = tcpTransport.WritePacket(ctx, testData)
	if err == nil {
		t.Error("WritePacket should fail when not connected (TCP)")
	}
}

func TestTransport_ReadWithoutConnection(t *testing.T) {
	ctx := context.Background()
	serialTransport := NewSerialTransport("/dev/ttyUSB0", 115200)
	tcpTransport := NewTCPTransport("localhost", 4403)
	
	_, err := serialTransport.ReadPacket(ctx)
	if err == nil {
		t.Error("ReadPacket should fail when not connected (serial)")
	}
	
	_, err = tcpTransport.ReadPacket(ctx)
	if err == nil {
		t.Error("ReadPacket should fail when not connected (TCP)")
	}
}

func TestTransport_ContextCancellation(t *testing.T) {
	serialTransport := NewSerialTransport("/dev/ttyUSB0", 115200)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	_, err := serialTransport.ReadPacket(ctx)
	if err == nil {
		t.Error("ReadPacket should respect context cancellation")
	}
	
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestTransport_ContextTimeout(t *testing.T) {
	serialTransport := NewSerialTransport("/dev/ttyUSB0", 115200)
	
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	time.Sleep(2 * time.Millisecond) // Ensure timeout
	
	_, err := serialTransport.ReadPacket(ctx)
	if err == nil {
		t.Error("ReadPacket should respect context timeout")
	}
	
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestSerialTransport_FramePacket(t *testing.T) {
	transport := NewSerialTransport("/dev/ttyUSB0", 115200)
	testData := []byte("Hello")
	
	framed := transport.framePacket(testData)
	
	// Check frame structure: magic bytes + size + data
	if len(framed) != len(testData)+4 {
		t.Errorf("Framed packet length = %d, want %d", len(framed), len(testData)+4)
	}
	
	// Check magic bytes
	if framed[0] != 0x94 || framed[1] != 0xC3 {
		t.Errorf("Invalid magic bytes: %02x %02x", framed[0], framed[1])
	}
	
	// Check size
	expectedSize := len(testData)
	actualSize := int(framed[2])<<8 | int(framed[3])
	if actualSize != expectedSize {
		t.Errorf("Frame size = %d, want %d", actualSize, expectedSize)
	}
	
	// Check data
	for i, b := range testData {
		if framed[4+i] != b {
			t.Errorf("Data byte %d = %02x, want %02x", i, framed[4+i], b)
		}
	}
}

func TestTCPTransport_FramePacket(t *testing.T) {
	transport := NewTCPTransport("localhost", 4403)
	testData := []byte("Test message")
	
	framed := transport.framePacket(testData)
	
	// Check frame structure: magic bytes + size + data
	if len(framed) != len(testData)+4 {
		t.Errorf("Framed packet length = %d, want %d", len(framed), len(testData)+4)
	}
	
	// Check magic bytes
	if framed[0] != 0x94 || framed[1] != 0xC3 {
		t.Errorf("Invalid magic bytes: %02x %02x", framed[0], framed[1])
	}
	
	// Check size
	expectedSize := len(testData)
	actualSize := int(framed[2])<<8 | int(framed[3])
	if actualSize != expectedSize {
		t.Errorf("Frame size = %d, want %d", actualSize, expectedSize)
	}
}

func TestDetectSerialPorts(t *testing.T) {
	// This test might fail on systems without serial ports
	// or without proper permissions, so we just check it doesn't crash
	ports, err := DetectSerialPorts()
	
	if err != nil {
		t.Logf("Serial port detection failed (this may be expected): %v", err)
		return
	}
	
	t.Logf("Detected %d serial ports: %v", len(ports), ports)
	
	// Test that we get a slice (even if empty)
	if ports == nil {
		t.Error("DetectSerialPorts should return non-nil slice")
	}
}

func TestIsLikelyMeshtasticPort(t *testing.T) {
	tests := []struct {
		port     string
		expected bool
	}{
		{"/dev/ttyUSB0", true},
		{"/dev/ttyACM0", true},
		{"COM3", true},
		{"/dev/cu.usbserial-1234", true},
		{"/dev/cu.wchusbserial-5678", true},
		{"/dev/ttyS0", false},
		{"/dev/null", false},
		{"random", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.port, func(t *testing.T) {
			result := isLikelyMeshtasticPort(tt.port)
			if result != tt.expected {
				t.Errorf("isLikelyMeshtasticPort(%s) = %v, want %v", 
					tt.port, result, tt.expected)
			}
		})
	}
}