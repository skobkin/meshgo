package domain

import "testing"

func TestComputeSignalQuality(t *testing.T) {
	cases := []struct {
		rssi int
		snr  float64
		want SignalQuality
	}{
		{-90, 9, SignalGood},
		{-100, 3, SignalFair},
		{-120, 1, SignalBad},
	}
	for _, c := range cases {
		if got := ComputeSignalQuality(c.rssi, c.snr); got != c.want {
			t.Errorf("ComputeSignalQuality(%d,%f)=%v want %v", c.rssi, c.snr, got, c.want)
		}
	}
}
