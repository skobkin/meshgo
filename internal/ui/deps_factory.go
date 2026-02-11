package ui

import (
	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/platform"
)

func BuildRuntimeDependencies(rt *meshapp.Runtime, launch LaunchOptions, onQuit func()) RuntimeDependencies {
	systemActions := platform.NewSystemActions()
	dep := RuntimeDependencies{
		Launch: launch,
		Actions: ActionDependencies{
			OnQuit: onQuit,
		},
		Platform: PlatformDependencies{
			OpenBluetoothSettings: systemActions.OpenBluetoothSettings,
		},
	}

	if rt == nil {
		return dep
	}

	dep.Data = DataDependencies{
		Config:            rt.Core.Config,
		ChatStore:         rt.Domain.ChatStore,
		NodeStore:         rt.Domain.NodeStore,
		Bus:               rt.Domain.Bus,
		LastSelectedChat:  rt.Core.Config.UI.LastSelectedChat,
		CurrentConnStatus: rt.CurrentConnStatus,
	}

	dep.Platform = PlatformDependencies{
		BluetoothScanner:      NewTinyGoBluetoothScanner(defaultBluetoothScanDuration),
		OpenBluetoothSettings: systemActions.OpenBluetoothSettings,
	}

	dep.Actions.OnSave = rt.SaveAndApplyConfig
	dep.Actions.OnChatSelected = rt.RememberSelectedChat
	dep.Actions.OnClearDB = rt.ClearDatabase

	if rt.Connectivity.Radio != nil {
		dep.Actions.Sender = rt.Connectivity.Radio
		dep.Data.LocalNodeID = rt.Connectivity.Radio.LocalNodeID
		if rt.Core.LogManager != nil {
			dep.Actions.NodeSettings = meshapp.NewNodeSettingsService(
				rt.Domain.Bus,
				rt.Connectivity.Radio,
				rt.CurrentConnStatus,
				rt.Core.LogManager.Logger("node-settings"),
			)
		} else {
			dep.Actions.NodeSettings = meshapp.NewNodeSettingsService(
				rt.Domain.Bus,
				rt.Connectivity.Radio,
				rt.CurrentConnStatus,
				nil,
			)
		}
	}

	return dep
}
