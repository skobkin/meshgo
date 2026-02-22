package app

import (
	"strings"

	"github.com/skobkin/meshgo/internal/domain"
)

// LocalNodeSnapshot represents current local-node identity and cached store state.
type LocalNodeSnapshot struct {
	ID      string
	Node    domain.Node
	Present bool
}

func (r *Runtime) LocalNodeID() string {
	if r == nil || r.Connectivity.Radio == nil {
		return ""
	}

	return strings.TrimSpace(r.Connectivity.Radio.LocalNodeID())
}

func (r *Runtime) LocalNodeSnapshot() LocalNodeSnapshot {
	localID := r.LocalNodeID()
	if localID == "" {
		return LocalNodeSnapshot{}
	}

	snapshot := LocalNodeSnapshot{
		ID:   localID,
		Node: domain.Node{NodeID: localID},
	}
	if r == nil || r.Domain.NodeStore == nil {
		return snapshot
	}
	node, ok := r.Domain.NodeStore.Get(localID)
	if !ok {
		return snapshot
	}
	if strings.TrimSpace(node.NodeID) == "" {
		node.NodeID = localID
	}
	snapshot.Node = node
	snapshot.Present = true

	return snapshot
}
