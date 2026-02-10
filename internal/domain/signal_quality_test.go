package domain

import "testing"

func TestDetermineSignalQuality(t *testing.T) {
	tests := []struct {
		name string
		snr  float32
		rssi int
		want SignalQuality
	}{
		{name: "unknown when rssi missing", snr: -1, rssi: 0, want: SignalUnknown},
		{name: "good on exact boundary", snr: SNRGood, rssi: RSSIGood, want: SignalGood},
		{name: "fair on exact boundary", snr: SNRFair, rssi: RSSIFair, want: SignalFair},
		{name: "bad when below fair", snr: SNRFair - 0.1, rssi: RSSIFair - 1, want: SignalBad},
	}

	for _, tt := range tests {
		if got := DetermineSignalQuality(tt.snr, tt.rssi); got != tt.want {
			t.Fatalf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}
