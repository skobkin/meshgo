package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/traceroute"
)

func TestTracerouteHopSignalLabel(t *testing.T) {
	if got := tracerouteHopSignalLabel(-128); got != "SNR: ?" {
		t.Fatalf("expected unknown marker, got %q", got)
	}
	if got := tracerouteHopSignalLabel(22); got != "SNR: 5.50 dB" {
		t.Fatalf("unexpected SNR label: %q", got)
	}
	if got := tracerouteHopSignalLabel(-101); got != "SNR: -25.25 dB" {
		t.Fatalf("unexpected SNR label: %q", got)
	}
}

func TestFormatTraceroutePath_InvalidSnrLengthUsesUnknowns(t *testing.T) {
	text := formatTraceroutePath([]string{"!00000001", "!00000002"}, nil, nil)
	if !strings.Contains(text, "■ !00000001") {
		t.Fatalf("expected source node line in path: %q", text)
	}
	if !strings.Contains(text, "⇊ SNR: ?") {
		t.Fatalf("expected unknown snr marker in path: %q", text)
	}
	if !strings.Contains(text, "■ !00000002") {
		t.Fatalf("expected destination node line in path: %q", text)
	}
}

func TestTracerouteProgressValue_CompletedAlwaysFull(t *testing.T) {
	update := connectors.TracerouteUpdate{Status: traceroute.StatusCompleted}
	if got := tracerouteProgressValue(update, 5*time.Second); got != 1 {
		t.Fatalf("expected completed progress to be full, got %f", got)
	}
}

func TestTracerouteProgressValue_ClampsBounds(t *testing.T) {
	update := connectors.TracerouteUpdate{Status: traceroute.StatusProgress}
	if got := tracerouteProgressValue(update, -1*time.Second); got != 0 {
		t.Fatalf("expected negative progress clamp to 0, got %f", got)
	}
	if got := tracerouteProgressValue(update, app.DefaultTracerouteRequestTimeout*2); got != 1 {
		t.Fatalf("expected over-time progress clamp to 1, got %f", got)
	}
}

func TestFormatTracerouteResults(t *testing.T) {
	text := formatTracerouteResults("forward-body", "return-body")
	if !strings.Contains(text, "Route traced toward destination:\nforward-body") {
		t.Fatalf("expected forward section in result text: %q", text)
	}
	if !strings.Contains(text, "Route traced back to us:\nreturn-body") {
		t.Fatalf("expected return section in result text: %q", text)
	}
}

func TestIsTracerouteCopyAvailable(t *testing.T) {
	if isTracerouteCopyAvailable(traceroute.StatusStarted) {
		t.Fatalf("copy must be unavailable while traceroute is started")
	}
	if isTracerouteCopyAvailable(traceroute.StatusProgress) {
		t.Fatalf("copy must be unavailable while traceroute is in progress")
	}
	if !isTracerouteCopyAvailable(traceroute.StatusCompleted) {
		t.Fatalf("copy must be available when traceroute is completed")
	}
	if !isTracerouteCopyAvailable(traceroute.StatusFailed) {
		t.Fatalf("copy must be available when traceroute failed")
	}
	if !isTracerouteCopyAvailable(traceroute.StatusTimedOut) {
		t.Fatalf("copy must be available when traceroute timed out")
	}
}

func TestTracerouteStatusText(t *testing.T) {
	cases := []struct {
		name   string
		status traceroute.Status
		want   string
	}{
		{name: "started", status: traceroute.StatusStarted, want: "Started"},
		{name: "progress", status: traceroute.StatusProgress, want: "Waiting"},
		{name: "completed", status: traceroute.StatusCompleted, want: "Complete"},
		{name: "failed", status: traceroute.StatusFailed, want: "Failed"},
		{name: "timed out", status: traceroute.StatusTimedOut, want: "Timed out"},
		{name: "unknown", status: "unknown", want: "Update"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			update := connectors.TracerouteUpdate{Status: tc.status}
			if got := tracerouteStatusText(update); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
