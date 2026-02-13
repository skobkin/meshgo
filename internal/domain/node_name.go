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

func NodeDisplayNameByID(store *NodeStore, nodeID string) string {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return ""
	}
	if store == nil {
		return nodeID
	}
	node, ok := store.Get(nodeID)
	if !ok {
		return nodeID
	}
	if display := strings.TrimSpace(NodeDisplayName(node)); display != "" {
		return display
	}

	return nodeID
}
