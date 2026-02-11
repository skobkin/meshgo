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
		Config:            rt.Config,
		ChatStore:         rt.ChatStore,
		NodeStore:         rt.NodeStore,
		Bus:               rt.Bus,
		LastSelectedChat:  rt.Config.UI.LastSelectedChat,
		CurrentConnStatus: rt.CurrentConnStatus,
	}

	dep.Platform = PlatformDependencies{
		BluetoothScanner:      NewTinyGoBluetoothScanner(defaultBluetoothScanDuration),
		OpenBluetoothSettings: systemActions.OpenBluetoothSettings,
	}

	dep.Actions.OnSave = rt.SaveAndApplyConfig
	dep.Actions.OnChatSelected = rt.RememberSelectedChat
	dep.Actions.OnClearDB = rt.ClearDatabase

	if rt.Radio != nil {
		dep.Actions.Sender = rt.Radio
		dep.Data.LocalNodeID = rt.Radio.LocalNodeID
	}

	return dep
}
