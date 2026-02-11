package ui

import (
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

func Run(dep Dependencies) error {
	fyApp := app.NewWithID("meshgo")
	initialVariant := fyApp.Settings().ThemeVariant()
	fyApp.SetIcon(resources.AppIconResource(initialVariant))

	initialStatus := initialConnStatus(dep)
	window := fyApp.NewWindow(formatWindowTitle(initialStatus))
	window.Resize(fyne.NewSize(1000, 700))

	settingsConnStatus := widget.NewLabel(formatConnStatus(initialStatus))
	settingsConnStatus.Truncation = fyne.TextTruncateEllipsis

	chatsTab := newChatsTab(
		dep.ChatStore,
		dep.Sender,
		resolveNodeDisplayName(dep.NodeStore),
		dep.LocalNodeID,
		nodeChanges(dep.NodeStore),
		dep.LastSelectedChat,
		dep.OnChatSelected,
	)
	nodesTab := newNodesTab(dep.NodeStore, DefaultNodeRowRenderer())
	mapTab := disabledTab("Map is not implemented yet")
	nodeSettingsTab := newNodeTab(dep.NodeStore, dep.LocalNodeID)
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
	disabledTabs := map[string]bool{
		"Map": true,
	}

	updateNavSelection := func() {
		for name, button := range navButtons {
			button.SetSelected(name == active && !button.Disabled())
		}
	}

	switchTab := func(name string) {
		if name == active {
			return
		}
		tabContent[active].Hide()
		active = name
		tabContent[active].Show()
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
	updateNavSelection()
	left.Add(layout.NewSpacer())

	setTrayIcon := func(_ fyne.ThemeVariant) {}
	applyThemeResources := func(variant fyne.ThemeVariant) {
		fyApp.SetIcon(resources.AppIconResource(variant))
		setTrayIcon(variant)
		for tabName, button := range navButtons {
			icon := resources.UIIconResource(tabIcons[tabName], variant)
			button.SetIcon(icon)
		}
	}

	fyApp.Settings().AddListener(func(_ fyne.Settings) {
		applyThemeResources(fyApp.Settings().ThemeVariant())
	})

	if dep.Bus != nil {
		sub := dep.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for raw := range sub {
				status, ok := raw.(connectors.ConnStatus)
				if !ok {
					continue
				}
				text := formatConnStatus(status)
				fyne.Do(func() {
					window.SetTitle(formatWindowTitle(status))
					settingsConnStatus.SetText(text)
				})
			}
		}()
	}

	content := container.NewBorder(nil, nil, left, nil, rightStack)
	window.SetContent(content)

	var shutdownOnce sync.Once
	quit := func() {
		shutdownOnce.Do(func() {
			if dep.OnQuit != nil {
				dep.OnQuit()
			}
			fyApp.Quit()
		})
	}

	window.SetCloseIntercept(func() {
		window.Hide()
	})

	if desk, ok := fyApp.(desktop.App); ok {
		setTrayIcon = func(variant fyne.ThemeVariant) {
			desk.SetSystemTrayIcon(resources.TrayIconResource(variant))
		}
		setTrayIcon(initialVariant)
		desk.SetSystemTrayMenu(fyne.NewMenu("meshgo",
			fyne.NewMenuItem("Show", func() {
				window.Show()
				window.RequestFocus()
			}),
			fyne.NewMenuItem("Quit", func() {
				quit()
			}),
		))
	}
	applyThemeResources(initialVariant)

	window.Show()
	fyApp.Run()
	shutdownOnce.Do(func() {
		if dep.OnQuit != nil {
			dep.OnQuit()
		}
	})
	return nil
}

func disabledTab(text string) fyne.CanvasObject {
	return container.NewCenter(widget.NewLabel(text))
}

func initialConnStatus(dep Dependencies) connectors.ConnStatus {
	status := connectors.ConnStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: "unknown",
	}
	switch dep.Config.Connection.Connector {
	case config.ConnectorIP:
		status.TransportName = "ip"
		if strings.TrimSpace(dep.Config.Connection.Host) != "" {
			status.State = connectors.ConnectionStateConnecting
		}
	case config.ConnectorSerial:
		status.TransportName = "serial"
		if strings.TrimSpace(dep.Config.Connection.SerialPort) != "" {
			status.State = connectors.ConnectionStateConnecting
		}
	case config.ConnectorBluetooth:
		status.TransportName = "bluetooth"
	default:
		status.TransportName = string(dep.Config.Connection.Connector)
	}
	return status
}

func formatConnStatus(status connectors.ConnStatus) string {
	text := string(status.State)
	if status.TransportName != "" {
		text += " via " + status.TransportName
	}
	if status.Err != "" {
		text += " (" + status.Err + ")"
	}
	return text
}

func formatWindowTitle(status connectors.ConnStatus) string {
	return "MeshGo - " + formatConnStatus(status)
}

func resolveNodeDisplayName(store *domain.NodeStore) func(string) string {
	if store == nil {
		return nil
	}
	return func(nodeID string) string {
		node, ok := store.Get(nodeID)
		if !ok {
			return ""
		}
		if v := strings.TrimSpace(node.LongName); v != "" {
			return v
		}
		if v := strings.TrimSpace(node.ShortName); v != "" {
			return v
		}
		return ""
	}
}

func nodeChanges(store *domain.NodeStore) <-chan struct{} {
	if store == nil {
		return nil
	}
	return store.Changes()
}
