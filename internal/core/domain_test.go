package core

import (
	"testing"
)

func TestCalculateSignalQuality(t *testing.T) {
	tests := []struct {
		name     string
		rssi     int
		snr      float32
		expected SignalQuality
	}{
		{"Good signal", -80, 10.0, SignalGood},
		{"Good signal boundary", -95, 8.0, SignalGood},
		{"Fair signal", -100, 5.0, SignalFair},
		{"Fair signal boundary", -110, 2.0, SignalFair},
		{"Bad signal low RSSI", -120, 10.0, SignalBad},
		{"Bad signal low SNR", -80, 1.0, SignalBad},
		{"Bad signal both low", -130, -5.0, SignalBad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSignalQuality(tt.rssi, tt.snr)
			if result != tt.expected {
				t.Errorf("CalculateSignalQuality(%d, %f) = %v, want %v",
					tt.rssi, tt.snr, result, tt.expected)
			}
		})
	}
}

func TestSignalQuality_String(t *testing.T) {
	tests := []struct {
		quality  SignalQuality
		expected string
	}{
		{SignalGood, "Good"},
		{SignalFair, "Fair"},
		{SignalBad, "Bad"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.quality.String()
			if result != tt.expected {
				t.Errorf("%v.String() = %s, want %s", tt.quality, result, tt.expected)
			}
		})
	}
}

func TestDetermineEncryption(t *testing.T) {
	tests := []struct {
		name     string
		psk      []byte
		expected EncryptionState
	}{
		{"No encryption", []byte{}, EncryptionNone},
		{"Default key sentinel", []byte{1}, EncryptionDefault},
		{"Default key sentinel high", []byte{10}, EncryptionDefault},
		{"Invalid sentinel", []byte{11}, EncryptionNone},
		{"Custom 16-byte key", make([]byte, 16), EncryptionCustom},
		{"Custom 32-byte key", make([]byte, 32), EncryptionCustom},
		{"Invalid length", make([]byte, 24), EncryptionNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetermineEncryption(tt.psk)
			if result != tt.expected {
				t.Errorf("DetermineEncryption(%v) = %v, want %v",
					tt.psk, result, tt.expected)
			}
		})
	}
}

func TestEncryptionState_String(t *testing.T) {
	tests := []struct {
		state    EncryptionState
		expected string
	}{
		{EncryptionNone, "Not encrypted"},
		{EncryptionDefault, "Encrypted (default)"},
		{EncryptionCustom, "Encrypted (custom)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("%v.String() = %s, want %s", tt.state, result, tt.expected)
			}
		})
	}
}

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateDisconnected, "Disconnected"},
		{StateConnecting, "Connecting"},
		{StateConnected, "Connected"},
		{StateRetrying, "Retrying"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("%v.String() = %s, want %s", tt.state, result, tt.expected)
			}
		})
	}
}

func TestPosition_LatitudeLongitude(t *testing.T) {
	pos := &Position{
		LatitudeI:  375594120,  // 37.5594120 degrees
		LongitudeI: -1213894470, // -121.3894470 degrees
	}

	expectedLat := 37.5594120
	expectedLon := -121.3894470

	lat := pos.Latitude()
	lon := pos.Longitude()

	if abs(lat-expectedLat) > 0.0000001 {
		t.Errorf("Latitude() = %f, want %f", lat, expectedLat)
	}

	if abs(lon-expectedLon) > 0.0000001 {
		t.Errorf("Longitude() = %f, want %f", lon, expectedLon)
	}
}

func TestDeviceMetrics_IsCharging(t *testing.T) {
	tests := []struct {
		name         string
		batteryLevel uint32
		expected     bool
	}{
		{"Normal battery", 75, false},
		{"Full battery", 100, false},
		{"External power", 101, true},
		{"High external power", 255, true},
		{"Zero battery", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &DeviceMetrics{BatteryLevel: tt.batteryLevel}
			result := dm.IsCharging()
			if result != tt.expected {
				t.Errorf("IsCharging() with battery %d = %v, want %v",
					tt.batteryLevel, result, tt.expected)
			}
		})
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}