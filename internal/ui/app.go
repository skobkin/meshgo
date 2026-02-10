package ui

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

func Run(dep Dependencies) error {
	fyApp := app.NewWithID("meshgo")
	icon := resources.TrayIconResource()
	fyApp.SetIcon(icon)

	window := fyApp.NewWindow("meshgo")
	window.Resize(fyne.NewSize(1000, 700))

	initialStatus := initialConnStatus(dep)
	sidebarConnStatus := widget.NewLabel(formatConnStatus(initialStatus))
	settingsConnStatus := widget.NewLabel(formatConnStatus(initialStatus))

	chatsTab := newChatsTab(dep.ChatStore, dep.Sender, resolveNodeDisplayName(dep.NodeStore))
	nodesTab := newNodesTab(dep.NodeStore, DefaultNodeRowRenderer())
	mapTab := disabledTab("Map is not implemented yet")
	nodeSettingsTab := disabledTab("Node Settings is not implemented yet")
	settingsTab := newSettingsTab(dep, settingsConnStatus)

	tabContent := map[string]fyne.CanvasObject{
		"Chats": chatsTab,
		"Nodes": nodesTab,
		"Map":   mapTab,
		"Node":  nodeSettingsTab,
		"App":   settingsTab,
	}
	order := []string{"Chats", "Nodes", "Map", "Node", "App"}

	rightStack := container.NewMax()
	for _, key := range order {
		rightStack.Add(tabContent[key])
		tabContent[key].Hide()
	}
	active := "Chats"
	tabContent[active].Show()

	switchTab := func(name string) {
		tabContent[active].Hide()
		active = name
		tabContent[active].Show()
		rightStack.Refresh()
	}

	left := container.NewVBox(widget.NewLabel("Menu"))
	for _, name := range order {
		nameCopy := name
		button := widget.NewButton(nameCopy, func() {
			switchTab(nameCopy)
		})
		if name == "Map" || name == "Node Settings" {
			button.Disable()
		}
		left.Add(button)
	}
	left.Add(layout.NewSpacer())
	left.Add(sidebarConnStatus)

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
					sidebarConnStatus.SetText(text)
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
		desk.SetSystemTrayIcon(icon)
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
	if dep.IPTransport == nil {
		return status
	}

	status.TransportName = dep.IPTransport.Name()
	if dep.IPTransport.Connected() {
		status.State = connectors.ConnectionStateConnected
		return status
	}
	if dep.IPTransport.Host() != "" {
		status.State = connectors.ConnectionStateConnecting
	}
	return status
}

func formatConnStatus(status connectors.ConnStatus) string {
	text := fmt.Sprintf("Connection: %s", status.State)
	if status.TransportName != "" {
		text += " via " + status.TransportName
	}
	if status.Err != "" {
		text += " (" + status.Err + ")"
	}
	return text
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
