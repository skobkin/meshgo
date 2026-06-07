package ui

import (
	"math"
	"testing"
)

func TestNodeDisplayScreenOnMappingUsesAndroidAlwaysOnValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		label string
		value uint32
	}{
		{name: "normal duration", label: "15 seconds", value: 15},
		{name: "always on", label: "Always on", value: math.MaxInt32},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value, err := nodeDisplayParseScreenOnSelectLabel(tc.label)
			if err != nil {
				t.Fatalf("parse screen-on label: %v", err)
			}
			if value != tc.value {
				t.Fatalf("parsed value = %d, want %d", value, tc.value)
			}
			if label := nodeDisplayScreenOnLabel(tc.value); label != tc.label {
				t.Fatalf("formatted label = %q, want %q", label, tc.label)
			}
		})
	}
}
