package platform

import "testing"

func TestNormalizeAutostartMode(t *testing.T) {
	if got := normalizeAutostartMode(AutostartModeBackground); got != AutostartModeBackground {
		t.Fatalf("expected %q, got %q", AutostartModeBackground, got)
	}
	if got := normalizeAutostartMode(AutostartMode("invalid")); got != AutostartModeNormal {
		t.Fatalf("expected invalid mode to normalize to %q, got %q", AutostartModeNormal, got)
	}
}

func TestLaunchArgsForMode(t *testing.T) {
	if got := launchArgsForMode(AutostartModeNormal); len(got) != 0 {
		t.Fatalf("expected no args for normal mode, got %#v", got)
	}

	got := launchArgsForMode(AutostartModeBackground)
	if len(got) != 1 || got[0] != startHiddenArg {
		t.Fatalf("unexpected args for background mode: %#v", got)
	}
}
