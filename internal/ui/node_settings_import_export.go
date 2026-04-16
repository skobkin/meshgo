package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const nodeSettingsProfileFileExt = ".cfg"

func newNodeImportExportPage(dep RuntimeDependencies) fyne.CanvasObject {
	status := widget.NewLabel("Import and export node settings using Android-compatible Meshtastic profile files.")
	status.Wrapping = fyne.TextWrapWord
	exportButton := widget.NewButton("Export profile…", nil)
	importButton := widget.NewButton("Import profile…", nil)

	if dep.Actions.NodeSettings == nil {
		exportButton.Disable()
		importButton.Disable()
		status.SetText("Node settings service is unavailable.")
	}

	exportButton.OnTapped = func() {
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
		saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				showErrorModal(dep, err)

				return
			}
			if writer == nil {
				return
			}
			go func() {
				defer func() {
					_ = writer.Close()
				}()
				ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
				defer cancel()

				profile, exportErr := dep.Actions.NodeSettings.ExportProfile(ctx, target)
				if exportErr != nil {
					fyne.Do(func() {
						status.SetText(fmt.Sprintf("Export failed: %v", exportErr))
						showErrorModal(dep, exportErr)
					})

					return
				}
				raw, exportErr := app.EncodeDeviceProfile(profile)
				if exportErr != nil {
					fyne.Do(func() {
						status.SetText(fmt.Sprintf("Export failed: %v", exportErr))
						showErrorModal(dep, exportErr)
					})

					return
				}
				if _, exportErr = writer.Write(raw); exportErr != nil {
					fyne.Do(func() {
						status.SetText(fmt.Sprintf("Export failed: %v", exportErr))
						showErrorModal(dep, exportErr)
					})

					return
				}
				fyne.Do(func() {
					status.SetText(fmt.Sprintf("Exported profile to %s.", writer.URI().Name()))
				})
			}()
		}, window)
		saveDialog.SetFileName(defaultNodeSettingsProfileFilename(dep))
		saveDialog.SetFilter(storage.NewExtensionFileFilter([]string{nodeSettingsProfileFileExt}))
		saveDialog.Show()
	}

	importButton.OnTapped = func() {
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
		openDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				showErrorModal(dep, err)

				return
			}
			if reader == nil {
				return
			}
			go func() {
				defer func() {
					_ = reader.Close()
				}()
				raw, readErr := io.ReadAll(reader)
				if readErr != nil {
					fyne.Do(func() { showErrorModal(dep, readErr) })

					return
				}
				profile, decodeErr := app.DecodeDeviceProfile(raw)
				if decodeErr != nil {
					fyne.Do(func() { showErrorModal(dep, decodeErr) })

					return
				}
				summary := buildDeviceProfileSummary(profile)
				fyne.Do(func() {
					dialog.ShowConfirm(
						"Import node settings profile",
						summary,
						func(ok bool) {
							if !ok {
								return
							}
							go func() {
								ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
								defer cancel()
								importErr := dep.Actions.NodeSettings.ImportProfile(ctx, target, profile)
								fyne.Do(func() {
									if importErr != nil {
										status.SetText(fmt.Sprintf("Import failed: %v", importErr))
										showErrorModal(dep, importErr)

										return
									}
									status.SetText(fmt.Sprintf("Imported profile from %s.", reader.URI().Name()))
								})
							}()
						},
						window,
					)
				})
			}()
		}, window)
		openDialog.SetFilter(storage.NewExtensionFileFilter([]string{nodeSettingsProfileFileExt}))
		openDialog.Show()
	}

	return container.NewVBox(
		widget.NewLabel("Node settings profile"),
		status,
		container.NewHBox(exportButton, importButton),
	)
}

func defaultNodeSettingsProfileFilename(dep RuntimeDependencies) string {
	nodeName := "node"
	if snapshot := localNodeSnapshot(dep); snapshot.Present {
		if name := strings.TrimSpace(snapshot.Node.LongName); name != "" {
			nodeName = sanitizeProfileFilenamePart(name)
		} else if name := strings.TrimSpace(snapshot.Node.ShortName); name != "" {
			nodeName = sanitizeProfileFilenamePart(name)
		}
	}

	return fmt.Sprintf("Meshtastic_%s_%s_nodeConfig%s", nodeName, time.Now().Format("2006-01-02"), nodeSettingsProfileFileExt)
}

func sanitizeProfileFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "node"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	result := strings.Trim(b.String(), "_")
	if result == "" {
		return "node"
	}

	return result
}

func buildDeviceProfileSummary(profile *generated.DeviceProfile) string {
	if profile == nil {
		return "The selected file does not contain a Meshtastic device profile."
	}
	configCount := 0
	if cfg := profile.GetConfig(); cfg != nil {
		if cfg.GetDevice() != nil {
			configCount++
		}
		if cfg.GetPosition() != nil {
			configCount++
		}
		if cfg.GetPower() != nil {
			configCount++
		}
		if cfg.GetNetwork() != nil {
			configCount++
		}
		if cfg.GetDisplay() != nil {
			configCount++
		}
		if cfg.GetLora() != nil {
			configCount++
		}
		if cfg.GetBluetooth() != nil {
			configCount++
		}
		if cfg.GetSecurity() != nil {
			configCount++
		}
	}
	moduleCount := 0
	if cfg := profile.GetModuleConfig(); cfg != nil {
		if cfg.GetMqtt() != nil {
			moduleCount++
		}
		if cfg.GetSerial() != nil {
			moduleCount++
		}
		if cfg.GetExternalNotification() != nil {
			moduleCount++
		}
		if cfg.GetStoreForward() != nil {
			moduleCount++
		}
		if cfg.GetRangeTest() != nil {
			moduleCount++
		}
		if cfg.GetTelemetry() != nil {
			moduleCount++
		}
		if cfg.GetCannedMessage() != nil {
			moduleCount++
		}
		if cfg.GetAudio() != nil {
			moduleCount++
		}
		if cfg.GetRemoteHardware() != nil {
			moduleCount++
		}
		if cfg.GetNeighborInfo() != nil {
			moduleCount++
		}
		if cfg.GetAmbientLighting() != nil {
			moduleCount++
		}
		if cfg.GetDetectionSensor() != nil {
			moduleCount++
		}
		if cfg.GetPaxcounter() != nil {
			moduleCount++
		}
		if cfg.GetStatusmessage() != nil {
			moduleCount++
		}
	}

	return fmt.Sprintf(
		"Import profile for \"%s\" / \"%s\"?\n\nConfig sections: %d\nModule sections: %d\nFixed position: %t\nRingtone: %t\nCanned messages: %t",
		orUnknown(profile.GetLongName()),
		orUnknown(profile.GetShortName()),
		configCount,
		moduleCount,
		profile.GetFixedPosition() != nil,
		profile.Ringtone != nil,
		profile.CannedMessages != nil,
	)
}
