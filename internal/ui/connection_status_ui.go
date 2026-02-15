package ui

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

type connectionStatusPresenter struct {
	window         fyne.Window
	statusLabel    *widget.Label
	sidebarIcon    *widget.Icon
	localShortName func() string

	mu      sync.RWMutex
	current connectors.ConnectionStatus
}

func newConnectionStatusPresenter(
	window fyne.Window,
	statusLabel *widget.Label,
	initialStatus connectors.ConnectionStatus,
	initialVariant fyne.ThemeVariant,
	localShortName func() string,
) *connectionStatusPresenter {
	presenter := &connectionStatusPresenter{
		window:         window,
		statusLabel:    statusLabel,
		sidebarIcon:    widget.NewIcon(resources.UIIconResource(sidebarStatusIcon(initialStatus), initialVariant)),
		localShortName: localShortName,
		current:        initialStatus,
	}
	presenter.applyUI(initialStatus, initialVariant)

	return presenter
}

func (p *connectionStatusPresenter) SidebarIcon() *widget.Icon {
	return p.sidebarIcon
}

func (p *connectionStatusPresenter) Set(status connectors.ConnectionStatus, variant fyne.ThemeVariant) {
	p.mu.Lock()
	p.current = status
	p.mu.Unlock()
	p.applyUI(status, variant)
}

func (p *connectionStatusPresenter) Refresh(variant fyne.ThemeVariant) {
	p.mu.RLock()
	status := p.current
	p.mu.RUnlock()
	p.applyUI(status, variant)
}

func (p *connectionStatusPresenter) ApplyTheme(variant fyne.ThemeVariant) {
	p.mu.RLock()
	status := p.current
	p.mu.RUnlock()
	if p.sidebarIcon != nil {
		setConnStatusIcon(p.sidebarIcon, status, variant)
	}
}

func (p *connectionStatusPresenter) CurrentStatus() connectors.ConnectionStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.current
}

func (p *connectionStatusPresenter) applyUI(status connectors.ConnectionStatus, variant fyne.ThemeVariant) {
	localShortName := ""
	if p.localShortName != nil {
		localShortName = p.localShortName()
	}
	applyConnStatusUI(p.window, p.statusLabel, p.sidebarIcon, status, variant, localShortName)
}

func formatConnStatus(status connectors.ConnectionStatus, localShortName string) string {
	text := string(status.State)
	if transportName := transportDisplayName(status.TransportName); transportName != "" {
		text = transportName + " " + text
	}
	details := make([]string, 0, 2)
	if target := strings.TrimSpace(status.Target); target != "" {
		details = append(details, target)
	}
	if shortName := strings.TrimSpace(localShortName); shortName != "" {
		details = append(details, shortName)
	}
	if len(details) > 0 {
		text += " (" + strings.Join(details, ", ") + ")"
	}
	if status.Err != "" {
		text += " (" + status.Err + ")"
	}

	return text
}

func transportDisplayName(name string) string {
	normalized := config.ConnectorType(strings.ToLower(strings.TrimSpace(name)))
	switch normalized {
	case config.ConnectorIP, config.ConnectorSerial, config.ConnectorBluetooth:
		return connectorOptionFromType(normalized)
	default:
		return strings.TrimSpace(name)
	}
}

func formatWindowTitle(status connectors.ConnectionStatus, localShortName string) string {
	return fmt.Sprintf("MeshGo %s - %s", meshapp.BuildVersion(), formatConnStatus(status, localShortName))
}

func applyConnStatusUI(
	window fyne.Window,
	statusLabel *widget.Label,
	sidebarIcon *widget.Icon,
	status connectors.ConnectionStatus,
	variant fyne.ThemeVariant,
	localShortName string,
) {
	if window != nil {
		window.SetTitle(formatWindowTitle(status, localShortName))
	}
	if statusLabel != nil {
		statusLabel.SetText(formatConnStatus(status, localShortName))
	}
	if sidebarIcon != nil {
		setConnStatusIcon(sidebarIcon, status, variant)
	}
}

func setConnStatusIcon(sidebarIcon *widget.Icon, status connectors.ConnectionStatus, variant fyne.ThemeVariant) {
	sidebarIcon.SetResource(resources.UIIconResource(sidebarStatusIcon(status), variant))
}

func sidebarStatusIcon(status connectors.ConnectionStatus) resources.UIIcon {
	if status.State == connectors.ConnectionStateConnected {
		return resources.UIIconConnected
	}

	return resources.UIIconDisconnected
}

func localNodeDisplayName(localNodeID func() string, store *domain.NodeStore) string {
	if localNodeID == nil {
		return ""
	}
	nodeID := strings.TrimSpace(localNodeID())
	if nodeID == "" {
		return ""
	}

	return domain.NodeDisplayNameByID(store, nodeID)
}
