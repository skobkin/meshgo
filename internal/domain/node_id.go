package domain

import "strings"

// NormalizeNodeID trims and rejects placeholder/unknown node ids.
func NormalizeNodeID(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, "unknown") || v == "!ffffffff" {
		return ""
	}

	return v
}

// NodeIDLastByte extracts the last byte from canonical "!1234abcd" node ids.
func NodeIDLastByte(nodeID string) (uint32, bool) {
	nodeID = NormalizeNodeID(nodeID)
	if len(nodeID) != 9 || nodeID[0] != '!' {
		return 0, false
	}

	high, ok := hexNibble(nodeID[7])
	if !ok {
		return 0, false
	}
	low, ok := hexNibble(nodeID[8])
	if !ok {
		return 0, false
	}

	return (high << 4) | low, true
}

func hexNibble(ch byte) (uint32, bool) {
	switch {
	case ch >= '0' && ch <= '9':
		return uint32(ch - '0'), true
	case ch >= 'a' && ch <= 'f':
		return uint32(ch-'a') + 10, true
	case ch >= 'A' && ch <= 'F':
		return uint32(ch-'A') + 10, true
	default:
		return 0, false
	}
}
