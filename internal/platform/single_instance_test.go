package platform

import "testing"

func TestNormalizeLockComponent(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		fallback string
		want     string
	}{
		{name: "preserves alnum and separators", raw: "meshgo-v1.2_3", fallback: "app", want: "meshgo-v1.2_3"},
		{name: "replaces unsupported runes", raw: "meshgo:/v1", fallback: "app", want: "meshgo__v1"},
		{name: "trims separator edges", raw: ".._meshgo-._", fallback: "app", want: "meshgo"},
		{name: "empty uses fallback", raw: "   ", fallback: "fallback", want: "fallback"},
		{name: "all unsupported uses fallback", raw: "[]{}", fallback: "fallback", want: "fallback"},
	}

	for _, tc := range tests {
		got := normalizeInstanceLockComponent(tc.raw, tc.fallback)
		if got != tc.want {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.want, got)
		}
	}
}
