package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

type mainView struct {
	left                *fyne.Container
	rightStack          *fyne.Container
	sidebar             sidebarLayout
	applyMapTheme       func(fyne.ThemeVariant)
	updateIndicator     *updateIndicator
	connStatusPresenter *connectionStatusPresenter
}

func buildMainView(
	dep RuntimeDependencies,
	fyApp fyne.App,
	window fyne.Window,
	initialVariant fyne.ThemeVariant,
	initialStatus connectors.ConnectionStatus,
) mainView {
	settingsConnStatus := widget.NewLabel("")
	settingsConnStatus.Truncation = fyne.TextTruncateEllipsis

	chatsTab := newChatsTab(
		dep.Data.ChatStore,
		dep.Actions.Sender,
		resolveNodeDisplayName(dep.Data.NodeStore),
		resolveRelayNodeDisplayNameByLastByte(dep.Data.NodeStore),
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
	nodeSettingsTab, onNodeSettingsTabShow := newNodeTabWithOnShow(dep)
	settingsTab := newSettingsTab(dep, settingsConnStatus)

	tabContent := map[string]fyne.CanvasObject{
		"Chats": chatsTab,
		"Nodes": nodesTab,
		"Map":   mapTab,
		"Node":  nodeSettingsTab,
		"App":   settingsTab,
	}
	tabOnShow := map[string]func(){
		"Node": onNodeSettingsTabShow,
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
		false,
		func(snapshot meshapp.UpdateSnapshot) {
			showUpdateDialog(window, fyApp.Settings().ThemeVariant(), snapshot, openExternalURL)
		},
	)
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
		tabOnShow,
		order,
		tabIcons,
		updateIndicator.Button(),
		connStatusPresenter.SidebarIcon(),
	)
	applyMapTheme := func(fyne.ThemeVariant) {}
	if mapWidget, ok := mapTab.(*mapTabWidget); ok {
		applyMapTheme = mapWidget.applyThemeVariant
	}

	return mainView{
		left:                sidebar.left,
		rightStack:          sidebar.rightStack,
		sidebar:             sidebar,
		applyMapTheme:       applyMapTheme,
		updateIndicator:     updateIndicator,
		connStatusPresenter: connStatusPresenter,
	}
}
