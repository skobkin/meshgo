package domain

import "strings"

func NodeDisplayName(node Node) string {
	if value := strings.TrimSpace(node.LongName); value != "" {
		return value
	}
	if value := strings.TrimSpace(node.ShortName); value != "" {
		return value
	}
	return strings.TrimSpace(node.NodeID)
}
