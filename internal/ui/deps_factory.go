package ui

import (
	meshapp "github.com/skobkin/meshgo/internal/app"
)

func NewDependenciesFromRuntime(rt *meshapp.Runtime, launch LaunchOptions, onQuit func()) Dependencies {
	dep := Dependencies{
		Launch: launch,
		Actions: ActionDeps{
			OnQuit: onQuit,
		},
	}

	if rt == nil {
		return dep
	}

	dep.Data = DataDeps{
		Config:            rt.Config,
		ChatStore:         rt.ChatStore,
		NodeStore:         rt.NodeStore,
		Bus:               rt.Bus,
		LastSelectedChat:  rt.Config.UI.LastSelectedChat,
		CurrentConnStatus: rt.CurrentConnStatus,
	}

	dep.Platform = PlatformDeps{
		BluetoothScanner: NewTinyGoBluetoothScanner(defaultBluetoothScanDuration),
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
