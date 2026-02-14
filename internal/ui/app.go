package ui

import (
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

var appLogger = slog.With("component", "ui.app")

func Run(dep RuntimeDependencies) error {
	fyApp := fyneapp.NewWithID("meshgo")
	initialVariant := fyApp.Settings().ThemeVariant()
	fyApp.SetIcon(resources.AppIconResource(initialVariant))
	appLogger.Info(
		"starting UI runtime",
		"start_hidden", dep.Launch.StartHidden,
		"initial_theme", initialVariant,
	)

	initialStatus := resolveInitialConnStatus(dep)

	initialUpdateSnapshot, initialUpdateSnapshotKnown := currentUpdateSnapshot(dep)

	window := fyApp.NewWindow("")
	window.Resize(fyne.NewSize(1000, 700))

	settingsConnStatus := widget.NewLabel("")
	settingsConnStatus.Truncation = fyne.TextTruncateEllipsis

	chatsTab := newChatsTab(
		dep.Data.ChatStore,
		dep.Actions.Sender,
		resolveNodeDisplayName(dep.Data.NodeStore),
		dep.Data.LocalNodeID,
		nodeChanges(dep.Data.NodeStore),
		dep.Data.LastSelectedChat,
		dep.Actions.OnChatSelected,
	)
	nodeActionHandler := func(node domain.Node, action NodeAction) {
		if action != NodeActionTraceroute {
			return
		}
		handleNodeTracerouteAction(window, dep, node)
	}
	nodesTab := newNodesTabWithActions(dep.Data.NodeStore, DefaultNodeRowRenderer(), NodesTabActions{
		OnNodeSecondaryTapped: func(node domain.Node, position fyne.Position) {
			showNodeContextMenu(window.Canvas(), position, node, nodeActionHandler)
		},
	})
	mapTab := newMapTab(dep.Data.NodeStore, dep.Data.LocalNodeID, dep.Data.Paths, dep.Data.Config.UI.MapViewport, initialVariant, dep.Actions.OnMapViewportChanged)
	nodeSettingsTab := newNodeTab(dep)
	settingsTab := newSettingsTab(dep, settingsConnStatus)

	tabContent := map[string]fyne.CanvasObject{
		"Chats": chatsTab,
		"Nodes": nodesTab,
		"Map":   mapTab,
		"Node":  nodeSettingsTab,
		"App":   settingsTab,
	}
	order := []string{"Chats", "Nodes", "Map", "Node", "App"}
	tabIcons := map[string]resources.UIIcon{
		"Chats": resources.UIIconChats,
		"Nodes": resources.UIIconNodes,
		"Map":   resources.UIIconMap,
		"Node":  resources.UIIconNodeSettings,
		"App":   resources.UIIconAppSettings,
	}

	updateIndicator := newUpdateIndicator(
		initialVariant,
		initialUpdateSnapshot,
		initialUpdateSnapshotKnown,
		func(snapshot meshapp.UpdateSnapshot) {
			showUpdateDialog(window, fyApp.Settings().ThemeVariant(), snapshot, openExternalURL)
		},
	)
	updateButton := updateIndicator.Button()

	connStatusPresenter := newConnectionStatusPresenter(
		window,
		settingsConnStatus,
		initialStatus,
		initialVariant,
		func() string {
			return localNodeDisplayName(dep.Data.LocalNodeID, dep.Data.NodeStore)
		},
	)
	sidebar := buildSidebarLayout(
		initialVariant,
		tabContent,
		order,
		tabIcons,
		updateButton,
		connStatusPresenter.SidebarIcon(),
	)
	left := sidebar.left
	rightStack := sidebar.rightStack

	themeRuntime := newThemeRuntime(fyApp, sidebar, updateIndicator, mapTab, connStatusPresenter)
	themeRuntime.BindSettings()

	stopNotifications := startNotificationService(dep, fyApp, dep.Launch.StartHidden)

	appLogger.Debug("starting UI event listeners")
	stopUIListeners := startUIEventListeners(
		dep.Data.Bus,
		func(status connectors.ConnectionStatus) {
			fyne.Do(func() {
				connStatusPresenter.Set(status, fyApp.Settings().ThemeVariant())
			})
		},
		func() {
			fyne.Do(func() {
				connStatusPresenter.Refresh(fyApp.Settings().ThemeVariant())
			})
		},
	)
	if status, ok := currentConnStatus(dep); ok {
		connStatusPresenter.Set(status, fyApp.Settings().ThemeVariant())
	}
	appLogger.Debug("starting update snapshot listener")
	stopUpdateSnapshots := startUpdateSnapshotListener(dep.Data.UpdateSnapshots, func(snapshot meshapp.UpdateSnapshot) {
		fyne.Do(func() {
			updateIndicator.ApplySnapshot(snapshot)
			left.Refresh()
		})
	})
	if snapshot, ok := currentUpdateSnapshot(dep); ok {
		updateIndicator.ApplySnapshot(snapshot)
		left.Refresh()
	}

	content := container.NewBorder(nil, nil, left, nil, rightStack)
	window.SetContent(content)

	uiRuntime := newUIRuntime(
		fyApp,
		window,
		stopNotifications,
		stopUIListeners,
		stopUpdateSnapshots,
		dep.Actions.OnQuit,
	)
	uiRuntime.BindCloseIntercept()

	setTrayIcon := configureSystemTray(fyApp, window, initialVariant, uiRuntime.Quit)
	themeRuntime.SetTrayIconSetter(setTrayIcon)
	themeRuntime.Apply(initialVariant)

	uiRuntime.Run(dep.Launch.StartHidden)

	return nil
}

func initialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
	return meshapp.ConnectionStatusFromConfig(dep.Data.Config.Connection)
}

func resolveInitialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
	fallback := initialConnStatus(dep)
	status, ok := currentConnStatus(dep)
	if !ok || status.State == "" {
		return fallback
	}
	if strings.TrimSpace(status.TransportName) == "" {
		status.TransportName = fallback.TransportName
	}
	if strings.TrimSpace(status.Target) == "" {
		status.Target = fallback.Target
	}

	return status
}

func currentConnStatus(dep RuntimeDependencies) (connectors.ConnectionStatus, bool) {
	if dep.Data.CurrentConnStatus == nil {
		return connectors.ConnectionStatus{}, false
	}

	return dep.Data.CurrentConnStatus()
}

func resolveNodeDisplayName(store *domain.NodeStore) func(string) string {
	if store == nil {
		return nil
	}

	return func(nodeID string) string {
		return domain.NodeDisplayNameByID(store, nodeID)
	}
}

func nodeChanges(store *domain.NodeStore) <-chan struct{} {
	if store == nil {
		return nil
	}

	return store.Changes()
}

func currentUpdateSnapshot(dep RuntimeDependencies) (meshapp.UpdateSnapshot, bool) {
	if dep.Data.CurrentUpdateSnapshot == nil {
		return meshapp.UpdateSnapshot{}, false
	}

	return dep.Data.CurrentUpdateSnapshot()
}
