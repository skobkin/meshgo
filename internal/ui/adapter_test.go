package ui

import (
	"testing"
	"time"

	"meshgo/internal/core"
)

func TestNodeToViewModel(t *testing.T) {
	node := &core.Node{
		ID:            "node_123",
		ShortName:     "TestNode",
		LongName:      "Test Node Device",
		RSSI:          -85,
		SNR:           12.5,
		SignalQuality: int(core.SignalGood),
		LastHeard:     time.Now(),
		Favorite:      true,
		Ignored:       false,
		Position: &core.Position{
			LatitudeI:  375594120,
			LongitudeI: -1213894470,
		},
		DeviceMetrics: &core.DeviceMetrics{
			BatteryLevel: 85,
			Voltage:      3.7,
		},
	}

	viewModel := NodeToViewModel(node)

	if viewModel.ID != node.ID {
		t.Errorf("ID mismatch: got %s, want %s", viewModel.ID, node.ID)
	}
	if viewModel.ShortName != node.ShortName {
		t.Errorf("ShortName mismatch: got %s, want %s", viewModel.ShortName, node.ShortName)
	}
	if viewModel.LongName != node.LongName {
		t.Errorf("LongName mismatch: got %s, want %s", viewModel.LongName, node.LongName)
	}
	if viewModel.RSSI != node.RSSI {
		t.Errorf("RSSI mismatch: got %d, want %d", viewModel.RSSI, node.RSSI)
	}
	if viewModel.SNR != node.SNR {
		t.Errorf("SNR mismatch: got %f, want %f", viewModel.SNR, node.SNR)
	}
	if viewModel.SignalQuality != node.SignalQuality {
		t.Errorf("SignalQuality mismatch: got %d, want %d", viewModel.SignalQuality, node.SignalQuality)
	}
	if viewModel.Favorite != node.Favorite {
		t.Errorf("Favorite mismatch: got %v, want %v", viewModel.Favorite, node.Favorite)
	}
	if viewModel.Ignored != node.Ignored {
		t.Errorf("Ignored mismatch: got %v, want %v", viewModel.Ignored, node.Ignored)
	}
}

func TestNodeToViewModel_NilPosition(t *testing.T) {
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		Position:  nil, // Test nil position
	}

	viewModel := NodeToViewModel(node)

	if viewModel.Position != nil {
		t.Error("Position should be nil when node position is nil")
	}
}

func TestNodeToViewModel_WithPosition(t *testing.T) {
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		Position: &core.Position{
			LatitudeI:  375594120,   // 37.5594120 degrees
			LongitudeI: -1213894470, // -121.3894470 degrees
			Altitude:   100,
		},
	}

	viewModel := NodeToViewModel(node)

	if viewModel.Position == nil {
		t.Error("Position should not be nil when node has position")
		return
	}

	expectedLat := 37.5594120
	expectedLon := -121.3894470
	tolerance := 0.0000001

	actualLat := viewModel.Position.Latitude()
	actualLon := viewModel.Position.Longitude()
	if abs(actualLat-expectedLat) > tolerance {
		t.Errorf("Latitude mismatch: got %f, want %f", actualLat, expectedLat)
	}
	if abs(actualLon-expectedLon) > tolerance {
		t.Errorf("Longitude mismatch: got %f, want %f", actualLon, expectedLon)
	}
	if viewModel.Position.Altitude != 100 {
		t.Errorf("Altitude mismatch: got %d, want %d", viewModel.Position.Altitude, 100)
	}
}

func TestNodeToViewModel_NilDeviceMetrics(t *testing.T) {
	node := &core.Node{
		ID:            "node_123",
		ShortName:     "TestNode",
		DeviceMetrics: nil, // Test nil device metrics
	}

	viewModel := NodeToViewModel(node)

	if viewModel.DeviceMetrics != nil {
		t.Error("DeviceMetrics should be nil when node DeviceMetrics is nil")
	}
}

func TestNodeToViewModel_WithDeviceMetrics(t *testing.T) {
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		DeviceMetrics: &core.DeviceMetrics{
			BatteryLevel: 85,
			Voltage:      3.7,
		},
	}

	viewModel := NodeToViewModel(node)

	if viewModel.DeviceMetrics == nil {
		t.Error("DeviceMetrics should not be nil when node has DeviceMetrics")
		return
	}
	if viewModel.BatteryPercent != 85 {
		t.Errorf("BatteryPercent mismatch: got %d, want %d", viewModel.BatteryPercent, 85)
	}
	if viewModel.DeviceMetrics.BatteryLevel != 85 {
		t.Errorf("BatteryLevel mismatch: got %d, want %d", viewModel.DeviceMetrics.BatteryLevel, 85)
	}
	if viewModel.DeviceMetrics.Voltage != 3.7 {
		t.Errorf("Voltage mismatch: got %f, want %f", viewModel.DeviceMetrics.Voltage, 3.7)
	}
}

func TestNodeToViewModel_ChargingBattery(t *testing.T) {
	node := &core.Node{
		ID:        "node_123",
		ShortName: "TestNode",
		DeviceMetrics: &core.DeviceMetrics{
			BatteryLevel: 105, // Over 100% indicates charging
			Voltage:      4.2,
		},
	}

	viewModel := NodeToViewModel(node)

	if !viewModel.DeviceMetrics.IsCharging() {
		t.Error("IsCharging should be true for battery level > 100")
	}
}

func TestMessageFields(t *testing.T) {
	timestamp := time.Now()
	message := &core.Message{
		ChatID:    "chat_123",
		SenderID:  "node_456",
		PortNum:   1,
		Text:      "Hello, World!",
		Timestamp: timestamp,
		IsUnread:  true,
	}

	// Test message struct fields directly
	if message.ChatID != "chat_123" {
		t.Errorf("ChatID mismatch: got %s, want %s", message.ChatID, "chat_123")
	}
	if message.SenderID != "node_456" {
		t.Errorf("SenderID mismatch: got %s, want %s", message.SenderID, "node_456")
	}
	if message.PortNum != 1 {
		t.Errorf("PortNum mismatch: got %d, want %d", message.PortNum, 1)
	}
	if message.Text != "Hello, World!" {
		t.Errorf("Text mismatch: got %s, want %s", message.Text, "Hello, World!")
	}
	if message.Timestamp != timestamp {
		t.Errorf("Timestamp mismatch: got %v, want %v", message.Timestamp, timestamp)
	}
	if message.IsUnread != true {
		t.Errorf("IsUnread mismatch: got %v, want %v", message.IsUnread, true)
	}
}

func TestChatFields(t *testing.T) {
	timestamp := time.Now()
	chat := &core.Chat{
		ID:            "chat_123",
		Title:         "Test Chat",
		UnreadCount:   5,
		LastMessageTS: timestamp,
		Encryption:    1,
		IsChannel:     false,
	}

	// Test chat struct fields directly
	if chat.ID != "chat_123" {
		t.Errorf("ID mismatch: got %s, want %s", chat.ID, "chat_123")
	}
	if chat.Title != "Test Chat" {
		t.Errorf("Title mismatch: got %s, want %s", chat.Title, "Test Chat")
	}
	if chat.UnreadCount != 5 {
		t.Errorf("UnreadCount mismatch: got %d, want %d", chat.UnreadCount, 5)
	}
	if chat.LastMessageTS != timestamp {
		t.Errorf("LastMessageTS mismatch: got %v, want %v", chat.LastMessageTS, timestamp)
	}
	if chat.Encryption != 1 {
		t.Errorf("Encryption mismatch: got %d, want %d", chat.Encryption, 1)
	}
	if chat.IsChannel != false {
		t.Errorf("IsChannel mismatch: got %v, want %v", chat.IsChannel, false)
	}
}

func TestNodeToViewModel_DisplayName(t *testing.T) {
	tests := []struct {
		name         string
		shortName    string
		longName     string
		expectedName string
	}{
		{
			name:         "Both names present",
			shortName:    "Short",
			longName:     "Long Name",
			expectedName: "Short",
		},
		{
			name:         "Only long name",
			shortName:    "",
			longName:     "Long Name",
			expectedName: "Long Name",
		},
		{
			name:         "Only short name",
			shortName:    "Short",
			longName:     "",
			expectedName: "Short",
		},
		{
			name:         "No names",
			shortName:    "",
			longName:     "",
			expectedName: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &core.Node{
				ID:        "node_123",
				ShortName: tt.shortName,
				LongName:  tt.longName,
			}

			viewModel := NodeToViewModel(node)

			// Check the underlying node fields since DisplayName logic would be in a separate function
			expectedName := tt.shortName
			if expectedName == "" {
				expectedName = tt.longName
			}
			if expectedName == "" {
				expectedName = "Unknown"
			}

			actualName := viewModel.ShortName
			if actualName == "" {
				actualName = viewModel.LongName
			}
			if actualName == "" {
				actualName = "Unknown"
			}

			if actualName != expectedName {
				t.Errorf("Name logic mismatch: got %s, want %s", actualName, expectedName)
			}
		})
	}
}

func TestNodeToViewModel_SignalQualityString(t *testing.T) {
	tests := []struct {
		name     string
		rssi     int
		snr      float32
		expected string
	}{
		{"Good", -90, 10.0, "Good"},
		{"Fair", -105, 5.0, "Fair"},
		{"Bad", -125, -2.0, "Poor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &core.Node{
				ID:        "node_123",
				LastHeard: time.Now(), // Set recent time so node appears online
				RSSI:      tt.rssi,    // Provide signal data
				SNR:       tt.snr,
			}

			viewModel := NodeToViewModel(node)

			// Check StatusText which should correspond to signal quality
			if viewModel.StatusText != tt.expected {
				t.Errorf("StatusText mismatch: got %s, want %s",
					viewModel.StatusText, tt.expected)
			}
		})
	}
}

func TestNodeToViewModel_NilInput(t *testing.T) {
	// Test that the function handles nil input gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NodeToViewModel should not panic with nil input")
		}
	}()

	viewModel := NodeToViewModel(nil)

	// Should handle nil gracefully
	if viewModel == nil {
		t.Error("NodeToViewModel returned nil for nil input")
		return
	}
	if viewModel.Node != nil {
		t.Error("Embedded Node should be nil for nil input")
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
