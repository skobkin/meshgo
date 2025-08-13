package domain

// SignalQuality represents a qualitative signal strength.
type SignalQuality int

const (
	SignalBad SignalQuality = iota
	SignalFair
	SignalGood
)

// Node captures information about a Meshtastic node.
type Node struct {
	ID            string
	ShortName     string
	LongName      string
	Favorite      bool
	Ignored       bool
	Unencrypted   bool
	EncDefaultKey bool
	EncCustomKey  bool
	RSSI          int
	SNR           float64
	Signal        SignalQuality
	BatteryLevel  *int
	IsCharging    *bool
	LastHeard     int64
}

// ComputeSignalQuality returns an approximate quality bucket based on
// RSSI and SNR thresholds similar to the Android client.
func ComputeSignalQuality(rssi int, snr float64) SignalQuality {
	if snr >= 8 && rssi >= -95 {
		return SignalGood
	}
	if snr >= 2 && rssi >= -110 {
		return SignalFair
	}
	return SignalBad
}
