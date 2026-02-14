package ui

import (
	"fmt"
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

func resolveNodeDisplayName(store *domain.NodeStore) func(string) string {
	if store == nil {
		return nil
	}

	return func(nodeID string) string {
		return domain.NodeDisplayNameByID(store, nodeID)
	}
}

func resolveRelayNodeDisplayNameByLastByte(store *domain.NodeStore) func(uint32) string {
	if store == nil {
		return nil
	}

	return func(relayNode uint32) string {
		relayByte := relayNode & 0xff
		if relayByte == 0 {
			return ""
		}

		var matched domain.Node
		matchCount := 0
		for _, node := range store.SnapshotSorted() {
			nodeByte, ok := domain.NodeIDLastByte(node.NodeID)
			if !ok || nodeByte != relayByte {
				continue
			}
			matched = node
			matchCount++
			if matchCount > 1 {
				return ""
			}
		}
		if matchCount != 1 {
			return ""
		}

		return strings.TrimSpace(domain.NodeDisplayName(matched))
	}
}

func resolveRelayNodeDisplay(
	meta messageMeta,
	hasMeta bool,
	nodeNameByID func(string) string,
	relayNodeNameByLastByte func(uint32) string,
) string {
	if !hasMeta || meta.RelayNode == nil {
		return ""
	}

	relayByte := *meta.RelayNode & 0xff
	if relayByte == 0 {
		return ""
	}
	if from := resolveRelayDisplayFromNodeID(relayByte, meta.From, nodeNameByID); from != "" {
		return from
	}
	if to := resolveRelayDisplayFromNodeID(relayByte, meta.To, nodeNameByID); to != "" {
		return to
	}
	if relayNodeNameByLastByte != nil {
		if relay := strings.TrimSpace(relayNodeNameByLastByte(relayByte)); relay != "" {
			return relay
		}
	}

	return fmt.Sprintf("0x%02x", relayByte)
}

func resolveRelayDisplayFromNodeID(relayByte uint32, nodeID string, nodeNameByID func(string) string) string {
	nodeID = domain.NormalizeNodeID(nodeID)
	if nodeID == "" {
		return ""
	}
	lastByte, ok := domain.NodeIDLastByte(nodeID)
	if !ok || lastByte != relayByte {
		return ""
	}

	return displaySender(nodeID, nodeNameByID)
}
