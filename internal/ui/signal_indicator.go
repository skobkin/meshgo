package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/skobkin/meshgo/internal/domain"
)

var (
	signalColorGood = color.NRGBA{R: 0x2e, G: 0xcc, B: 0x71, A: 0xff}
	signalColorFair = color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff}
	signalColorBad  = color.NRGBA{R: 0xe5, G: 0x39, B: 0x35, A: 0xff}
)

func signalQualityFromMetrics(rssi *int, snr *float64) (domain.SignalQuality, bool) {
	if rssi == nil || snr == nil {
		return domain.SignalUnknown, false
	}
	quality := domain.DetermineSignalQuality(float32(*snr), *rssi)
	if quality == domain.SignalUnknown {
		return quality, false
	}

	return quality, true
}

func signalBarsForQuality(quality domain.SignalQuality) string {
	switch quality {
	case domain.SignalGood:
		return "▂▄▆█"
	case domain.SignalFair:
		return "▂▄▆ "
	case domain.SignalBad:
		return "▂▄  "
	default:
		return ""
	}
}

func signalColorForQuality(quality domain.SignalQuality) color.Color {
	switch quality {
	case domain.SignalGood:
		return signalColorGood
	case domain.SignalFair:
		return signalColorFair
	case domain.SignalBad:
		return signalColorBad
	default:
		return signalColorBad
	}
}

func signalThemeColorForQuality(quality domain.SignalQuality) fyne.ThemeColorName {
	switch quality {
	case domain.SignalGood:
		return theme.ColorNameSuccess
	case domain.SignalFair:
		return theme.ColorNameWarning
	case domain.SignalBad:
		return theme.ColorNameError
	default:
		return theme.ColorNameForeground
	}
}

func signalThemeColorForRSSI(rssi int) fyne.ThemeColorName {
	switch {
	case rssi >= domain.RSSIGood:
		return theme.ColorNameSuccess
	case rssi >= domain.RSSIFair:
		return theme.ColorNameWarning
	default:
		return theme.ColorNameError
	}
}

func signalThemeColorForSNR(snr float64) fyne.ThemeColorName {
	switch {
	case snr >= float64(domain.SNRGood):
		return theme.ColorNameSuccess
	case snr >= float64(domain.SNRFair):
		return theme.ColorNameWarning
	default:
		return theme.ColorNameError
	}
}
