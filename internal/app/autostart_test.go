package app

import (
	"errors"
	"testing"
)

func TestAutostartSyncWarningError(t *testing.T) {
	warning := &AutostartSyncWarning{Err: errors.New("boom")}
	if got := warning.Error(); got != "autostart sync failed: boom" {
		t.Fatalf("unexpected warning error text: %q", got)
	}
	if !errors.Is(warning, warning.Err) {
		t.Fatalf("expected warning to unwrap original error")
	}
}
