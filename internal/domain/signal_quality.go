package domain

const (
	SNRGood  = float32(-7)
	SNRFair  = float32(-15)
	RSSIGood = -115
	RSSIFair = -126
)

type SignalQuality int

const (
	SignalUnknown SignalQuality = iota
	SignalBad
	SignalFair
	SignalGood
)

// DetermineSignalQuality Thresholds match Meshtastic Android signal indicator:
// https://github.com/meshtastic/Meshtastic-Android/blob/fe5d7d6b92ae185fad5b4df9587a18c756512684/core/ui/src/main/kotlin/org/meshtastic/core/ui/component/LoraSignalIndicator.kt#L62-L66
func DetermineSignalQuality(snr float32, rssi int) SignalQuality {
	if rssi == 0 {
		return SignalUnknown
	}
	if snr >= SNRGood && rssi >= RSSIGood {
		return SignalGood
	}
	if snr >= SNRFair && rssi >= RSSIFair {
		return SignalFair
	}
	return SignalBad
}
