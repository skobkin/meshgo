package ui

import (
	"testing"
	"time"

	"github.com/skobkin/meshgo/internal/domain"
)

func TestNodeDisplayName(t *testing.T) {
	tests := []struct {
		name string
		node domain.Node
		want string
	}{
		{
			name: "short and long",
			node: domain.Node{NodeID: "!abcd1234", ShortName: "ABCD", LongName: "Alpha Bravo"},
			want: "[ABCD] Alpha Bravo",
		},
		{
			name: "long only",
			node: domain.Node{NodeID: "!abcd1234", LongName: "Alpha Bravo"},
			want: "Alpha Bravo",
		},
		{
			name: "short only",
			node: domain.Node{NodeID: "!abcd1234", ShortName: "ABCD"},
			want: "[ABCD]",
		},
		{
			name: "fallback id",
			node: domain.Node{NodeID: "!abcd1234"},
			want: "!abcd1234",
		},
	}

	for _, tt := range tests {
		if got := nodeDisplayName(tt.node); got != tt.want {
			t.Fatalf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}

func TestNodeCharge(t *testing.T) {
	level := uint32(75)
	if got := nodeCharge(domain.Node{BatteryLevel: &level}); got != "Charge: 75%" {
		t.Fatalf("unexpected charge line: %q", got)
	}
	external := uint32(101)
	if got := nodeCharge(domain.Node{BatteryLevel: &external}); got != "Charge: ext" {
		t.Fatalf("unexpected external power line: %q", got)
	}
	if got := nodeCharge(domain.Node{}); got != "" {
		t.Fatalf("unexpected empty charge line: %q", got)
	}
}

func TestFormatSeenAgo(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	if got := formatSeenAgo(now.Add(-90*time.Second), now); got != "2 min" {
		t.Fatalf("unexpected minutes line: %q", got)
	}
	if got := formatSeenAgo(now.Add(-110*time.Minute), now); got != "2 hours" {
		t.Fatalf("unexpected hours line: %q", got)
	}
	if got := formatSeenAgo(now.Add(-(72*time.Hour + 2*time.Hour)), now); got != "3 days" {
		t.Fatalf("unexpected days line: %q", got)
	}
}
