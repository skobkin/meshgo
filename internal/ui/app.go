package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/bus"
	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

const sidebarConnIconSize float32 = 32

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
	currentStatus := initialStatus
	var connStatusMu sync.RWMutex

	latestUpdateSnapshot, latestUpdateSnapshotKnown := currentUpdateSnapshot(dep)

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

	rightStack := container.NewStack()
	for _, key := range order {
		rightStack.Add(tabContent[key])
		tabContent[key].Hide()
	}
	active := "Chats"
	tabContent[active].Show()

	navButtons := make(map[string]*iconNavButton, len(order))
	disabledTabs := map[string]bool{}

	updateNavSelection := func() {
		for name, button := range navButtons {
			button.SetSelected(name == active && !button.Disabled())
		}
	}

	switchTab := func(name string) {
		if name == active {
			return
		}
		appLogger.Debug("switching sidebar tab", "from", active, "to", name)
		tabContent[active].Hide()
		active = name
		tabContent[active].Show()
		if onShow, ok := tabContent[active].(interface{ OnShow() }); ok {
			onShow.OnShow()
		}
		updateNavSelection()
		rightStack.Refresh()
	}

	left := container.NewVBox()
	for _, name := range order {
		nameCopy := name
		button := newIconNavButton(resources.UIIconResource(tabIcons[name], initialVariant), 48, func() {
			switchTab(nameCopy)
		})
		if disabledTabs[name] {
			button.Disable()
		}
		navButtons[name] = button
		left.Add(button)
	}

	updateButton := newIconNavButton(resources.UIIconResource(resources.UIIconUpdateAvailable, initialVariant), 48, func() {
		if !latestUpdateSnapshotKnown || !latestUpdateSnapshot.UpdateAvailable {
			appLogger.Debug("update button tap ignored: no available update")

			return
		}
		appLogger.Info(
			"opening update dialog",
			"current_version", strings.TrimSpace(latestUpdateSnapshot.CurrentVersion),
			"latest_version", strings.TrimSpace(latestUpdateSnapshot.Latest.Version),
			"release_count", len(latestUpdateSnapshot.Releases),
		)
		showUpdateDialog(window, fyApp.Settings().ThemeVariant(), latestUpdateSnapshot, openExternalURL)
	})
	if latestUpdateSnapshotKnown && latestUpdateSnapshot.UpdateAvailable {
		updateButton.SetText(latestUpdateSnapshot.Latest.Version)
	}
	if !latestUpdateSnapshotKnown || !latestUpdateSnapshot.UpdateAvailable {
		updateButton.Hide()
	}

	updateNavSelection()
	left.Add(layout.NewSpacer())
	left.Add(updateButton)
	sidebarConnIcon := widget.NewIcon(resources.UIIconResource(sidebarStatusIcon(currentStatus), initialVariant))
	left.Add(container.NewCenter(container.NewGridWrap(
		fyne.NewSquareSize(sidebarConnIconSize),
		sidebarConnIcon,
	)))

	setConnStatus := func(status connectors.ConnectionStatus) {
		connStatusMu.Lock()
		currentStatus = status
		connStatusMu.Unlock()
		applyConnStatusUI(
			window,
			settingsConnStatus,
			sidebarConnIcon,
			status,
			fyApp.Settings().ThemeVariant(),
			localNodeDisplayName(dep.Data.LocalNodeID, dep.Data.NodeStore),
		)
	}
	setConnStatus(initialStatus)

	setTrayIcon := func(_ fyne.ThemeVariant) {}
	applyThemeResources := func(variant fyne.ThemeVariant) {
		appLogger.Debug("applying theme resources", "theme", variant)
		connStatusMu.RLock()
		status := currentStatus
		connStatusMu.RUnlock()
		fyApp.SetIcon(resources.AppIconResource(variant))
		setTrayIcon(variant)
		for tabName, button := range navButtons {
			icon := resources.UIIconResource(tabIcons[tabName], variant)
			button.SetIcon(icon)
		}
		updateButton.SetIcon(resources.UIIconResource(resources.UIIconUpdateAvailable, variant))
		if mapWidget, ok := mapTab.(*mapTabWidget); ok {
			mapWidget.applyThemeVariant(variant)
		}
		setConnStatusIcon(sidebarConnIcon, status, variant)
	}

	applyUpdateSnapshot := func(snapshot meshapp.UpdateSnapshot) {
		prevSnapshot := latestUpdateSnapshot
		prevKnown := latestUpdateSnapshotKnown
		latestUpdateSnapshot = snapshot
		latestUpdateSnapshotKnown = true
		if !prevKnown || prevSnapshot.UpdateAvailable != snapshot.UpdateAvailable || prevSnapshot.Latest.Version != snapshot.Latest.Version {
			appLogger.Info(
				"applied update snapshot",
				"current_version", strings.TrimSpace(snapshot.CurrentVersion),
				"latest_version", strings.TrimSpace(snapshot.Latest.Version),
				"update_available", snapshot.UpdateAvailable,
				"release_count", len(snapshot.Releases),
			)
		} else {
			appLogger.Debug(
				"refreshed unchanged update snapshot",
				"latest_version", strings.TrimSpace(snapshot.Latest.Version),
				"update_available", snapshot.UpdateAvailable,
			)
		}
		if snapshot.UpdateAvailable {
			updateButton.SetText(snapshot.Latest.Version)
			updateButton.Show()
		} else {
			updateButton.SetText("")
			updateButton.Hide()
		}
		left.Refresh()
	}

	fyApp.Settings().AddListener(func(_ fyne.Settings) {
		appLogger.Debug("theme settings changed")
		applyThemeResources(fyApp.Settings().ThemeVariant())
	})

	var appForeground atomic.Bool
	appForeground.Store(!dep.Launch.StartHidden)
	fyApp.Lifecycle().SetOnEnteredForeground(func() {
		appForeground.Store(true)
	})
	fyApp.Lifecycle().SetOnExitedForeground(func() {
		appForeground.Store(false)
	})
	notificationsCtx, stopNotifications := context.WithCancel(context.Background())
	notificationService := meshapp.NewNotificationService(
		dep.Data.Bus,
		dep.Data.ChatStore,
		dep.Data.NodeStore,
		dep.Data.CurrentConfig,
		appForeground.Load,
		NewFyneNotificationSender(fyApp),
		slog.With("component", "ui.notifications"),
	)
	notificationService.Start(notificationsCtx)

	appLogger.Debug("starting UI event listeners")
	stopUIListeners := startUIEventListeners(
		dep.Data.Bus,
		func(status connectors.ConnectionStatus) {
			fyne.Do(func() {
				setConnStatus(status)
			})
		},
		func() {
			fyne.Do(func() {
				connStatusMu.RLock()
				status := currentStatus
				connStatusMu.RUnlock()
				setConnStatus(status)
			})
		},
	)
	if status, ok := currentConnStatus(dep); ok {
		setConnStatus(status)
	}
	appLogger.Debug("starting update snapshot listener")
	stopUpdateSnapshots := startUpdateSnapshotListener(dep.Data.UpdateSnapshots, func(snapshot meshapp.UpdateSnapshot) {
		fyne.Do(func() {
			applyUpdateSnapshot(snapshot)
		})
	})
	if snapshot, ok := currentUpdateSnapshot(dep); ok {
		applyUpdateSnapshot(snapshot)
	}

	content := container.NewBorder(nil, nil, left, nil, rightStack)
	window.SetContent(content)

	var shutdownOnce sync.Once
	quit := func() {
		shutdownOnce.Do(func() {
			appLogger.Info("quitting UI runtime")
			stopNotifications()
			stopUIListeners()
			stopUpdateSnapshots()
			if dep.Actions.OnQuit != nil {
				dep.Actions.OnQuit()
			}
			fyApp.Quit()
		})
	}

	window.SetCloseIntercept(func() {
		appLogger.Debug("main window close intercepted: hiding to tray")
		window.Hide()
	})

	if desk, ok := fyApp.(desktop.App); ok {
		setTrayIcon = func(variant fyne.ThemeVariant) {
			desk.SetSystemTrayIcon(resources.TrayIconResource(variant))
		}
		setTrayIcon(initialVariant)
		desk.SetSystemTrayMenu(fyne.NewMenu("meshgo",
			fyne.NewMenuItem("Show", func() {
				appLogger.Debug("system tray show action invoked")
				window.Show()
				window.RequestFocus()
			}),
			fyne.NewMenuItem("Quit", func() {
				appLogger.Debug("system tray quit action invoked")
				quit()
			}),
		))
	}
	applyThemeResources(initialVariant)

	window.Show()
	if dep.Launch.StartHidden {
		appLogger.Info("launch option start_hidden is enabled: hiding main window")
		window.Hide()
	}
	fyApp.Run()
	appLogger.Info("UI runtime stopped")
	shutdownOnce.Do(func() {
		stopNotifications()
		stopUIListeners()
		stopUpdateSnapshots()
		if dep.Actions.OnQuit != nil {
			dep.Actions.OnQuit()
		}
	})

	return nil
}

func startUIEventListeners(
	messageBus bus.MessageBus,
	onConnStatus func(connectors.ConnectionStatus),
	onNodeInfo func(),
) func() {
	if messageBus == nil {
		appLogger.Debug("skipping UI event listeners: message bus is nil")

		return func() {}
	}

	connSub := messageBus.Subscribe(connectors.TopicConnStatus)
	nodeSub := messageBus.Subscribe(connectors.TopicNodeInfo)
	appLogger.Debug(
		"subscribed to UI bus topics",
		"topics", []string{connectors.TopicConnStatus, connectors.TopicNodeInfo},
	)
	done := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		for {
			select {
			case <-done:
				return
			case raw, ok := <-connSub:
				if !ok {
					appLogger.Debug("connection status subscription closed")

					return
				}
				status, ok := raw.(connectors.ConnectionStatus)
				if !ok {
					appLogger.Debug("ignoring unexpected connection status payload", "payload_type", fmt.Sprintf("%T", raw))

					continue
				}
				select {
				case <-done:
					return
				default:
				}
				if onConnStatus != nil {
					onConnStatus(status)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case _, ok := <-nodeSub:
				if !ok {
					appLogger.Debug("node info subscription closed")

					return
				}
				select {
				case <-done:
					return
				default:
				}
				if onNodeInfo != nil {
					onNodeInfo()
				}
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			appLogger.Debug("stopping UI event listeners")
			close(done)
			messageBus.Unsubscribe(connSub, connectors.TopicConnStatus)
			messageBus.Unsubscribe(nodeSub, connectors.TopicNodeInfo)
		})
	}
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
	window.SetTitle(formatWindowTitle(status, localShortName))
	statusLabel.SetText(formatConnStatus(status, localShortName))
	setConnStatusIcon(sidebarIcon, status, variant)
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

func resolveNodeDisplayName(store *domain.NodeStore) func(string) string {
	if store == nil {
		return nil
	}

	return func(nodeID string) string {
		return domain.NodeDisplayNameByID(store, nodeID)
	}
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

func startUpdateSnapshotListener(
	snapshots <-chan meshapp.UpdateSnapshot,
	onSnapshot func(meshapp.UpdateSnapshot),
) func() {
	if snapshots == nil {
		appLogger.Debug("skipping update snapshot listener: channel is nil")

		return func() {}
	}

	done := make(chan struct{})
	var stopOnce sync.Once

	go func() {
		for {
			select {
			case <-done:
				return
			case snapshot, ok := <-snapshots:
				if !ok {
					appLogger.Debug("update snapshot channel closed")

					return
				}
				select {
				case <-done:
					return
				default:
				}
				if onSnapshot != nil {
					onSnapshot(snapshot)
				}
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			appLogger.Debug("stopping update snapshot listener")
			close(done)
		})
	}
}
