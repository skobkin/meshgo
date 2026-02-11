package platform

import "testing"

func TestNormalizeLaunchMode(t *testing.T) {
	if got := normalizeLaunchMode(LaunchModeBackground); got != LaunchModeBackground {
		t.Fatalf("expected %q, got %q", LaunchModeBackground, got)
	}
	if got := normalizeLaunchMode(LaunchMode("invalid")); got != LaunchModeNormal {
		t.Fatalf("expected invalid mode to normalize to %q, got %q", LaunchModeNormal, got)
	}
}

func TestLaunchArgsForMode(t *testing.T) {
	if got := launchArgsForMode(LaunchModeNormal); len(got) != 0 {
		t.Fatalf("expected no args for normal mode, got %#v", got)
	}

	got := launchArgsForMode(LaunchModeBackground)
	if len(got) != 1 || got[0] != startHiddenArg {
		t.Fatalf("unexpected args for background mode: %#v", got)
	}
}
