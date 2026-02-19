package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func currentRuntimeWindow(dep RuntimeDependencies) fyne.Window {
	if dep.UIHooks.CurrentWindow != nil {
		return dep.UIHooks.CurrentWindow()
	}

	app := fyne.CurrentApp()
	if app == nil {
		return nil
	}

	driver := app.Driver()
	if driver == nil {
		return nil
	}
	windows := driver.AllWindows()
	if len(windows) == 0 {
		return nil
	}

	return windows[0]
}

func showErrorModal(dep RuntimeDependencies, err error) {
	if err == nil {
		return
	}

	window := currentRuntimeWindow(dep)
	if dep.UIHooks.ShowErrorDialog != nil {
		dep.UIHooks.ShowErrorDialog(err, window)

		return
	}
	if window != nil {
		dialog.ShowError(err, window)
	}
}
