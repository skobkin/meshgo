package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

func newNodeMaintenancePage(dep RuntimeDependencies) fyne.CanvasObject {
	status := widget.NewLabel("Run node maintenance actions.")
	status.Wrapping = fyne.TextWrapWord
	preserveFavorites := widget.NewCheck("Preserve favorites when resetting node DB", nil)
	rebootButton := widget.NewButton("Reboot", nil)
	shutdownButton := widget.NewButton("Shutdown", nil)
	factoryResetButton := widget.NewButton("Factory reset", nil)
	resetNodeDBButton := widget.NewButton("Reset node DB", nil)

	if dep.Actions.NodeSettings == nil {
		rebootButton.Disable()
		shutdownButton.Disable()
		factoryResetButton.Disable()
		resetNodeDBButton.Disable()
		status.SetText("Node settings service is unavailable.")
	}

	runAction := func(title, message string, action func(context.Context, app.NodeSettingsTarget) error) {
		window := currentRuntimeWindow(dep)
		if window == nil {
			showErrorModal(dep, fmt.Errorf("window is unavailable"))

			return
		}
		target, ok := localNodeSettingsTarget(dep)
		if !ok {
			showErrorModal(dep, fmt.Errorf("local node is unavailable"))

			return
		}
		dialog.ShowConfirm(title, message, func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
				defer cancel()
				err := action(ctx, target)
				fyne.Do(func() {
					if err != nil {
						status.SetText(fmt.Sprintf("%s failed: %v", title, err))
						showErrorModal(dep, err)

						return
					}
					status.SetText(title + " command sent.")
				})
			}()
		}, window)
	}

	rebootButton.OnTapped = func() {
		runAction("Reboot node", "Send a reboot command to the connected node?", func(ctx context.Context, target app.NodeSettingsTarget) error {
			return dep.Actions.NodeSettings.RebootNode(ctx, target)
		})
	}
	shutdownButton.OnTapped = func() {
		runAction("Shutdown node", "Send a shutdown command to the connected node?", func(ctx context.Context, target app.NodeSettingsTarget) error {
			return dep.Actions.NodeSettings.ShutdownNode(ctx, target)
		})
	}
	factoryResetButton.OnTapped = func() {
		runAction("Factory reset node", "Factory reset will erase node configuration on the device. Continue?", func(ctx context.Context, target app.NodeSettingsTarget) error {
			return dep.Actions.NodeSettings.FactoryResetNode(ctx, target)
		})
	}
	resetNodeDBButton.OnTapped = func() {
		window := currentRuntimeWindow(dep)
		if window == nil {
			showErrorModal(dep, fmt.Errorf("window is unavailable"))

			return
		}
		target, ok := localNodeSettingsTarget(dep)
		if !ok {
			showErrorModal(dep, fmt.Errorf("local node is unavailable"))

			return
		}
		dialog.ShowConfirm("Reset node DB", "Reset the node database on the connected device?", func(ok bool) {
			if !ok {
				return
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
				defer cancel()
				err := dep.Actions.NodeSettings.ResetNodeDB(ctx, target, preserveFavorites.Checked)
				fyne.Do(func() {
					if err != nil {
						status.SetText(fmt.Sprintf("Reset node DB failed: %v", err))
						showErrorModal(dep, err)

						return
					}
					status.SetText("Reset node DB command sent.")
				})
			}()
		}, window)
	}

	return container.NewVBox(
		status,
		preserveFavorites,
		container.NewGridWithColumns(2, rebootButton, shutdownButton),
		container.NewGridWithColumns(2, factoryResetButton, resetNodeDBButton),
	)
}
