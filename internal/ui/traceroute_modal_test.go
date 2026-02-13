package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
)

func TestTracerouteHopSignalLabel(t *testing.T) {
	if got := tracerouteHopSignalLabel(-128); got != "SNR: ?" {
		t.Fatalf("expected unknown marker, got %q", got)
	}
	if got := tracerouteHopSignalLabel(22); got != "SNR: 5.50 dB" {
		t.Fatalf("unexpected SNR label: %q", got)
	}
	if got := tracerouteHopSignalLabel(-101); got != "RSSI: -101 dBm" {
		t.Fatalf("unexpected RSSI label: %q", got)
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
	update := connectors.TracerouteUpdate{Status: connectors.TracerouteStatusCompleted}
	if got := tracerouteProgressValue(update, 5*time.Second); got != 1 {
		t.Fatalf("expected completed progress to be full, got %f", got)
	}
}

func TestTracerouteProgressValue_ClampsBounds(t *testing.T) {
	update := connectors.TracerouteUpdate{Status: connectors.TracerouteStatusProgress}
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
	if isTracerouteCopyAvailable(connectors.TracerouteStatusStarted) {
		t.Fatalf("copy must be unavailable while traceroute is started")
	}
	if isTracerouteCopyAvailable(connectors.TracerouteStatusProgress) {
		t.Fatalf("copy must be unavailable while traceroute is in progress")
	}
	if !isTracerouteCopyAvailable(connectors.TracerouteStatusCompleted) {
		t.Fatalf("copy must be available when traceroute is completed")
	}
	if !isTracerouteCopyAvailable(connectors.TracerouteStatusFailed) {
		t.Fatalf("copy must be available when traceroute failed")
	}
	if !isTracerouteCopyAvailable(connectors.TracerouteStatusTimedOut) {
		t.Fatalf("copy must be available when traceroute timed out")
	}
}

func TestTracerouteStatusText(t *testing.T) {
	cases := []struct {
		name   string
		status connectors.TracerouteStatus
		want   string
	}{
		{name: "started", status: connectors.TracerouteStatusStarted, want: "Started"},
		{name: "progress", status: connectors.TracerouteStatusProgress, want: "Waiting"},
		{name: "completed", status: connectors.TracerouteStatusCompleted, want: "Complete"},
		{name: "failed", status: connectors.TracerouteStatusFailed, want: "Failed"},
		{name: "timed out", status: connectors.TracerouteStatusTimedOut, want: "Timed out"},
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

func TestTracerouteHopSignalIsRSSI(t *testing.T) {
	if tracerouteHopSignalIsRSSI(-80) {
		t.Fatalf("expected -80 to be treated as SNR-scaled value")
	}
	if !tracerouteHopSignalIsRSSI(-81) {
		t.Fatalf("expected values below -80 to be treated as RSSI")
	}
}
