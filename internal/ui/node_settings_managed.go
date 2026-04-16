package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

var (
	nodeSettingsBroadcastShortIntervalOptions = []nodeSettingsUint32Option{
		{Label: nodeSettingsSecondsKnownLabel(30*60, ""), Value: 30 * 60},
		{Label: nodeSettingsSecondsKnownLabel(60*60, ""), Value: 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(2*60*60, ""), Value: 2 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(3*60*60, ""), Value: 3 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(4*60*60, ""), Value: 4 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(5*60*60, ""), Value: 5 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(6*60*60, ""), Value: 6 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(12*60*60, ""), Value: 12 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(18*60*60, ""), Value: 18 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(24*60*60, ""), Value: 24 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(36*60*60, ""), Value: 36 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(48*60*60, ""), Value: 48 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(72*60*60, ""), Value: 72 * 60 * 60},
	}
	nodeSettingsDetectionMinimumIntervalOptions = []nodeSettingsUint32Option{
		{Label: "Unset", Value: 0},
		{Label: nodeSettingsSecondsKnownLabel(15, ""), Value: 15},
		{Label: nodeSettingsSecondsKnownLabel(30, ""), Value: 30},
		{Label: nodeSettingsSecondsKnownLabel(60, ""), Value: 60},
		{Label: nodeSettingsSecondsKnownLabel(2*60, ""), Value: 2 * 60},
		{Label: nodeSettingsSecondsKnownLabel(5*60, ""), Value: 5 * 60},
		{Label: nodeSettingsSecondsKnownLabel(10*60, ""), Value: 10 * 60},
		{Label: nodeSettingsSecondsKnownLabel(15*60, ""), Value: 15 * 60},
		{Label: nodeSettingsSecondsKnownLabel(30*60, ""), Value: 30 * 60},
		{Label: nodeSettingsSecondsKnownLabel(60*60, ""), Value: 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(2*60*60, ""), Value: 2 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(3*60*60, ""), Value: 3 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(4*60*60, ""), Value: 4 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(5*60*60, ""), Value: 5 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(6*60*60, ""), Value: 6 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(12*60*60, ""), Value: 12 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(18*60*60, ""), Value: 18 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(24*60*60, ""), Value: 24 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(36*60*60, ""), Value: 36 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(48*60*60, ""), Value: 48 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(72*60*60, ""), Value: 72 * 60 * 60},
	}
	nodeSettingsDetectionStateIntervalOptions = []nodeSettingsUint32Option{
		{Label: "Unset", Value: 0},
		{Label: nodeSettingsSecondsKnownLabel(15*60, ""), Value: 15 * 60},
		{Label: nodeSettingsSecondsKnownLabel(30*60, ""), Value: 30 * 60},
		{Label: nodeSettingsSecondsKnownLabel(60*60, ""), Value: 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(2*60*60, ""), Value: 2 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(3*60*60, ""), Value: 3 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(4*60*60, ""), Value: 4 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(5*60*60, ""), Value: 5 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(6*60*60, ""), Value: 6 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(12*60*60, ""), Value: 12 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(18*60*60, ""), Value: 18 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(24*60*60, ""), Value: 24 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(36*60*60, ""), Value: 36 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(48*60*60, ""), Value: 48 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(72*60*60, ""), Value: 72 * 60 * 60},
	}
	nodeSettingsPaxcounterIntervalOptions = []nodeSettingsUint32Option{
		{Label: nodeSettingsSecondsKnownLabel(15*60, ""), Value: 15 * 60},
		{Label: nodeSettingsSecondsKnownLabel(30*60, ""), Value: 30 * 60},
		{Label: nodeSettingsSecondsKnownLabel(60*60, ""), Value: 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(2*60*60, ""), Value: 2 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(3*60*60, ""), Value: 3 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(4*60*60, ""), Value: 4 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(5*60*60, ""), Value: 5 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(6*60*60, ""), Value: 6 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(12*60*60, ""), Value: 12 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(18*60*60, ""), Value: 18 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(24*60*60, ""), Value: 24 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(36*60*60, ""), Value: 36 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(48*60*60, ""), Value: 48 * 60 * 60},
		{Label: nodeSettingsSecondsKnownLabel(72*60*60, ""), Value: 72 * 60 * 60},
	}
	nodeSettingsNagTimeoutOptions = []nodeSettingsUint32Option{
		{Label: "Unset", Value: 0},
		{Label: nodeSettingsSecondsKnownLabel(1, ""), Value: 1},
		{Label: nodeSettingsSecondsKnownLabel(5, ""), Value: 5},
		{Label: nodeSettingsSecondsKnownLabel(10, ""), Value: 10},
		{Label: nodeSettingsSecondsKnownLabel(15, ""), Value: 15},
		{Label: nodeSettingsSecondsKnownLabel(30, ""), Value: 30},
		{Label: nodeSettingsSecondsKnownLabel(60, ""), Value: 60},
	}
	nodeSettingsOutputDurationOptions = []nodeSettingsUint32Option{
		{Label: "Unset", Value: 0},
		{Label: "1 ms", Value: 1},
		{Label: "2 ms", Value: 2},
		{Label: "3 ms", Value: 3},
		{Label: "4 ms", Value: 4},
		{Label: "5 ms", Value: 5},
		{Label: "10 ms", Value: 10},
	}
	nodeSettingsSerialBaudOptions = []nodeSettingsInt32Option{
		{Label: "Default", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_DEFAULT)},
		{Label: "110", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_110)},
		{Label: "300", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_300)},
		{Label: "600", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_600)},
		{Label: "1200", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_1200)},
		{Label: "2400", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_2400)},
		{Label: "4800", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_4800)},
		{Label: "9600", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_9600)},
		{Label: "19200", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_19200)},
		{Label: "38400", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_38400)},
		{Label: "57600", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_57600)},
		{Label: "115200", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_115200)},
		{Label: "230400", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_230400)},
		{Label: "460800", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_460800)},
		{Label: "576000", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_576000)},
		{Label: "921600", Value: int32(generated.ModuleConfig_SerialConfig_BAUD_921600)},
	}
	nodeSettingsSerialModeOptions = []nodeSettingsInt32Option{
		{Label: "Default", Value: int32(generated.ModuleConfig_SerialConfig_DEFAULT)},
		{Label: "Simple", Value: int32(generated.ModuleConfig_SerialConfig_SIMPLE)},
		{Label: "Proto", Value: int32(generated.ModuleConfig_SerialConfig_PROTO)},
		{Label: "Text message", Value: int32(generated.ModuleConfig_SerialConfig_TEXTMSG)},
		{Label: "NMEA", Value: int32(generated.ModuleConfig_SerialConfig_NMEA)},
		{Label: "CalTopo", Value: int32(generated.ModuleConfig_SerialConfig_CALTOPO)},
		{Label: "WS85", Value: int32(generated.ModuleConfig_SerialConfig_WS85)},
		{Label: "VE Direct", Value: int32(generated.ModuleConfig_SerialConfig_VE_DIRECT)},
		{Label: "MS Config", Value: int32(generated.ModuleConfig_SerialConfig_MS_CONFIG)},
		{Label: "Log", Value: int32(generated.ModuleConfig_SerialConfig_LOG)},
		{Label: "Log text", Value: int32(generated.ModuleConfig_SerialConfig_LOGTEXT)},
	}
	nodeSettingsAudioBitrateOptions = []nodeSettingsInt32Option{
		{Label: "Default", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_DEFAULT)},
		{Label: "3200", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_3200)},
		{Label: "2400", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_2400)},
		{Label: "1600", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_1600)},
		{Label: "1400", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_1400)},
		{Label: "1300", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_1300)},
		{Label: "1200", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_1200)},
		{Label: "700", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_700)},
		{Label: "700B", Value: int32(generated.ModuleConfig_AudioConfig_CODEC2_700B)},
	}
	nodeSettingsDetectionTriggerTypeOptions = []nodeSettingsInt32Option{
		{Label: "Logic low", Value: int32(generated.ModuleConfig_DetectionSensorConfig_LOGIC_LOW)},
		{Label: "Logic high", Value: int32(generated.ModuleConfig_DetectionSensorConfig_LOGIC_HIGH)},
		{Label: "Falling edge", Value: int32(generated.ModuleConfig_DetectionSensorConfig_FALLING_EDGE)},
		{Label: "Rising edge", Value: int32(generated.ModuleConfig_DetectionSensorConfig_RISING_EDGE)},
		{Label: "Either edge active low", Value: int32(generated.ModuleConfig_DetectionSensorConfig_EITHER_EDGE_ACTIVE_LOW)},
		{Label: "Either edge active high", Value: int32(generated.ModuleConfig_DetectionSensorConfig_EITHER_EDGE_ACTIVE_HIGH)},
	}
)

type nodeManagedSettingsForm[T any] struct {
	content   fyne.CanvasObject
	set       func(T)
	read      func(base T, target app.NodeSettingsTarget) (T, error)
	setSaving func(bool)
}

func newManagedNodeSettingsPage[T any](
	dep RuntimeDependencies,
	saveGate *nodeSettingsSaveGate,
	pageID string,
	loadingStatus string,
	loadedStatus string,
	load func(context.Context, app.NodeSettingsTarget) (T, error),
	save func(context.Context, app.NodeSettingsTarget, T) error,
	clone func(T) T,
	fingerprint func(T) string,
	buildForm func(onChanged func()) nodeManagedSettingsForm[T],
) (fyne.CanvasObject, func()) {
	nodeSettingsTabLogger.Debug("building managed node settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls(loadingStatus)
	form := buildForm(func() {})

	var (
		baseline             T
		baselineFingerprint  string
		dirty                bool
		saving               bool
		initialReloadStarted atomic.Bool
		applyingForm         atomic.Bool
		mu                   sync.Mutex
	)

	updateButtons := func() {
		mu.Lock()
		activePage := ""
		if saveGate != nil {
			activePage = strings.TrimSpace(saveGate.ActivePage())
		}
		connected := isNodeSettingsConnected(dep)
		canSave := dep.Actions.NodeSettings != nil && connected && !saving && dirty && (activePage == "" || activePage == pageID)
		canCancel := !saving && dirty
		canReload := dep.Actions.NodeSettings != nil && !saving
		isSaving := saving
		mu.Unlock()

		if canSave {
			controls.saveButton.Enable()
		} else {
			controls.saveButton.Disable()
		}
		if canCancel {
			controls.cancelButton.Enable()
		} else {
			controls.cancelButton.Disable()
		}
		if canReload {
			controls.reloadButton.Enable()
		} else {
			controls.reloadButton.Disable()
		}
		if form.setSaving != nil {
			form.setSaving(isSaving)
		}
	}

	markDirty := func() {
		if applyingForm.Load() {
			return
		}

		target, ok := localNodeSettingsTarget(dep)
		if !ok {
			updateButtons()

			return
		}
		mu.Lock()
		base := clone(baseline)
		mu.Unlock()
		next, err := form.read(base, target)
		if err != nil {
			updateButtons()

			return
		}
		mu.Lock()
		dirty = fingerprint(next) != baselineFingerprint
		mu.Unlock()
		updateButtons()
	}

	form = buildForm(markDirty)

	applyLoadedSettings := func(next T) {
		mu.Lock()
		baseline = clone(next)
		baselineFingerprint = fingerprint(next)
		dirty = false
		current := clone(baseline)
		mu.Unlock()

		applyingForm.Store(true)
		form.set(current)
		applyingForm.Store(false)
		controls.SetStatus(loadedStatus, 1, 1)
		updateButtons()
	}

	reloadFromDevice := func() {
		target, ok := localNodeSettingsTarget(dep)
		if !ok {
			controls.SetStatus("Local node is unavailable.", 0, 1)
			updateButtons()

			return
		}
		if dep.Actions.NodeSettings == nil {
			controls.SetStatus("Node settings service is unavailable.", 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		dirty = false
		mu.Unlock()
		updateButtons()
		controls.SetStatus(loadingStatus, 0, 1)

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()

			loaded, err := load(ctx, target)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				mu.Unlock()
				if err != nil {
					controls.SetStatus(fmt.Sprintf("Load failed: %v", err), 0, 1)
					updateButtons()
					showErrorModal(dep, err)

					return
				}
				applyLoadedSettings(loaded)
			})
		}()
	}

	controls.cancelButton.OnTapped = func() {
		mu.Lock()
		current := clone(baseline)
		dirty = false
		mu.Unlock()
		applyingForm.Store(true)
		form.set(current)
		applyingForm.Store(false)
		controls.SetStatus(loadedStatus, 1, 1)
		updateButtons()
	}

	controls.reloadButton.OnTapped = reloadFromDevice

	controls.saveButton.OnTapped = func() {
		target, ok := localNodeSettingsTarget(dep)
		if !ok {
			showErrorModal(dep, fmt.Errorf("local node is unavailable"))

			return
		}
		if dep.Actions.NodeSettings == nil {
			showErrorModal(dep, fmt.Errorf("node settings service is unavailable"))

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			showErrorModal(dep, fmt.Errorf("another settings page save is already running"))

			return
		}

		mu.Lock()
		base := clone(baseline)
		mu.Unlock()

		next, err := form.read(base, target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			showErrorModal(dep, err)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		updateButtons()
		controls.SetStatus("Saving settings…", 0, 1)

		go func(payload T) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()

			err := save(ctx, target, payload)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				mu.Unlock()
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					controls.SetStatus(fmt.Sprintf("Save failed: %v", err), 0, 1)
					updateButtons()
					showErrorModal(dep, err)

					return
				}
				applyLoadedSettings(payload)
				controls.SetStatus("Settings saved.", 1, 1)
			})
		}(next)
	}

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			controls.SetStatus("Node settings service is unavailable.", 0, 1)
			updateButtons()

			return
		}
		if initialReloadStarted.CompareAndSwap(false, true) {
			reloadFromDevice()

			return
		}
		updateButtons()
	}

	updateButtons()

	return wrapNodeSettingsPage(form.content, controls), onTabOpened
}

func disableWidgets(objects ...fyne.Disableable) func(bool) {
	return func(disabled bool) {
		for _, object := range objects {
			if object == nil {
				continue
			}
			if disabled {
				object.Disable()
			} else {
				object.Enable()
			}
		}
	}
}

func newNumberEntry(onChanged func()) *widget.Entry {
	entry := widget.NewEntry()
	entry.OnChanged = func(string) { onChanged() }

	return entry
}

func newSettingsCheck(onChanged func()) *widget.Check {
	return widget.NewCheck("", func(bool) { onChanged() })
}

func parseOptionalUint32(raw string) (uint32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(parsed), nil
}

func parseOptionalInt32(raw string) (int32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(parsed), nil
}

func parseUint32List(raw string) ([]uint32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]uint32, 0, len(parts))
	for _, part := range parts {
		value, err := parseOptionalUint32(part)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}

	return out, nil
}

func nodeSettingsCustomMillisecondsLabel(milliseconds uint32) string {
	return fmt.Sprintf("Custom (%d ms)", milliseconds)
}

func nodeSettingsParseMillisecondsSelectLabel(
	fieldName string,
	selected string,
	options []nodeSettingsUint32Option,
) (uint32, error) {
	selected = strings.TrimSpace(selected)
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}
	if strings.HasPrefix(selected, nodeSettingsCustomLabelPrefix) && strings.HasSuffix(selected, " ms)") {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, nodeSettingsCustomLabelPrefix), " ms)")
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return uint32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}

func fieldParseError(field string, err error) error {
	return fmt.Errorf("parse %s: %w", field, err)
}

func cloneNodeRemoteHardwareSettings(in app.NodeRemoteHardwareSettings) app.NodeRemoteHardwareSettings {
	out := in
	if len(in.AvailablePins) > 0 {
		out.AvailablePins = append([]uint32(nil), in.AvailablePins...)
	}

	return out
}
