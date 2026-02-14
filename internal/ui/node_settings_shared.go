package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
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

func localNodeSnapshot(store *domain.NodeStore, localNodeID func() string) (domain.Node, bool) {
	if localNodeID == nil {
		return domain.Node{}, false
	}
	id := strings.TrimSpace(localNodeID())
	if id == "" {
		return domain.Node{}, false
	}
	if store == nil {
		return domain.Node{NodeID: id}, false
	}

	node, ok := store.Get(id)
	if !ok {
		return domain.Node{NodeID: id}, false
	}
	if strings.TrimSpace(node.NodeID) == "" {
		node.NodeID = id
	}

	return node, true
}

func isNodeSettingsConnected(dep RuntimeDependencies) bool {
	if dep.Data.CurrentConnStatus == nil {
		return false
	}
	status, known := dep.Data.CurrentConnStatus()
	if !known {
		return false
	}

	return status.State == connectors.ConnectionStateConnected
}

func localNodeSettingsTarget(dep RuntimeDependencies) (app.NodeSettingsTarget, bool) {
	if dep.Data.LocalNodeID == nil {
		return app.NodeSettingsTarget{}, false
	}
	nodeID := strings.TrimSpace(dep.Data.LocalNodeID())
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
