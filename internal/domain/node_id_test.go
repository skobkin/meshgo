package domain

import "testing"

func TestNormalizeNodeID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "trim", in: " !1234abcd ", want: "!1234abcd"},
		{name: "empty", in: " ", want: ""},
		{name: "unknown lower", in: "unknown", want: ""},
		{name: "unknown upper", in: "UNKNOWN", want: ""},
		{name: "broadcast placeholder", in: "!ffffffff", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeNodeID(tc.in); got != tc.want {
				t.Fatalf("unexpected normalized value: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNodeIDLastByte(t *testing.T) {
	tests := []struct {
		name   string
		nodeID string
		want   uint32
		ok     bool
	}{
		{name: "lower hex", nodeID: "!1234abcd", want: 0xcd, ok: true},
		{name: "upper hex", nodeID: "!1234ABCD", want: 0xcd, ok: true},
		{name: "trim", nodeID: " !1234abcd ", want: 0xcd, ok: true},
		{name: "invalid char", nodeID: "!1234abcx", want: 0, ok: false},
		{name: "short", nodeID: "!1234abc", want: 0, ok: false},
		{name: "unknown", nodeID: "unknown", want: 0, ok: false},
		{name: "placeholder", nodeID: "!ffffffff", want: 0, ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NodeIDLastByte(tc.nodeID)
			if ok != tc.ok {
				t.Fatalf("unexpected ok flag: got %v want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("unexpected byte: got 0x%02x want 0x%02x", got, tc.want)
			}
		})
	}
}
