package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"

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
		{
			name: "infrastructure node suffix",
			node: domain.Node{
				NodeID:    "!abcd1234",
				ShortName: "ABCD",
				LongName:  "Alpha Bravo",
				IsUnmessageable: func() *bool {
					v := true
					return &v
				}(),
			},
			want: "[ABCD] Alpha Bravo {INFRA}",
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

func TestDefaultNodeRowRenderer_UsesMonospaceIDAndCenteredRole(t *testing.T) {
	renderer := DefaultNodeRowRenderer()
	obj := renderer.Create()
	rssi := -114
	snr := -6.9
	renderer.Update(obj, domain.Node{
		NodeID:     "!abcd1234",
		BoardModel: "T-Echo",
		Role:       "CLIENT",
		RSSI:       &rssi,
		SNR:        &snr,
		BatteryLevel: func() *uint32 {
			v := uint32(80)
			return &v
		}(),
	})

	row, ok := extractNodeRowLabels(obj)
	if !ok {
		t.Fatalf("failed to parse row labels")
	}
	role := row.role
	model := row.model
	id := row.id
	signal := row.signal

	if model.Text != "T-Echo" {
		t.Fatalf("unexpected model text: %q", model.Text)
	}
	if role.Text != "CLIENT" {
		t.Fatalf("unexpected role text: %q", role.Text)
	}
	if !id.TextStyle.Monospace {
		t.Fatalf("node id label should be monospace")
	}
	if role.Alignment != fyne.TextAlignCenter {
		t.Fatalf("role label should be center aligned")
	}
	if !signal.Visible() {
		t.Fatalf("signal should be visible")
	}
	if signal.Text != "▂▄▆█ Good" {
		t.Fatalf("unexpected signal text: %q", signal.Text)
	}
}

func TestNodeLine2Signal(t *testing.T) {
	goodRSSI, goodSNR := -110, -6.0
	fairRSSI, fairSNR := -125, -14.0
	badRSSI, badSNR := -127, -15.5

	tests := []struct {
		name string
		node domain.Node
		want string
		show bool
	}{
		{name: "unknown without metrics", node: domain.Node{}, want: "", show: false},
		{name: "good", node: domain.Node{RSSI: &goodRSSI, SNR: &goodSNR}, want: "▂▄▆█ Good", show: true},
		{name: "fair", node: domain.Node{RSSI: &fairRSSI, SNR: &fairSNR}, want: "▂▄▆  Fair", show: true},
		{name: "bad", node: domain.Node{RSSI: &badRSSI, SNR: &badSNR}, want: "▂▄   Bad", show: true},
	}

	for _, tt := range tests {
		got := nodeLine2Signal(tt.node)
		if got.Text != tt.want {
			t.Fatalf("%s: got %q want %q", tt.name, got.Text, tt.want)
		}
		if got.Visible != tt.show {
			t.Fatalf("%s: got visible=%v want visible=%v", tt.name, got.Visible, tt.show)
		}
	}
}

func TestFilterNodesByName(t *testing.T) {
	nodes := []domain.Node{
		{NodeID: "!00000001", ShortName: "ABCD", LongName: "Alpha Bravo"},
		{NodeID: "!00000002", ShortName: "EFGH", LongName: "Echo Foxtrot"},
		{NodeID: "!00000003", ShortName: "", LongName: "Golf Hotel"},
		{NodeID: "!00000004", ShortName: "IJKL", LongName: ""},
	}

	t.Run("empty filter keeps all", func(t *testing.T) {
		filtered := filterNodesByName(nodes, "")
		if len(filtered) != 4 {
			t.Fatalf("expected all nodes, got %d", len(filtered))
		}
	})

	t.Run("matches short name", func(t *testing.T) {
		filtered := filterNodesByName(nodes, "ef")
		if len(filtered) != 1 || filtered[0].NodeID != "!00000002" {
			t.Fatalf("unexpected filtered result: %+v", filtered)
		}
	})

	t.Run("matches long name case insensitive", func(t *testing.T) {
		filtered := filterNodesByName(nodes, "golf")
		if len(filtered) != 1 || filtered[0].NodeID != "!00000003" {
			t.Fatalf("unexpected filtered result: %+v", filtered)
		}
	})

	t.Run("trim spaces in filter", func(t *testing.T) {
		filtered := filterNodesByName(nodes, "  bravo  ")
		if len(filtered) != 1 || filtered[0].NodeID != "!00000001" {
			t.Fatalf("unexpected filtered result: %+v", filtered)
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		filtered := filterNodesByName(nodes, "zzz")
		if len(filtered) != 0 {
			t.Fatalf("expected empty result, got %d", len(filtered))
		}
	})
}

func TestNodeCountLabelText(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		visible  int
		filter   string
		expected string
	}{
		{name: "no filter shows total", total: 52, visible: 52, filter: "", expected: "Nodes (52)"},
		{name: "whitespace filter counts as empty", total: 52, visible: 52, filter: "  ", expected: "Nodes (52)"},
		{name: "active filter shows visible over total", total: 52, visible: 7, filter: "abc", expected: "Nodes (7/52)"},
	}

	for _, tt := range tests {
		got := nodeCountLabelText(tt.total, tt.visible, tt.filter)
		if got != tt.expected {
			t.Fatalf("%s: got %q want %q", tt.name, got, tt.expected)
		}
	}
}
