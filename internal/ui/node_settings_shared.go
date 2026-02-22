package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

const nodeSettingsOpTimeout = 12 * time.Second

var nodeSettingsTabLogger = slog.With("component", "ui.node_settings_tab")

type nodeSettingsSaveGate struct {
	mu   sync.Mutex
	page string
}

func (g *nodeSettingsSaveGate) TryAcquire(page string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if strings.TrimSpace(g.page) != "" {
		return false
	}
	g.page = strings.TrimSpace(page)

	return true
}

func (g *nodeSettingsSaveGate) Release(page string) {
	g.mu.Lock()
	if strings.TrimSpace(g.page) == strings.TrimSpace(page) {
		g.page = ""
	}
	g.mu.Unlock()
}

func (g *nodeSettingsSaveGate) ActivePage() string {
	g.mu.Lock()
	active := g.page
	g.mu.Unlock()

	return active
}

func localNodeSnapshot(dep RuntimeDependencies) app.LocalNodeSnapshot {
	if dep.Data.LocalNodeSnapshot != nil {
		snapshot := dep.Data.LocalNodeSnapshot()
		snapshot.ID = strings.TrimSpace(snapshot.ID)
		if snapshot.ID != "" && strings.TrimSpace(snapshot.Node.NodeID) == "" {
			snapshot.Node.NodeID = snapshot.ID
		}

		return snapshot
	}

	if dep.Data.LocalNodeID == nil {
		return app.LocalNodeSnapshot{}
	}
	id := strings.TrimSpace(dep.Data.LocalNodeID())
	if id == "" {
		return app.LocalNodeSnapshot{}
	}

	snapshot := app.LocalNodeSnapshot{
		ID:   id,
		Node: domain.Node{NodeID: id},
	}
	if dep.Data.NodeStore == nil {
		return snapshot
	}
	node, ok := dep.Data.NodeStore.Get(id)
	if !ok {
		return snapshot
	}
	if strings.TrimSpace(node.NodeID) == "" {
		node.NodeID = id
	}
	snapshot.Node = node
	snapshot.Present = true

	return snapshot
}

func isNodeSettingsConnected(dep RuntimeDependencies) bool {
	if dep.Data.CurrentConnStatus == nil {
		return false
	}
	status, known := dep.Data.CurrentConnStatus()
	if !known {
		return false
	}

	return status.State == busmsg.ConnectionStateConnected
}

func localNodeSettingsTarget(dep RuntimeDependencies) (app.NodeSettingsTarget, bool) {
	nodeID := localNodeSnapshot(dep).ID
	if nodeID == "" {
		return app.NodeSettingsTarget{}, false
	}

	return app.NodeSettingsTarget{NodeID: nodeID, IsLocal: true}, true
}

func copyTextToClipboard(value string) error {
	app := fyne.CurrentApp()
	if app == nil || app.Clipboard() == nil {
		return fmt.Errorf("clipboard is unavailable")
	}

	app.Clipboard().SetContent(value)

	return nil
}

func orUnknown(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "unknown"
	}

	return v
}
