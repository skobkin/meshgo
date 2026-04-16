package ui

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

func newNodeNetworkSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep,
		saveGate,
		"device.network",
		"Loading network settings…",
		"Network settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
			return dep.Actions.NodeSettings.LoadNetworkSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNetworkSettings) error {
			return dep.Actions.NodeSettings.SaveNetworkSettings(ctx, target, settings)
		},
		func(v app.NodeNetworkSettings) app.NodeNetworkSettings { return v },
		func(v app.NodeNetworkSettings) string {
			return fmt.Sprintf("%s|%t|%s|%s|%s|%t|%d|%d|%d|%d|%d|%s|%d|%t",
				v.NodeID, v.WifiEnabled, v.WifiSSID, v.WifiPSK, v.NTPServer, v.EthernetEnabled, v.AddressMode,
				v.IPv4Address, v.IPv4Gateway, v.IPv4Subnet, v.IPv4DNS, v.RsyslogServer, v.EnabledProtocols, v.IPv6Enabled)
		},
		buildNodeNetworkSettingsForm,
	)
}

func buildNodeNetworkSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeNetworkSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	wifiEnabled := newSettingsCheck(onChanged)
	wifiSSID := widget.NewEntry()
	wifiSSID.OnChanged = func(string) { onChanged() }
	wifiPSK := widget.NewPasswordEntry()
	wifiPSK.OnChanged = func(string) { onChanged() }
	ntpServer := widget.NewEntry()
	ntpServer.OnChanged = func(string) { onChanged() }
	ethernetEnabled := newSettingsCheck(onChanged)
	addressMode := widget.NewEntry()
	addressMode.OnChanged = func(string) { onChanged() }
	ipv4Address := widget.NewEntry()
	ipv4Address.OnChanged = func(string) { onChanged() }
	ipv4Gateway := widget.NewEntry()
	ipv4Gateway.OnChanged = func(string) { onChanged() }
	ipv4Subnet := widget.NewEntry()
	ipv4Subnet.OnChanged = func(string) { onChanged() }
	ipv4DNS := widget.NewEntry()
	ipv4DNS.OnChanged = func(string) { onChanged() }
	rsyslogServer := widget.NewEntry()
	rsyslogServer.OnChanged = func(string) { onChanged() }
	enabledProtocols := widget.NewEntry()
	enabledProtocols.OnChanged = func(string) { onChanged() }
	ipv6Enabled := newSettingsCheck(onChanged)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("WiFi enabled", wifiEnabled),
		widget.NewFormItem("WiFi SSID", wifiSSID),
		widget.NewFormItem("WiFi password", wifiPSK),
		widget.NewFormItem("NTP server", ntpServer),
		widget.NewFormItem("Ethernet enabled", ethernetEnabled),
		widget.NewFormItem("Address mode", addressMode),
		widget.NewFormItem("IPv4 address", ipv4Address),
		widget.NewFormItem("IPv4 gateway", ipv4Gateway),
		widget.NewFormItem("IPv4 subnet", ipv4Subnet),
		widget.NewFormItem("IPv4 DNS", ipv4DNS),
		widget.NewFormItem("Rsyslog server", rsyslogServer),
		widget.NewFormItem("Enabled protocols", enabledProtocols),
		widget.NewFormItem("IPv6 enabled", ipv6Enabled),
	)

	return nodeManagedSettingsForm[app.NodeNetworkSettings]{
		content: form,
		set: func(v app.NodeNetworkSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			wifiEnabled.SetChecked(v.WifiEnabled)
			wifiSSID.SetText(v.WifiSSID)
			wifiPSK.SetText(v.WifiPSK)
			ntpServer.SetText(v.NTPServer)
			ethernetEnabled.SetChecked(v.EthernetEnabled)
			addressMode.SetText(strconv.FormatInt(int64(v.AddressMode), 10))
			ipv4Address.SetText(strconv.FormatUint(uint64(v.IPv4Address), 10))
			ipv4Gateway.SetText(strconv.FormatUint(uint64(v.IPv4Gateway), 10))
			ipv4Subnet.SetText(strconv.FormatUint(uint64(v.IPv4Subnet), 10))
			ipv4DNS.SetText(strconv.FormatUint(uint64(v.IPv4DNS), 10))
			rsyslogServer.SetText(v.RsyslogServer)
			enabledProtocols.SetText(strconv.FormatUint(uint64(v.EnabledProtocols), 10))
			ipv6Enabled.SetChecked(v.IPv6Enabled)
		},
		read: func(base app.NodeNetworkSettings, target app.NodeSettingsTarget) (app.NodeNetworkSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.WifiEnabled = wifiEnabled.Checked
			base.WifiSSID = strings.TrimSpace(wifiSSID.Text)
			base.WifiPSK = wifiPSK.Text
			base.NTPServer = strings.TrimSpace(ntpServer.Text)
			base.EthernetEnabled = ethernetEnabled.Checked
			value, err := parseOptionalInt32(addressMode.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("address mode", err)
			}
			base.AddressMode = value
			base.IPv4Address, err = parseOptionalUint32(ipv4Address.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 address", err)
			}
			base.IPv4Gateway, err = parseOptionalUint32(ipv4Gateway.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 gateway", err)
			}
			base.IPv4Subnet, err = parseOptionalUint32(ipv4Subnet.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 subnet", err)
			}
			base.IPv4DNS, err = parseOptionalUint32(ipv4DNS.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("IPv4 DNS", err)
			}
			base.RsyslogServer = strings.TrimSpace(rsyslogServer.Text)
			base.EnabledProtocols, err = parseOptionalUint32(enabledProtocols.Text)
			if err != nil {
				return app.NodeNetworkSettings{}, fieldParseError("enabled protocols", err)
			}
			base.IPv6Enabled = ipv6Enabled.Checked

			return base, nil
		},
		setSaving: disableWidgets(
			wifiEnabled, wifiSSID, wifiPSK, ntpServer, ethernetEnabled, addressMode, ipv4Address,
			ipv4Gateway, ipv4Subnet, ipv4DNS, rsyslogServer, enabledProtocols, ipv6Enabled,
		),
	}
}

func newNodeSerialSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.serial", "Loading serial settings…", "Serial settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeSerialSettings, error) {
			return dep.Actions.NodeSettings.LoadSerialSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeSerialSettings) error {
			return dep.Actions.NodeSettings.SaveSerialSettings(ctx, target, settings)
		},
		func(v app.NodeSerialSettings) app.NodeSerialSettings { return v },
		func(v app.NodeSerialSettings) string {
			return fmt.Sprintf("%s|%t|%t|%d|%d|%d|%d|%d|%t", v.NodeID, v.Enabled, v.EchoEnabled, v.RXGPIO, v.TXGPIO, v.Baud, v.Timeout, v.Mode, v.OverrideConsoleSerialPort)
		},
		buildNodeSerialSettingsForm,
	)
}

func buildNodeSerialSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeSerialSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	echoEnabled := newSettingsCheck(onChanged)
	rxGPIO := newNumberEntry(onChanged)
	txGPIO := newNumberEntry(onChanged)
	baud := widget.NewSelect(nil, nil)
	baud.OnChanged = func(string) { onChanged() }
	timeout := newNumberEntry(onChanged)
	mode := widget.NewSelect(nil, nil)
	mode.OnChanged = func(string) { onChanged() }
	overrideConsole := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Echo enabled", echoEnabled),
		widget.NewFormItem("RX GPIO", rxGPIO),
		widget.NewFormItem("TX GPIO", txGPIO),
		widget.NewFormItem("Baud", baud),
		widget.NewFormItem("Timeout", timeout),
		widget.NewFormItem("Mode", mode),
		widget.NewFormItem("Override console serial port", overrideConsole),
	)

	return nodeManagedSettingsForm[app.NodeSerialSettings]{
		content: form,
		set: func(v app.NodeSerialSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			echoEnabled.SetChecked(v.EchoEnabled)
			rxGPIO.SetText(strconv.FormatUint(uint64(v.RXGPIO), 10))
			txGPIO.SetText(strconv.FormatUint(uint64(v.TXGPIO), 10))
			nodeSettingsSetInt32Select(baud, nodeSettingsSerialBaudOptions, v.Baud, nodeSettingsCustomInt32Label)
			timeout.SetText(strconv.FormatUint(uint64(v.Timeout), 10))
			nodeSettingsSetInt32Select(mode, nodeSettingsSerialModeOptions, v.Mode, nodeSettingsCustomInt32Label)
			overrideConsole.SetChecked(v.OverrideConsoleSerialPort)
		},
		read: func(base app.NodeSerialSettings, target app.NodeSettingsTarget) (app.NodeSerialSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.EchoEnabled = echoEnabled.Checked
			base.RXGPIO, err = parseOptionalUint32(rxGPIO.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("RX GPIO", err)
			}
			base.TXGPIO, err = parseOptionalUint32(txGPIO.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("TX GPIO", err)
			}
			base.Baud, err = nodeSettingsParseInt32SelectLabel("baud", baud.Selected, nodeSettingsSerialBaudOptions)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("baud", err)
			}
			base.Timeout, err = parseOptionalUint32(timeout.Text)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("timeout", err)
			}
			base.Mode, err = nodeSettingsParseInt32SelectLabel("mode", mode.Selected, nodeSettingsSerialModeOptions)
			if err != nil {
				return app.NodeSerialSettings{}, fieldParseError("mode", err)
			}
			base.OverrideConsoleSerialPort = overrideConsole.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, echoEnabled, rxGPIO, txGPIO, baud, timeout, mode, overrideConsole),
	}
}

func newNodeExternalNotificationSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.external_notification", "Loading external notification settings…", "External notification settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error) {
			return dep.Actions.NodeSettings.LoadExternalNotificationSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeExternalNotificationSettings) error {
			return dep.Actions.NodeSettings.SaveExternalNotificationSettings(ctx, target, settings)
		},
		func(v app.NodeExternalNotificationSettings) app.NodeExternalNotificationSettings { return v },
		func(v app.NodeExternalNotificationSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%t|%t|%t|%t|%t|%t|%t|%t|%d|%s|%t",
				v.NodeID, v.Enabled, v.OutputMS, v.OutputGPIO, v.OutputVibraGPIO, v.OutputBuzzerGPIO, v.OutputActiveHigh,
				v.AlertMessageLED, v.AlertMessageVibra, v.AlertMessageBuzzer, v.AlertBellLED, v.AlertBellVibra, v.AlertBellBuzzer,
				v.UsePWMBuzzer, v.NagTimeoutSecs, v.Ringtone, v.UseI2SAsBuzzer)
		},
		buildNodeExternalNotificationSettingsForm,
	)
}

func buildNodeExternalNotificationSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeExternalNotificationSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	outputMS := widget.NewSelect(nil, nil)
	outputMS.OnChanged = func(string) { onChanged() }
	outputGPIO := newNumberEntry(onChanged)
	outputVibraGPIO := newNumberEntry(onChanged)
	outputBuzzerGPIO := newNumberEntry(onChanged)
	outputActiveHigh := newSettingsCheck(onChanged)
	alertMessageLED := newSettingsCheck(onChanged)
	alertMessageVibra := newSettingsCheck(onChanged)
	alertMessageBuzzer := newSettingsCheck(onChanged)
	alertBellLED := newSettingsCheck(onChanged)
	alertBellVibra := newSettingsCheck(onChanged)
	alertBellBuzzer := newSettingsCheck(onChanged)
	usePWMBuzzer := newSettingsCheck(onChanged)
	nagTimeout := widget.NewSelect(nil, nil)
	nagTimeout.OnChanged = func(string) { onChanged() }
	ringtone := widget.NewMultiLineEntry()
	ringtone.SetMinRowsVisible(4)
	ringtone.OnChanged = func(string) { onChanged() }
	useI2S := newSettingsCheck(onChanged)

	configForm := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("External notification enabled", enabled),
	)
	messageForm := widget.NewForm(
		widget.NewFormItem("Alert message LED", alertMessageLED),
		widget.NewFormItem("Alert message buzzer", alertMessageBuzzer),
		widget.NewFormItem("Alert message vibra", alertMessageVibra),
	)
	bellForm := widget.NewForm(
		widget.NewFormItem("Alert bell LED", alertBellLED),
		widget.NewFormItem("Alert bell buzzer", alertBellBuzzer),
		widget.NewFormItem("Alert bell vibra", alertBellVibra),
	)
	activeHighItem := widget.NewForm(
		widget.NewFormItem("Output LED active high", outputActiveHigh),
	)
	pwmItem := widget.NewForm(
		widget.NewFormItem("Use PWM buzzer", usePWMBuzzer),
	)
	advancedForm := widget.NewForm(
		widget.NewFormItem("Output LED GPIO", outputGPIO),
		widget.NewFormItem("Output buzzer GPIO", outputBuzzerGPIO),
		widget.NewFormItem("Output vibra GPIO", outputVibraGPIO),
		widget.NewFormItem("Output duration milliseconds", outputMS),
		widget.NewFormItem("Nag timeout seconds", nagTimeout),
		widget.NewFormItem("Ringtone", ringtone),
		widget.NewFormItem("Use I2S as buzzer", useI2S),
	)
	updateAdvancedVisibility := func() {
		if strings.TrimSpace(outputGPIO.Text) != "" && strings.TrimSpace(outputGPIO.Text) != "0" {
			activeHighItem.Show()
		} else {
			activeHighItem.Hide()
		}
		if strings.TrimSpace(outputBuzzerGPIO.Text) != "" && strings.TrimSpace(outputBuzzerGPIO.Text) != "0" {
			pwmItem.Show()
		} else {
			pwmItem.Hide()
		}
	}
	outputGPIO.OnChanged = func(string) {
		updateAdvancedVisibility()
		onChanged()
	}
	outputBuzzerGPIO.OnChanged = func(string) {
		updateAdvancedVisibility()
		onChanged()
	}
	content := container.NewVBox(
		widget.NewCard("External notification config", "", configForm),
		widget.NewCard("Notifications on message receipt", "", messageForm),
		widget.NewCard("Notifications on alert bell receipt", "", bellForm),
		widget.NewCard("Advanced", "", container.NewVBox(
			advancedForm,
			activeHighItem,
			pwmItem,
		)),
	)
	updateAdvancedVisibility()

	return nodeManagedSettingsForm[app.NodeExternalNotificationSettings]{
		content: content,
		set: func(v app.NodeExternalNotificationSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(outputMS, nodeSettingsOutputDurationOptions, v.OutputMS, nodeSettingsCustomMillisecondsLabel)
			outputGPIO.SetText(strconv.FormatUint(uint64(v.OutputGPIO), 10))
			outputVibraGPIO.SetText(strconv.FormatUint(uint64(v.OutputVibraGPIO), 10))
			outputBuzzerGPIO.SetText(strconv.FormatUint(uint64(v.OutputBuzzerGPIO), 10))
			outputActiveHigh.SetChecked(v.OutputActiveHigh)
			alertMessageLED.SetChecked(v.AlertMessageLED)
			alertMessageVibra.SetChecked(v.AlertMessageVibra)
			alertMessageBuzzer.SetChecked(v.AlertMessageBuzzer)
			alertBellLED.SetChecked(v.AlertBellLED)
			alertBellVibra.SetChecked(v.AlertBellVibra)
			alertBellBuzzer.SetChecked(v.AlertBellBuzzer)
			usePWMBuzzer.SetChecked(v.UsePWMBuzzer)
			nodeSettingsSetUint32Select(nagTimeout, nodeSettingsNagTimeoutOptions, v.NagTimeoutSecs, nodeSettingsCustomSecondsLabel)
			ringtone.SetText(v.Ringtone)
			useI2S.SetChecked(v.UseI2SAsBuzzer)
			updateAdvancedVisibility()
		},
		read: func(base app.NodeExternalNotificationSettings, target app.NodeSettingsTarget) (app.NodeExternalNotificationSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.OutputMS, err = nodeSettingsParseMillisecondsSelectLabel("output ms", outputMS.Selected, nodeSettingsOutputDurationOptions)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("output ms", err)
			}
			base.OutputGPIO, err = parseOptionalUint32(outputGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("output GPIO", err)
			}
			base.OutputVibraGPIO, err = parseOptionalUint32(outputVibraGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("vibra GPIO", err)
			}
			base.OutputBuzzerGPIO, err = parseOptionalUint32(outputBuzzerGPIO.Text)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("buzzer GPIO", err)
			}
			base.OutputActiveHigh = outputActiveHigh.Checked
			base.AlertMessageLED = alertMessageLED.Checked
			base.AlertMessageVibra = alertMessageVibra.Checked
			base.AlertMessageBuzzer = alertMessageBuzzer.Checked
			base.AlertBellLED = alertBellLED.Checked
			base.AlertBellVibra = alertBellVibra.Checked
			base.AlertBellBuzzer = alertBellBuzzer.Checked
			base.UsePWMBuzzer = usePWMBuzzer.Checked
			base.NagTimeoutSecs, err = nodeSettingsParseUint32SelectLabel("nag timeout", nagTimeout.Selected, nodeSettingsNagTimeoutOptions)
			if err != nil {
				return app.NodeExternalNotificationSettings{}, fieldParseError("nag timeout", err)
			}
			base.Ringtone = ringtone.Text
			base.UseI2SAsBuzzer = useI2S.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, outputMS, outputGPIO, outputVibraGPIO, outputBuzzerGPIO, outputActiveHigh, alertMessageLED, alertMessageVibra, alertMessageBuzzer, alertBellLED, alertBellVibra, alertBellBuzzer, usePWMBuzzer, nagTimeout, ringtone, useI2S),
	}
}

func newNodeStoreForwardSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.store_forward", "Loading Store & Forward settings…", "Store & Forward settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error) {
			return dep.Actions.NodeSettings.LoadStoreForwardSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStoreForwardSettings) error {
			return dep.Actions.NodeSettings.SaveStoreForwardSettings(ctx, target, settings)
		},
		func(v app.NodeStoreForwardSettings) app.NodeStoreForwardSettings { return v },
		func(v app.NodeStoreForwardSettings) string {
			return fmt.Sprintf("%s|%t|%t|%d|%d|%d|%t", v.NodeID, v.Enabled, v.Heartbeat, v.Records, v.HistoryReturnMax, v.HistoryReturnWindow, v.IsServer)
		},
		buildNodeStoreForwardSettingsForm,
	)
}

func buildNodeStoreForwardSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeStoreForwardSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	heartbeat := newSettingsCheck(onChanged)
	records := newNumberEntry(onChanged)
	historyReturnMax := newNumberEntry(onChanged)
	historyReturnWindow := newNumberEntry(onChanged)
	isServer := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Heartbeat", heartbeat),
		widget.NewFormItem("Records", records),
		widget.NewFormItem("History return max", historyReturnMax),
		widget.NewFormItem("History return window", historyReturnWindow),
		widget.NewFormItem("Server mode", isServer),
	)

	return nodeManagedSettingsForm[app.NodeStoreForwardSettings]{
		content: form,
		set: func(v app.NodeStoreForwardSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			heartbeat.SetChecked(v.Heartbeat)
			records.SetText(strconv.FormatUint(uint64(v.Records), 10))
			historyReturnMax.SetText(strconv.FormatUint(uint64(v.HistoryReturnMax), 10))
			historyReturnWindow.SetText(strconv.FormatUint(uint64(v.HistoryReturnWindow), 10))
			isServer.SetChecked(v.IsServer)
		},
		read: func(base app.NodeStoreForwardSettings, target app.NodeSettingsTarget) (app.NodeStoreForwardSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.Heartbeat = heartbeat.Checked
			base.Records, err = parseOptionalUint32(records.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("records", err)
			}
			base.HistoryReturnMax, err = parseOptionalUint32(historyReturnMax.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("history return max", err)
			}
			base.HistoryReturnWindow, err = parseOptionalUint32(historyReturnWindow.Text)
			if err != nil {
				return app.NodeStoreForwardSettings{}, fieldParseError("history return window", err)
			}
			base.IsServer = isServer.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, heartbeat, records, historyReturnMax, historyReturnWindow, isServer),
	}
}

func newNodeTelemetrySettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.telemetry", "Loading telemetry settings…", "Telemetry settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeTelemetrySettings, error) {
			return dep.Actions.NodeSettings.LoadTelemetrySettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeTelemetrySettings) error {
			return dep.Actions.NodeSettings.SaveTelemetrySettings(ctx, target, settings)
		},
		func(v app.NodeTelemetrySettings) app.NodeTelemetrySettings { return v },
		func(v app.NodeTelemetrySettings) string {
			return fmt.Sprintf("%s|%d|%d|%t|%t|%t|%t|%d|%t|%d|%t|%t|%d|%t|%t|%t",
				v.NodeID, v.DeviceUpdateInterval, v.EnvironmentUpdateInterval, v.EnvironmentMeasurementEnabled,
				v.EnvironmentScreenEnabled, v.EnvironmentDisplayFahrenheit, v.AirQualityEnabled, v.AirQualityInterval,
				v.PowerMeasurementEnabled, v.PowerUpdateInterval, v.PowerScreenEnabled, v.HealthMeasurementEnabled,
				v.HealthUpdateInterval, v.HealthScreenEnabled, v.DeviceTelemetryEnabled, v.AirQualityScreenEnabled)
		},
		buildNodeTelemetrySettingsForm,
	)
}

func buildNodeTelemetrySettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeTelemetrySettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	deviceUpdateInterval := widget.NewSelect(nil, nil)
	deviceUpdateInterval.OnChanged = func(string) { onChanged() }
	environmentUpdateInterval := widget.NewSelect(nil, nil)
	environmentUpdateInterval.OnChanged = func(string) { onChanged() }
	environmentMeasurement := newSettingsCheck(onChanged)
	environmentScreen := newSettingsCheck(onChanged)
	displayFahrenheit := newSettingsCheck(onChanged)
	airQualityEnabled := newSettingsCheck(onChanged)
	airQualityInterval := widget.NewSelect(nil, nil)
	airQualityInterval.OnChanged = func(string) { onChanged() }
	powerMeasurement := newSettingsCheck(onChanged)
	powerUpdateInterval := widget.NewSelect(nil, nil)
	powerUpdateInterval.OnChanged = func(string) { onChanged() }
	powerScreen := newSettingsCheck(onChanged)
	healthMeasurement := newSettingsCheck(onChanged)
	healthUpdateInterval := widget.NewSelect(nil, nil)
	healthUpdateInterval.OnChanged = func(string) { onChanged() }
	healthScreen := newSettingsCheck(onChanged)
	deviceTelemetry := newSettingsCheck(onChanged)
	airQualityScreen := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Device update interval", deviceUpdateInterval),
		widget.NewFormItem("Environment update interval", environmentUpdateInterval),
		widget.NewFormItem("Environment measurement enabled", environmentMeasurement),
		widget.NewFormItem("Environment screen enabled", environmentScreen),
		widget.NewFormItem("Display Fahrenheit", displayFahrenheit),
		widget.NewFormItem("Air quality enabled", airQualityEnabled),
		widget.NewFormItem("Air quality interval", airQualityInterval),
		widget.NewFormItem("Power measurement enabled", powerMeasurement),
		widget.NewFormItem("Power update interval", powerUpdateInterval),
		widget.NewFormItem("Power screen enabled", powerScreen),
		widget.NewFormItem("Health measurement enabled", healthMeasurement),
		widget.NewFormItem("Health update interval", healthUpdateInterval),
		widget.NewFormItem("Health screen enabled", healthScreen),
		widget.NewFormItem("Device telemetry enabled", deviceTelemetry),
		widget.NewFormItem("Air quality screen enabled", airQualityScreen),
	)

	return nodeManagedSettingsForm[app.NodeTelemetrySettings]{
		content: form,
		set: func(v app.NodeTelemetrySettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			nodeSettingsSetUint32Select(deviceUpdateInterval, nodeSettingsBroadcastShortIntervalOptions, v.DeviceUpdateInterval, nodeSettingsCustomSecondsLabel)
			nodeSettingsSetUint32Select(environmentUpdateInterval, nodeSettingsBroadcastShortIntervalOptions, v.EnvironmentUpdateInterval, nodeSettingsCustomSecondsLabel)
			environmentMeasurement.SetChecked(v.EnvironmentMeasurementEnabled)
			environmentScreen.SetChecked(v.EnvironmentScreenEnabled)
			displayFahrenheit.SetChecked(v.EnvironmentDisplayFahrenheit)
			airQualityEnabled.SetChecked(v.AirQualityEnabled)
			nodeSettingsSetUint32Select(airQualityInterval, nodeSettingsBroadcastShortIntervalOptions, v.AirQualityInterval, nodeSettingsCustomSecondsLabel)
			powerMeasurement.SetChecked(v.PowerMeasurementEnabled)
			nodeSettingsSetUint32Select(powerUpdateInterval, nodeSettingsBroadcastShortIntervalOptions, v.PowerUpdateInterval, nodeSettingsCustomSecondsLabel)
			powerScreen.SetChecked(v.PowerScreenEnabled)
			healthMeasurement.SetChecked(v.HealthMeasurementEnabled)
			nodeSettingsSetUint32Select(healthUpdateInterval, nodeSettingsBroadcastShortIntervalOptions, v.HealthUpdateInterval, nodeSettingsCustomSecondsLabel)
			healthScreen.SetChecked(v.HealthScreenEnabled)
			deviceTelemetry.SetChecked(v.DeviceTelemetryEnabled)
			airQualityScreen.SetChecked(v.AirQualityScreenEnabled)
		},
		read: func(base app.NodeTelemetrySettings, target app.NodeSettingsTarget) (app.NodeTelemetrySettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.DeviceUpdateInterval, err = nodeSettingsParseUint32SelectLabel("device update interval", deviceUpdateInterval.Selected, nodeSettingsBroadcastShortIntervalOptions)
			if err != nil {
				return app.NodeTelemetrySettings{}, fieldParseError("device update interval", err)
			}
			base.EnvironmentUpdateInterval, err = nodeSettingsParseUint32SelectLabel("environment update interval", environmentUpdateInterval.Selected, nodeSettingsBroadcastShortIntervalOptions)
			if err != nil {
				return app.NodeTelemetrySettings{}, fieldParseError("environment update interval", err)
			}
			base.EnvironmentMeasurementEnabled = environmentMeasurement.Checked
			base.EnvironmentScreenEnabled = environmentScreen.Checked
			base.EnvironmentDisplayFahrenheit = displayFahrenheit.Checked
			base.AirQualityEnabled = airQualityEnabled.Checked
			base.AirQualityInterval, err = nodeSettingsParseUint32SelectLabel("air quality interval", airQualityInterval.Selected, nodeSettingsBroadcastShortIntervalOptions)
			if err != nil {
				return app.NodeTelemetrySettings{}, fieldParseError("air quality interval", err)
			}
			base.PowerMeasurementEnabled = powerMeasurement.Checked
			base.PowerUpdateInterval, err = nodeSettingsParseUint32SelectLabel("power update interval", powerUpdateInterval.Selected, nodeSettingsBroadcastShortIntervalOptions)
			if err != nil {
				return app.NodeTelemetrySettings{}, fieldParseError("power update interval", err)
			}
			base.PowerScreenEnabled = powerScreen.Checked
			base.HealthMeasurementEnabled = healthMeasurement.Checked
			base.HealthUpdateInterval, err = nodeSettingsParseUint32SelectLabel("health update interval", healthUpdateInterval.Selected, nodeSettingsBroadcastShortIntervalOptions)
			if err != nil {
				return app.NodeTelemetrySettings{}, fieldParseError("health update interval", err)
			}
			base.HealthScreenEnabled = healthScreen.Checked
			base.DeviceTelemetryEnabled = deviceTelemetry.Checked
			base.AirQualityScreenEnabled = airQualityScreen.Checked

			return base, nil
		},
		setSaving: disableWidgets(deviceUpdateInterval, environmentUpdateInterval, environmentMeasurement, environmentScreen, displayFahrenheit, airQualityEnabled, airQualityInterval, powerMeasurement, powerUpdateInterval, powerScreen, healthMeasurement, healthUpdateInterval, healthScreen, deviceTelemetry, airQualityScreen),
	}
}

func newNodeCannedMessageSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.canned_message", "Loading canned message settings…", "Canned message settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error) {
			return dep.Actions.NodeSettings.LoadCannedMessageSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeCannedMessageSettings) error {
			return dep.Actions.NodeSettings.SaveCannedMessageSettings(ctx, target, settings)
		},
		func(v app.NodeCannedMessageSettings) app.NodeCannedMessageSettings { return v },
		func(v app.NodeCannedMessageSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%d|%d|%t|%t|%s|%t|%s",
				v.NodeID, v.Rotary1Enabled, v.InputBrokerPinA, v.InputBrokerPinB, v.InputBrokerPinPress,
				v.InputBrokerEventCW, v.InputBrokerEventCCW, v.InputBrokerEventPress, v.UpDown1Enabled,
				v.Enabled, v.AllowInputSource, v.SendBell, v.Messages)
		},
		buildNodeCannedMessageSettingsForm,
	)
}

func buildNodeCannedMessageSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeCannedMessageSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	rotary1Enabled := newSettingsCheck(onChanged)
	pinA := newNumberEntry(onChanged)
	pinB := newNumberEntry(onChanged)
	pinPress := newNumberEntry(onChanged)
	eventCW := newNumberEntry(onChanged)
	eventCCW := newNumberEntry(onChanged)
	eventPress := newNumberEntry(onChanged)
	upDown1Enabled := newSettingsCheck(onChanged)
	enabled := newSettingsCheck(onChanged)
	allowInputSource := widget.NewEntry()
	allowInputSource.OnChanged = func(string) { onChanged() }
	sendBell := newSettingsCheck(onChanged)
	messages := widget.NewMultiLineEntry()
	messages.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Rotary 1 enabled", rotary1Enabled),
		widget.NewFormItem("Input broker pin A", pinA),
		widget.NewFormItem("Input broker pin B", pinB),
		widget.NewFormItem("Input broker pin press", pinPress),
		widget.NewFormItem("Input broker event CW", eventCW),
		widget.NewFormItem("Input broker event CCW", eventCCW),
		widget.NewFormItem("Input broker event press", eventPress),
		widget.NewFormItem("Up/down 1 enabled", upDown1Enabled),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Allow input source", allowInputSource),
		widget.NewFormItem("Send bell", sendBell),
		widget.NewFormItem("Messages", messages),
	)

	return nodeManagedSettingsForm[app.NodeCannedMessageSettings]{
		content: form,
		set: func(v app.NodeCannedMessageSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			rotary1Enabled.SetChecked(v.Rotary1Enabled)
			pinA.SetText(strconv.FormatUint(uint64(v.InputBrokerPinA), 10))
			pinB.SetText(strconv.FormatUint(uint64(v.InputBrokerPinB), 10))
			pinPress.SetText(strconv.FormatUint(uint64(v.InputBrokerPinPress), 10))
			eventCW.SetText(strconv.FormatInt(int64(v.InputBrokerEventCW), 10))
			eventCCW.SetText(strconv.FormatInt(int64(v.InputBrokerEventCCW), 10))
			eventPress.SetText(strconv.FormatInt(int64(v.InputBrokerEventPress), 10))
			upDown1Enabled.SetChecked(v.UpDown1Enabled)
			enabled.SetChecked(v.Enabled)
			allowInputSource.SetText(v.AllowInputSource)
			sendBell.SetChecked(v.SendBell)
			messages.SetText(v.Messages)
		},
		read: func(base app.NodeCannedMessageSettings, target app.NodeSettingsTarget) (app.NodeCannedMessageSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Rotary1Enabled = rotary1Enabled.Checked
			base.InputBrokerPinA, err = parseOptionalUint32(pinA.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin A", err)
			}
			base.InputBrokerPinB, err = parseOptionalUint32(pinB.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin B", err)
			}
			base.InputBrokerPinPress, err = parseOptionalUint32(pinPress.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker pin press", err)
			}
			base.InputBrokerEventCW, err = parseOptionalInt32(eventCW.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event CW", err)
			}
			base.InputBrokerEventCCW, err = parseOptionalInt32(eventCCW.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event CCW", err)
			}
			base.InputBrokerEventPress, err = parseOptionalInt32(eventPress.Text)
			if err != nil {
				return app.NodeCannedMessageSettings{}, fieldParseError("input broker event press", err)
			}
			base.UpDown1Enabled = upDown1Enabled.Checked
			base.Enabled = enabled.Checked
			base.AllowInputSource = strings.TrimSpace(allowInputSource.Text)
			base.SendBell = sendBell.Checked
			base.Messages = messages.Text

			return base, nil
		},
		setSaving: disableWidgets(rotary1Enabled, pinA, pinB, pinPress, eventCW, eventCCW, eventPress, upDown1Enabled, enabled, allowInputSource, sendBell, messages),
	}
}

func newNodeAudioSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.audio", "Loading audio settings…", "Audio settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAudioSettings, error) {
			return dep.Actions.NodeSettings.LoadAudioSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAudioSettings) error {
			return dep.Actions.NodeSettings.SaveAudioSettings(ctx, target, settings)
		},
		func(v app.NodeAudioSettings) app.NodeAudioSettings { return v },
		func(v app.NodeAudioSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d|%d|%d", v.NodeID, v.Codec2Enabled, v.PTTPin, v.Bitrate, v.I2SWordSelect, v.I2SDataIn, v.I2SDataOut, v.I2SClock)
		},
		buildNodeAudioSettingsForm,
	)
}

func buildNodeAudioSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeAudioSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	codec2Enabled := newSettingsCheck(onChanged)
	pttPin := newNumberEntry(onChanged)
	bitrate := widget.NewSelect(nil, nil)
	bitrate.OnChanged = func(string) { onChanged() }
	i2sWs := newNumberEntry(onChanged)
	i2sSd := newNumberEntry(onChanged)
	i2sDin := newNumberEntry(onChanged)
	i2sSck := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Codec2 enabled", codec2Enabled),
		widget.NewFormItem("PTT pin", pttPin),
		widget.NewFormItem("Bitrate", bitrate),
		widget.NewFormItem("I2S WS", i2sWs),
		widget.NewFormItem("I2S SD", i2sSd),
		widget.NewFormItem("I2S DIN", i2sDin),
		widget.NewFormItem("I2S SCK", i2sSck),
	)

	return nodeManagedSettingsForm[app.NodeAudioSettings]{
		content: form,
		set: func(v app.NodeAudioSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			codec2Enabled.SetChecked(v.Codec2Enabled)
			pttPin.SetText(strconv.FormatUint(uint64(v.PTTPin), 10))
			nodeSettingsSetInt32Select(bitrate, nodeSettingsAudioBitrateOptions, v.Bitrate, nodeSettingsCustomInt32Label)
			i2sWs.SetText(strconv.FormatUint(uint64(v.I2SWordSelect), 10))
			i2sSd.SetText(strconv.FormatUint(uint64(v.I2SDataIn), 10))
			i2sDin.SetText(strconv.FormatUint(uint64(v.I2SDataOut), 10))
			i2sSck.SetText(strconv.FormatUint(uint64(v.I2SClock), 10))
		},
		read: func(base app.NodeAudioSettings, target app.NodeSettingsTarget) (app.NodeAudioSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Codec2Enabled = codec2Enabled.Checked
			base.PTTPin, err = parseOptionalUint32(pttPin.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("PTT pin", err)
			}
			base.Bitrate, err = nodeSettingsParseInt32SelectLabel("bitrate", bitrate.Selected, nodeSettingsAudioBitrateOptions)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("bitrate", err)
			}
			base.I2SWordSelect, err = parseOptionalUint32(i2sWs.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S WS", err)
			}
			base.I2SDataIn, err = parseOptionalUint32(i2sSd.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S SD", err)
			}
			base.I2SDataOut, err = parseOptionalUint32(i2sDin.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S DIN", err)
			}
			base.I2SClock, err = parseOptionalUint32(i2sSck.Text)
			if err != nil {
				return app.NodeAudioSettings{}, fieldParseError("I2S SCK", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(codec2Enabled, pttPin, bitrate, i2sWs, i2sSd, i2sDin, i2sSck),
	}
}

func newNodeRemoteHardwareSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.remote_hardware", "Loading remote hardware settings…", "Remote hardware settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error) {
			return dep.Actions.NodeSettings.LoadRemoteHardwareSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeRemoteHardwareSettings) error {
			return dep.Actions.NodeSettings.SaveRemoteHardwareSettings(ctx, target, settings)
		},
		cloneNodeRemoteHardwareSettings,
		func(v app.NodeRemoteHardwareSettings) string {
			parts := make([]string, 0, len(v.AvailablePins))
			for _, pin := range v.AvailablePins {
				parts = append(parts, strconv.FormatUint(uint64(pin), 10))
			}

			return fmt.Sprintf("%s|%t|%t|%s", v.NodeID, v.Enabled, v.AllowUndefinedPinAccess, strings.Join(parts, ","))
		},
		buildNodeRemoteHardwareSettingsForm,
	)
}

func buildNodeRemoteHardwareSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeRemoteHardwareSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	allowUndefined := newSettingsCheck(onChanged)
	availablePins := widget.NewEntry()
	availablePins.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Allow undefined pin access", allowUndefined),
		widget.NewFormItem("Available pins", availablePins),
	)

	return nodeManagedSettingsForm[app.NodeRemoteHardwareSettings]{
		content: form,
		set: func(v app.NodeRemoteHardwareSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			allowUndefined.SetChecked(v.AllowUndefinedPinAccess)
			parts := make([]string, 0, len(v.AvailablePins))
			for _, pin := range v.AvailablePins {
				parts = append(parts, strconv.FormatUint(uint64(pin), 10))
			}
			availablePins.SetText(strings.Join(parts, ", "))
		},
		read: func(base app.NodeRemoteHardwareSettings, target app.NodeSettingsTarget) (app.NodeRemoteHardwareSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.Enabled = enabled.Checked
			base.AllowUndefinedPinAccess = allowUndefined.Checked
			pins, err := parseUint32List(availablePins.Text)
			if err != nil {
				return app.NodeRemoteHardwareSettings{}, fieldParseError("available pins", err)
			}
			base.AvailablePins = pins

			return base, nil
		},
		setSaving: disableWidgets(enabled, allowUndefined, availablePins),
	}
}

func newNodeNeighborInfoSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.neighbor_info", "Loading neighbor info settings…", "Neighbor info settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
			return dep.Actions.NodeSettings.LoadNeighborInfoSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeNeighborInfoSettings) error {
			return dep.Actions.NodeSettings.SaveNeighborInfoSettings(ctx, target, settings)
		},
		func(v app.NodeNeighborInfoSettings) app.NodeNeighborInfoSettings { return v },
		func(v app.NodeNeighborInfoSettings) string {
			return fmt.Sprintf("%s|%t|%d|%t", v.NodeID, v.Enabled, v.UpdateIntervalSecs, v.TransmitOverLoRa)
		},
		buildNodeNeighborInfoSettingsForm,
	)
}

func buildNodeNeighborInfoSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeNeighborInfoSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	updateInterval := widget.NewSelect(nil, nil)
	updateInterval.OnChanged = func(string) { onChanged() }
	transmitOverLoRa := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Update interval secs", updateInterval),
		widget.NewFormItem("Transmit over LoRa", transmitOverLoRa),
	)

	return nodeManagedSettingsForm[app.NodeNeighborInfoSettings]{
		content: form,
		set: func(v app.NodeNeighborInfoSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(updateInterval, nodeSettingsDetectionMinimumIntervalOptions, v.UpdateIntervalSecs, nodeSettingsCustomSecondsLabel)
			transmitOverLoRa.SetChecked(v.TransmitOverLoRa)
		},
		read: func(base app.NodeNeighborInfoSettings, target app.NodeSettingsTarget) (app.NodeNeighborInfoSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.UpdateIntervalSecs, err = nodeSettingsParseUint32SelectLabel("update interval secs", updateInterval.Selected, nodeSettingsDetectionMinimumIntervalOptions)
			if err != nil {
				return app.NodeNeighborInfoSettings{}, fieldParseError("update interval secs", err)
			}
			base.TransmitOverLoRa = transmitOverLoRa.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, updateInterval, transmitOverLoRa),
	}
}

func newNodeAmbientLightingSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.ambient_lighting", "Loading ambient lighting settings…", "Ambient lighting settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
			return dep.Actions.NodeSettings.LoadAmbientLightingSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeAmbientLightingSettings) error {
			return dep.Actions.NodeSettings.SaveAmbientLightingSettings(ctx, target, settings)
		},
		func(v app.NodeAmbientLightingSettings) app.NodeAmbientLightingSettings { return v },
		func(v app.NodeAmbientLightingSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d|%d", v.NodeID, v.LEDState, v.Current, v.Red, v.Green, v.Blue)
		},
		buildNodeAmbientLightingSettingsForm,
	)
}

func buildNodeAmbientLightingSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeAmbientLightingSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	ledState := newSettingsCheck(onChanged)
	current := newNumberEntry(onChanged)
	red := newNumberEntry(onChanged)
	green := newNumberEntry(onChanged)
	blue := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("LED state", ledState),
		widget.NewFormItem("Current", current),
		widget.NewFormItem("Red", red),
		widget.NewFormItem("Green", green),
		widget.NewFormItem("Blue", blue),
	)

	return nodeManagedSettingsForm[app.NodeAmbientLightingSettings]{
		content: form,
		set: func(v app.NodeAmbientLightingSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			ledState.SetChecked(v.LEDState)
			current.SetText(strconv.FormatUint(uint64(v.Current), 10))
			red.SetText(strconv.FormatUint(uint64(v.Red), 10))
			green.SetText(strconv.FormatUint(uint64(v.Green), 10))
			blue.SetText(strconv.FormatUint(uint64(v.Blue), 10))
		},
		read: func(base app.NodeAmbientLightingSettings, target app.NodeSettingsTarget) (app.NodeAmbientLightingSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.LEDState = ledState.Checked
			base.Current, err = parseOptionalUint32(current.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("current", err)
			}
			base.Red, err = parseOptionalUint32(red.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("red", err)
			}
			base.Green, err = parseOptionalUint32(green.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("green", err)
			}
			base.Blue, err = parseOptionalUint32(blue.Text)
			if err != nil {
				return app.NodeAmbientLightingSettings{}, fieldParseError("blue", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(ledState, current, red, green, blue),
	}
}

func newNodeDetectionSensorSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.detection_sensor", "Loading detection sensor settings…", "Detection sensor settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error) {
			return dep.Actions.NodeSettings.LoadDetectionSensorSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeDetectionSensorSettings) error {
			return dep.Actions.NodeSettings.SaveDetectionSensorSettings(ctx, target, settings)
		},
		func(v app.NodeDetectionSensorSettings) app.NodeDetectionSensorSettings { return v },
		func(v app.NodeDetectionSensorSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%t|%s|%d|%d|%t", v.NodeID, v.Enabled, v.MinimumBroadcastSecs, v.StateBroadcastSecs, v.SendBell, v.Name, v.MonitorPin, v.DetectionTriggerType, v.UsePullup)
		},
		buildNodeDetectionSensorSettingsForm,
	)
}

func buildNodeDetectionSensorSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeDetectionSensorSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	minimumBroadcast := widget.NewSelect(nil, nil)
	minimumBroadcast.OnChanged = func(string) { onChanged() }
	stateBroadcast := widget.NewSelect(nil, nil)
	stateBroadcast.OnChanged = func(string) { onChanged() }
	sendBell := newSettingsCheck(onChanged)
	name := widget.NewEntry()
	name.OnChanged = func(string) { onChanged() }
	monitorPin := newNumberEntry(onChanged)
	triggerType := widget.NewSelect(nil, nil)
	triggerType.OnChanged = func(string) { onChanged() }
	usePullup := newSettingsCheck(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Minimum broadcast secs", minimumBroadcast),
		widget.NewFormItem("State broadcast secs", stateBroadcast),
		widget.NewFormItem("Send bell", sendBell),
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Monitor pin", monitorPin),
		widget.NewFormItem("Detection trigger type", triggerType),
		widget.NewFormItem("Use pullup", usePullup),
	)

	return nodeManagedSettingsForm[app.NodeDetectionSensorSettings]{
		content: form,
		set: func(v app.NodeDetectionSensorSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(minimumBroadcast, nodeSettingsDetectionMinimumIntervalOptions, v.MinimumBroadcastSecs, nodeSettingsCustomSecondsLabel)
			nodeSettingsSetUint32Select(stateBroadcast, nodeSettingsDetectionStateIntervalOptions, v.StateBroadcastSecs, nodeSettingsCustomSecondsLabel)
			sendBell.SetChecked(v.SendBell)
			name.SetText(v.Name)
			monitorPin.SetText(strconv.FormatUint(uint64(v.MonitorPin), 10))
			nodeSettingsSetInt32Select(triggerType, nodeSettingsDetectionTriggerTypeOptions, v.DetectionTriggerType, nodeSettingsCustomInt32Label)
			usePullup.SetChecked(v.UsePullup)
		},
		read: func(base app.NodeDetectionSensorSettings, target app.NodeSettingsTarget) (app.NodeDetectionSensorSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.MinimumBroadcastSecs, err = nodeSettingsParseUint32SelectLabel("minimum broadcast secs", minimumBroadcast.Selected, nodeSettingsDetectionMinimumIntervalOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("minimum broadcast secs", err)
			}
			base.StateBroadcastSecs, err = nodeSettingsParseUint32SelectLabel("state broadcast secs", stateBroadcast.Selected, nodeSettingsDetectionStateIntervalOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("state broadcast secs", err)
			}
			base.SendBell = sendBell.Checked
			base.Name = strings.TrimSpace(name.Text)
			base.MonitorPin, err = parseOptionalUint32(monitorPin.Text)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("monitor pin", err)
			}
			base.DetectionTriggerType, err = nodeSettingsParseInt32SelectLabel("detection trigger type", triggerType.Selected, nodeSettingsDetectionTriggerTypeOptions)
			if err != nil {
				return app.NodeDetectionSensorSettings{}, fieldParseError("detection trigger type", err)
			}
			base.UsePullup = usePullup.Checked

			return base, nil
		},
		setSaving: disableWidgets(enabled, minimumBroadcast, stateBroadcast, sendBell, name, monitorPin, triggerType, usePullup),
	}
}

func newNodePaxcounterSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.paxcounter", "Loading paxcounter settings…", "Paxcounter settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error) {
			return dep.Actions.NodeSettings.LoadPaxcounterSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodePaxcounterSettings) error {
			return dep.Actions.NodeSettings.SavePaxcounterSettings(ctx, target, settings)
		},
		func(v app.NodePaxcounterSettings) app.NodePaxcounterSettings { return v },
		func(v app.NodePaxcounterSettings) string {
			return fmt.Sprintf("%s|%t|%d|%d|%d", v.NodeID, v.Enabled, v.UpdateIntervalSecs, v.WifiThreshold, v.BLEThreshold)
		},
		buildNodePaxcounterSettingsForm,
	)
}

func buildNodePaxcounterSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodePaxcounterSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	enabled := newSettingsCheck(onChanged)
	updateInterval := widget.NewSelect(nil, nil)
	updateInterval.OnChanged = func(string) { onChanged() }
	wifiThreshold := newNumberEntry(onChanged)
	bleThreshold := newNumberEntry(onChanged)
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Enabled", enabled),
		widget.NewFormItem("Update interval secs", updateInterval),
		widget.NewFormItem("WiFi threshold", wifiThreshold),
		widget.NewFormItem("BLE threshold", bleThreshold),
	)

	return nodeManagedSettingsForm[app.NodePaxcounterSettings]{
		content: form,
		set: func(v app.NodePaxcounterSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			enabled.SetChecked(v.Enabled)
			nodeSettingsSetUint32Select(updateInterval, nodeSettingsPaxcounterIntervalOptions, v.UpdateIntervalSecs, nodeSettingsCustomSecondsLabel)
			wifiThreshold.SetText(strconv.FormatInt(int64(v.WifiThreshold), 10))
			bleThreshold.SetText(strconv.FormatInt(int64(v.BLEThreshold), 10))
		},
		read: func(base app.NodePaxcounterSettings, target app.NodeSettingsTarget) (app.NodePaxcounterSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			var err error
			base.Enabled = enabled.Checked
			base.UpdateIntervalSecs, err = nodeSettingsParseUint32SelectLabel("update interval secs", updateInterval.Selected, nodeSettingsPaxcounterIntervalOptions)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("update interval secs", err)
			}
			base.WifiThreshold, err = parseOptionalInt32(wifiThreshold.Text)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("WiFi threshold", err)
			}
			base.BLEThreshold, err = parseOptionalInt32(bleThreshold.Text)
			if err != nil {
				return app.NodePaxcounterSettings{}, fieldParseError("BLE threshold", err)
			}

			return base, nil
		},
		setSaving: disableWidgets(enabled, updateInterval, wifiThreshold, bleThreshold),
	}
}

func newNodeStatusMessageSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	return newManagedNodeSettingsPage(
		dep, saveGate, "module.status_message", "Loading status message settings…", "Status message settings loaded.",
		func(ctx context.Context, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error) {
			return dep.Actions.NodeSettings.LoadStatusMessageSettings(ctx, target)
		},
		func(ctx context.Context, target app.NodeSettingsTarget, settings app.NodeStatusMessageSettings) error {
			return dep.Actions.NodeSettings.SaveStatusMessageSettings(ctx, target, settings)
		},
		func(v app.NodeStatusMessageSettings) app.NodeStatusMessageSettings { return v },
		func(v app.NodeStatusMessageSettings) string { return v.NodeID + "|" + v.NodeStatus },
		buildNodeStatusMessageSettingsForm,
	)
}

func buildNodeStatusMessageSettingsForm(onChanged func()) nodeManagedSettingsForm[app.NodeStatusMessageSettings] {
	nodeID := widget.NewLabel("unknown")
	nodeID.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewMultiLineEntry()
	status.OnChanged = func(string) { onChanged() }
	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeID),
		widget.NewFormItem("Status", status),
	)

	return nodeManagedSettingsForm[app.NodeStatusMessageSettings]{
		content: form,
		set: func(v app.NodeStatusMessageSettings) {
			nodeID.SetText(orUnknown(v.NodeID))
			status.SetText(v.NodeStatus)
		},
		read: func(base app.NodeStatusMessageSettings, target app.NodeSettingsTarget) (app.NodeStatusMessageSettings, error) {
			base.NodeID = strings.TrimSpace(target.NodeID)
			base.NodeStatus = status.Text

			return base, nil
		},
		setSaving: disableWidgets(status),
	}
}

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
