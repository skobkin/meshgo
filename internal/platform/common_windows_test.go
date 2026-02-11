//go:build windows

package platform

import (
	"strings"
	"testing"
)

func TestBuildWindowsCommandLine(t *testing.T) {
	got := buildWindowsCommandLine(`C:\Program Files\meshgo\meshgo.exe`, []string{"--start-hidden"})
	if !strings.Contains(got, `"C:\Program Files\meshgo\meshgo.exe"`) {
		t.Fatalf("expected quoted executable, got %q", got)
	}
	if !strings.Contains(got, "--start-hidden") {
		t.Fatalf("expected start-hidden argument, got %q", got)
	}
}

func TestQuoteWindowsCommandLineArg(t *testing.T) {
	if got := quoteWindowsCommandLineArg("plain"); got != "plain" {
		t.Fatalf("expected unchanged plain arg, got %q", got)
	}
	if got := quoteWindowsCommandLineArg("with space"); got != `"with space"` {
		t.Fatalf("expected quoted arg, got %q", got)
	}
}
