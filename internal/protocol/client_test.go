package protocol

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

// Mock transport for testing
type mockTransport struct {
	connected   bool
	readData    []byte
	readError   error
	writeError  error
	writtenData []byte
	endpoint    string
}

func newMockTransport(endpoint string) *mockTransport {
	return &mockTransport{
		endpoint: endpoint,
	}
}

func (m *mockTransport) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockTransport) Close() error {
	m.connected = false
	return nil
}

func (m *mockTransport) ReadPacket(ctx context.Context) ([]byte, error) {
	if !m.connected {
		return nil, errors.New("not connected")
	}
	if m.readError != nil {
		return nil, m.readError
	}
	return m.readData, nil
}

func (m *mockTransport) WritePacket(ctx context.Context, data []byte) error {
	if !m.connected {
		return errors.New("not connected")
	}
	if m.writeError != nil {
		return m.writeError
	}
	m.writtenData = data
	return nil
}

func (m *mockTransport) IsConnected() bool {
	return m.connected
}

func (m *mockTransport) Endpoint() string {
	return m.endpoint
}

func TestNewRadioClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	if client == nil {
		t.Fatal("NewRadioClient returned nil")
	}

	if client.logger != logger {
		t.Error("Logger not set correctly")
	}

	if client.events == nil {
		t.Error("Events channel not initialized")
	}

	if client.nodeDB == nil {
		t.Error("Node database not initialized")
	}

	if client.channelDB == nil {
		t.Error("Channel database not initialized")
	}

	if client.running {
		t.Error("Client should not be running initially")
	}
}

func TestRadioClient_Start(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)
	transport := newMockTransport("mock://test")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test successful start
	err := client.Start(ctx, transport)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !client.isRunning() {
		t.Error("Client should be running after Start")
	}

	// Test double start (should fail)
	err = client.Start(ctx, transport)
	if err == nil {
		t.Error("Start should fail when already running")
	}

	// Cleanup
	_ = client.Stop()
}

func TestRadioClient_Stop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)
	transport := newMockTransport("mock://test")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start first
	err := client.Start(ctx, transport)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test stop
	err = client.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if client.isRunning() {
		t.Error("Client should not be running after Stop")
	}

	// Test double stop (should not fail)
	err = client.Stop()
	if err != nil {
		t.Errorf("Stop should not fail when already stopped: %v", err)
	}
}

func TestRadioClient_Events(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	eventsChan := client.Events()
	if eventsChan == nil {
		t.Error("Events channel should not be nil")
	}

	// Test that we can read from the channel (should be empty initially)
	select {
	case <-eventsChan:
		t.Error("Events channel should be empty initially")
	default:
		// Expected
	}
}

func TestRadioClient_SendText_NotRunning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	ctx := context.Background()
	err := client.SendText(ctx, "test_chat", 12345, "Hello")

	if err == nil {
		t.Error("SendText should fail when client is not running")
	}
}

func TestRadioClient_generatePacketID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	// Generate multiple IDs
	id1 := client.generatePacketID()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamp
	id2 := client.generatePacketID()

	if id1 == id2 {
		t.Error("Generated packet IDs should be different")
	}
}

func TestRadioClient_getOrCreateNode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	nodeID := "test_node_123"

	// Get non-existent node (should create)
	node1 := client.getOrCreateNode(nodeID)
	if node1 == nil {
		t.Fatal("getOrCreateNode returned nil")
	}

	if node1.ID != nodeID {
		t.Errorf("Node ID mismatch: got %s, want %s", node1.ID, nodeID)
	}

	// Get existing node (should return same instance)
	node2 := client.getOrCreateNode(nodeID)
	if node1 != node2 {
		t.Error("getOrCreateNode should return same instance for existing node")
	}
}

func TestRadioClient_GetNodes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	// Initially should be empty
	nodes := client.GetNodes()
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes initially, got %d", len(nodes))
	}

	// Create some nodes
	node1 := client.getOrCreateNode("node1")
	node1.LastHeard = time.Now()

	node2 := client.getOrCreateNode("node2")
	node2.LastHeard = time.Now()

	// Should return both nodes
	nodes = client.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}

	// Test stale node cleanup - create old node
	staleNode := client.getOrCreateNode("stale_node")
	staleNode.LastHeard = time.Now().Add(-15 * time.Minute) // Older than 10 minutes

	// Should still return 2 nodes (stale one should be removed)
	nodes = client.GetNodes()
	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes after cleanup, got %d", len(nodes))
	}

	// Verify stale node was removed
	for _, node := range nodes {
		if node.ID == "stale_node" {
			t.Error("Stale node should have been removed")
		}
	}
}

func TestRadioClient_cleanNodeName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	tests := []struct {
		input    string
		expected string
	}{
		{"ValidName", "ValidName"},
		{"Name with spaces", "Name with spaces"},
		{"\"QuotedName\"", "QuotedName"},
		{"'SingleQuoted'", "SingleQuoted"},
		{"Name\x00WithNull", "NameWithNull"},
		{"\x01\x02InvalidStart", "InvalidStart"},
		{"ValidEnd\x01\x02", "ValidEnd"},
		{"", ""},
		{"\x00\x01\x02", ""},
		{"Name\x7f\x80", "Name"}, // Remove high ASCII
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := client.cleanNodeName(tt.input)
			if result != tt.expected {
				t.Errorf("cleanNodeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRadioClient_isValidNodeName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid name", "MyDevice", true},
		{"Valid with numbers", "Device123", true},
		{"Valid with dash", "My-Device", true},
		{"Valid with underscore", "My_Device", true},
		{"Too short", "abc", false},
		{"Too long", "ThisNameIsTooLongForAValidNodeName", false},
		{"Too many special chars", "!@#$%^&*()", false},
		{"Mixed valid", "Node-1_Test", true},
		{"Empty", "", false},
		{"Only special chars", "!@#$", false},
		{"Mostly garbage", "abc!@#$%^&*()def", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isValidNodeName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidNodeName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRadioClient_requestAllChannels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)
	transport := newMockTransport("mock://test")

	ctx := context.Background()

	// Set up the client with transport
	client.transport = transport
	_ = transport.Connect(ctx)

	// This should not panic and should send requests
	client.requestAllChannels(ctx)

	// Verify that something was written to transport
	if len(transport.writtenData) == 0 {
		t.Error("requestAllChannels should have written data to transport")
	}
}

func TestRadioClient_isRunning(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	// Initially not running
	if client.isRunning() {
		t.Error("Client should not be running initially")
	}

	// Set to running
	client.runningMu.Lock()
	client.running = true
	client.runningMu.Unlock()

	if !client.isRunning() {
		t.Error("Client should be running after setting flag")
	}

	// Set back to not running
	client.runningMu.Lock()
	client.running = false
	client.runningMu.Unlock()

	if client.isRunning() {
		t.Error("Client should not be running after unsetting flag")
	}
}

func TestRadioClient_ConcurrentAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	// Test concurrent access to node database
	done := make(chan bool, 3)

	// Goroutine 1: Add nodes
	go func() {
		for i := 0; i < 50; i++ {
			nodeID := fmt.Sprintf("node_%d", i)
			node := client.getOrCreateNode(nodeID)
			node.LastHeard = time.Now()
		}
		done <- true
	}()

	// Goroutine 2: Get all nodes
	go func() {
		for i := 0; i < 50; i++ {
			_ = client.GetNodes()
		}
		done <- true
	}()

	// Goroutine 3: Access running state
	go func() {
		for i := 0; i < 50; i++ {
			_ = client.isRunning()
		}
		done <- true
	}()

	// Wait for all operations to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Good
		case <-timeout:
			t.Fatal("Concurrent access test timed out")
		}
	}
}

func TestIsChannelMessage(t *testing.T) {
	tests := []struct {
		name     string
		packet   *MeshPacket
		expected bool
	}{
		{
			name:     "Broadcast message",
			packet:   &MeshPacket{To: 0xFFFFFFFF},
			expected: true,
		},
		{
			name:     "Direct message",
			packet:   &MeshPacket{To: 12345},
			expected: false,
		},
		{
			name:     "Zero destination",
			packet:   &MeshPacket{To: 0},
			expected: false,
		},
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
		{
			name:     "Direct message",
			packet:   &MeshPacket{To: 12345},
			expected: true,
		},
		{
			name:     "Broadcast message",
			packet:   &MeshPacket{To: 0xFFFFFFFF},
			expected: false,
		},
		{
			name:     "Zero destination",
			packet:   &MeshPacket{To: 0},
			expected: true,
		},
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

func TestCreateStartConfigRequest(t *testing.T) {
	request := CreateStartConfigRequest()

	if request == nil {
		t.Fatal("CreateStartConfigRequest returned nil")
	}

	// Check that it has the correct payload variant
	if request.GetWantConfigId() != 42 {
		t.Errorf("Expected WantConfigId = 42, got %d", request.GetWantConfigId())
	}
}

// Benchmark tests
func BenchmarkRadioClient_generatePacketID(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.generatePacketID()
	}
}

func BenchmarkRadioClient_getOrCreateNode(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewRadioClient(logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nodeID := fmt.Sprintf("node_%d", i%100) // Reuse some nodes
		client.getOrCreateNode(nodeID)
	}
}
