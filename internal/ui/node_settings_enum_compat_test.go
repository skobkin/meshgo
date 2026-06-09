package ui

import (
	"testing"

	"fyne.io/fyne/v2/widget"

	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeLoRaUnsupportedSchemaEnumsRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   int32
		options []nodeLoRaEnumOption
	}{
		{
			name:    "region",
			value:   int32(generated.Config_LoRaConfig_ITU1_2M),
			options: nodeLoRaRegionOptions,
		},
		{
			name:    "modem preset",
			value:   int32(generated.Config_LoRaConfig_LITE_FAST),
			options: nodeLoRaModemPresetOptions,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			selectWidget := widget.NewSelect(nil, nil)
			nodeLoRaSetEnumSelect(selectWidget, tc.options, tc.value)

			got, err := nodeLoRaParseEnumLabel(tc.name, selectWidget.Selected, tc.options)
			if err != nil {
				t.Fatalf("parse unsupported value: %v", err)
			}
			if got != tc.value {
				t.Fatalf("round-trip value = %d, want %d", got, tc.value)
			}
		})
	}
}

func TestNodeDisplayUnsupportedRotatedOLEDTypeRoundTrips(t *testing.T) {
	t.Parallel()

	value := int32(generated.Config_DisplayConfig_OLED_SH1107_ROTATED)
	selectWidget := widget.NewSelect(nil, nil)
	nodeDisplaySetEnumSelect(selectWidget, nodeDisplayOledTypeOptions, value)

	got, err := nodeDisplayParseEnumLabel("OLED type", selectWidget.Selected, nodeDisplayOledTypeOptions)
	if err != nil {
		t.Fatalf("parse unsupported value: %v", err)
	}
	if got != value {
		t.Fatalf("round-trip value = %d, want %d", got, value)
	}
}
