package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

const (
	nodePositionBroadcastIntervalUnset  uint32 = 0            // unset
	nodePositionBroadcastInterval1Min   uint32 = 1 * 60       // 1 minute
	nodePositionBroadcastInterval90Sec  uint32 = 90           // 90 seconds
	nodePositionBroadcastInterval5Min   uint32 = 5 * 60       // 5 minutes
	nodePositionBroadcastInterval15Min  uint32 = 15 * 60      // 15 minutes
	nodePositionBroadcastInterval1Hour  uint32 = 1 * 60 * 60  // 1 hour
	nodePositionBroadcastInterval2Hour  uint32 = 2 * 60 * 60  // 2 hours
	nodePositionBroadcastInterval3Hour  uint32 = 3 * 60 * 60  // 3 hours
	nodePositionBroadcastInterval4Hour  uint32 = 4 * 60 * 60  // 4 hours
	nodePositionBroadcastInterval5Hour  uint32 = 5 * 60 * 60  // 5 hours
	nodePositionBroadcastInterval6Hour  uint32 = 6 * 60 * 60  // 6 hours
	nodePositionBroadcastInterval12Hour uint32 = 12 * 60 * 60 // 12 hours
	nodePositionBroadcastInterval18Hour uint32 = 18 * 60 * 60 // 18 hours
	nodePositionBroadcastInterval24Hour uint32 = 24 * 60 * 60 // 24 hours
	nodePositionBroadcastInterval36Hour uint32 = 36 * 60 * 60 // 36 hours
	nodePositionBroadcastInterval48Hour uint32 = 48 * 60 * 60 // 48 hours
	nodePositionBroadcastInterval72Hour uint32 = 72 * 60 * 60 // 72 hours

	nodePositionSmartMinimumInterval15Sec uint32 = 15          // 15 seconds
	nodePositionSmartMinimumInterval30Sec uint32 = 30          // 30 seconds
	nodePositionSmartMinimumInterval45Sec uint32 = 45          // 45 seconds
	nodePositionSmartMinimumInterval1Min  uint32 = 1 * 60      // 1 minute
	nodePositionSmartMinimumInterval5Min  uint32 = 5 * 60      // 5 minutes
	nodePositionSmartMinimumInterval10Min uint32 = 10 * 60     // 10 minutes
	nodePositionSmartMinimumInterval15Min uint32 = 15 * 60     // 15 minutes
	nodePositionSmartMinimumInterval30Min uint32 = 30 * 60     // 30 minutes
	nodePositionSmartMinimumInterval1Hour uint32 = 1 * 60 * 60 // 1 hour

	nodePositionGpsUpdateIntervalUnset  uint32 = 0            // unset
	nodePositionGpsUpdateInterval8Sec   uint32 = 8            // 8 seconds
	nodePositionGpsUpdateInterval20Sec  uint32 = 20           // 20 seconds
	nodePositionGpsUpdateInterval40Sec  uint32 = 40           // 40 seconds
	nodePositionGpsUpdateInterval1Min   uint32 = 1 * 60       // 1 minute
	nodePositionGpsUpdateInterval80Sec  uint32 = 80           // 80 seconds
	nodePositionGpsUpdateInterval2Min   uint32 = 2 * 60       // 2 minutes
	nodePositionGpsUpdateInterval5Min   uint32 = 5 * 60       // 5 minutes
	nodePositionGpsUpdateInterval10Min  uint32 = 10 * 60      // 10 minutes
	nodePositionGpsUpdateInterval15Min  uint32 = 15 * 60      // 15 minutes
	nodePositionGpsUpdateInterval30Min  uint32 = 30 * 60      // 30 minutes
	nodePositionGpsUpdateInterval1Hour  uint32 = 1 * 60 * 60  // 1 hour
	nodePositionGpsUpdateInterval6Hour  uint32 = 6 * 60 * 60  // 6 hours
	nodePositionGpsUpdateInterval12Hour uint32 = 12 * 60 * 60 // 12 hours
	nodePositionGpsUpdateInterval24Hour uint32 = 24 * 60 * 60 // 24 hours
)

func newNodePositionSettingsPage(dep RuntimeDependencies, saveGate *nodeSettingsSaveGate) (fyne.CanvasObject, func()) {
	const pageID = "device.position"
	nodeSettingsTabLogger.Debug("building node position settings page", "page_id", pageID, "service_configured", dep.Actions.NodeSettings != nil)

	controls := newNodeSettingsPageControls("Loading position settings…")
	saveButton := controls.saveButton
	cancelButton := controls.cancelButton
	reloadButton := controls.reloadButton

	nodeIDLabel := widget.NewLabel("unknown")
	nodeIDLabel.TextStyle = fyne.TextStyle{Monospace: true}

	positionBroadcastSecsSelect := widget.NewSelect(nil, nil)
	smartPositionEnabledBox := widget.NewCheck("", nil)
	minimumIntervalSelect := widget.NewSelect(nil, nil)
	minimumDistanceEntry := widget.NewEntry()
	fixedPositionBox := widget.NewCheck("", nil)
	fixedLatitudeEntry := widget.NewEntry()
	fixedLongitudeEntry := widget.NewEntry()
	fixedAltitudeEntry := widget.NewEntry()
	gpsModeSelect := widget.NewSelect(nil, nil)
	gpsUpdateIntervalSelect := widget.NewSelect(nil, nil)
	rxGPIOEntry := widget.NewEntry()
	txGPIOEntry := widget.NewEntry()
	gpsEnGPIOEntry := widget.NewEntry()

	flagAltitudeBox := widget.NewCheck("Altitude", nil)
	flagAltitudeMSLBox := widget.NewCheck("Altitude MSL", nil)
	flagGeoidalSeparationBox := widget.NewCheck("Geoidal separation", nil)
	flagDOPBox := widget.NewCheck("DOP", nil)
	flagHVDOPBox := widget.NewCheck("HVDOP", nil)
	flagSatInViewBox := widget.NewCheck("Satellites in view", nil)
	flagSeqNoBox := widget.NewCheck("Sequence number", nil)
	flagTimestampBox := widget.NewCheck("Timestamp", nil)
	flagHeadingBox := widget.NewCheck("Heading", nil)
	flagSpeedBox := widget.NewCheck("Speed", nil)

	positionFlags := container.NewGridWithColumns(
		5,
		flagAltitudeBox,
		flagAltitudeMSLBox,
		flagGeoidalSeparationBox,
		flagDOPBox,
		flagHVDOPBox,
		flagSatInViewBox,
		flagSeqNoBox,
		flagTimestampBox,
		flagHeadingBox,
		flagSpeedBox,
	)

	form := widget.NewForm(
		widget.NewFormItem("Node ID", nodeIDLabel),
		widget.NewFormItem("Position broadcast interval", positionBroadcastSecsSelect),
		widget.NewFormItem("Smart position enabled", smartPositionEnabledBox),
		widget.NewFormItem("Smart minimum interval", minimumIntervalSelect),
		widget.NewFormItem("Smart minimum distance (meters)", minimumDistanceEntry),
		widget.NewFormItem("Use fixed position", fixedPositionBox),
		widget.NewFormItem("Fixed latitude", fixedLatitudeEntry),
		widget.NewFormItem("Fixed longitude", fixedLongitudeEntry),
		widget.NewFormItem("Fixed altitude (meters)", fixedAltitudeEntry),
		widget.NewFormItem("GPS mode (physical hardware)", gpsModeSelect),
		widget.NewFormItem("GPS update interval", gpsUpdateIntervalSelect),
		widget.NewFormItem("Position flags", positionFlags),
		widget.NewFormItem("GPS RX GPIO", rxGPIOEntry),
		widget.NewFormItem("GPS TX GPIO", txGPIOEntry),
		widget.NewFormItem("GPS EN GPIO", gpsEnGPIOEntry),
	)

	var (
		baseline             app.NodePositionSettings
		baselineFormValues   nodePositionSettingsFormValues
		dirty                bool
		saving               bool
		initialReloadStarted atomic.Bool
		mu                   sync.Mutex
		applyingForm         atomic.Bool
	)

	isConnected := func() bool {
		return isNodeSettingsConnected(dep)
	}

	localTarget := func() (app.NodeSettingsTarget, bool) {
		return localNodeSettingsTarget(dep)
	}

	setForm := func(settings app.NodePositionSettings) {
		nodeIDLabel.SetText(orUnknown(settings.NodeID))
		nodePositionSetBroadcastIntervalSelect(positionBroadcastSecsSelect, settings.PositionBroadcastSecs)
		smartPositionEnabledBox.SetChecked(settings.PositionBroadcastSmartEnabled)
		nodePositionSetSmartMinimumIntervalSelect(minimumIntervalSelect, settings.BroadcastSmartMinimumIntervalSecs)
		minimumDistanceEntry.SetText(strconv.FormatUint(uint64(settings.BroadcastSmartMinimumDistance), 10))
		fixedPositionBox.SetChecked(settings.FixedPosition)
		if settings.FixedLatitude != nil {
			fixedLatitudeEntry.SetText(strconv.FormatFloat(*settings.FixedLatitude, 'f', -1, 64))
		} else {
			fixedLatitudeEntry.SetText("")
		}
		if settings.FixedLongitude != nil {
			fixedLongitudeEntry.SetText(strconv.FormatFloat(*settings.FixedLongitude, 'f', -1, 64))
		} else {
			fixedLongitudeEntry.SetText("")
		}
		if settings.FixedAltitude != nil {
			fixedAltitudeEntry.SetText(strconv.FormatInt(int64(*settings.FixedAltitude), 10))
		} else {
			fixedAltitudeEntry.SetText("")
		}
		nodePositionSetEnumSelect(gpsModeSelect, nodePositionGpsModeOptions, settings.GpsMode)
		nodePositionSetGpsUpdateIntervalSelect(gpsUpdateIntervalSelect, settings.GpsUpdateInterval)
		flagAltitudeBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_ALTITUDE)))
		flagAltitudeMSLBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_ALTITUDE_MSL)))
		flagGeoidalSeparationBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_GEOIDAL_SEPARATION)))
		flagDOPBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_DOP)))
		flagHVDOPBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_HVDOP)))
		flagSatInViewBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SATINVIEW)))
		flagSeqNoBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SEQ_NO)))
		flagTimestampBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_TIMESTAMP)))
		flagHeadingBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_HEADING)))
		flagSpeedBox.SetChecked(nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SPEED)))
		rxGPIOEntry.SetText(strconv.FormatUint(uint64(settings.RxGPIO), 10))
		txGPIOEntry.SetText(strconv.FormatUint(uint64(settings.TxGPIO), 10))
		gpsEnGPIOEntry.SetText(strconv.FormatUint(uint64(settings.GpsEnGPIO), 10))
	}

	applyForm := func(settings app.NodePositionSettings) {
		applyingForm.Store(true)
		setForm(settings)
		applyingForm.Store(false)
	}

	readFormValues := func() nodePositionSettingsFormValues {
		return nodePositionSettingsFormValues{
			PositionBroadcastSecs:             strings.TrimSpace(positionBroadcastSecsSelect.Selected),
			PositionBroadcastSmartEnabled:     smartPositionEnabledBox.Checked,
			BroadcastSmartMinimumIntervalSecs: strings.TrimSpace(minimumIntervalSelect.Selected),
			BroadcastSmartMinimumDistance:     strings.TrimSpace(minimumDistanceEntry.Text),
			FixedPosition:                     fixedPositionBox.Checked,
			FixedLatitude:                     strings.TrimSpace(fixedLatitudeEntry.Text),
			FixedLongitude:                    strings.TrimSpace(fixedLongitudeEntry.Text),
			FixedAltitude:                     strings.TrimSpace(fixedAltitudeEntry.Text),
			GpsMode:                           strings.TrimSpace(gpsModeSelect.Selected),
			GpsUpdateInterval:                 strings.TrimSpace(gpsUpdateIntervalSelect.Selected),
			FlagAltitude:                      flagAltitudeBox.Checked,
			FlagAltitudeMSL:                   flagAltitudeMSLBox.Checked,
			FlagGeoidalSeparation:             flagGeoidalSeparationBox.Checked,
			FlagDOP:                           flagDOPBox.Checked,
			FlagHVDOP:                         flagHVDOPBox.Checked,
			FlagSatInView:                     flagSatInViewBox.Checked,
			FlagSeqNo:                         flagSeqNoBox.Checked,
			FlagTimestamp:                     flagTimestampBox.Checked,
			FlagHeading:                       flagHeadingBox.Checked,
			FlagSpeed:                         flagSpeedBox.Checked,
			RxGPIO:                            strings.TrimSpace(rxGPIOEntry.Text),
			TxGPIO:                            strings.TrimSpace(txGPIOEntry.Text),
			GpsEnGPIO:                         strings.TrimSpace(gpsEnGPIOEntry.Text),
		}
	}

	updateFieldAvailability := func() {
		mu.Lock()
		isSaving := saving
		mu.Unlock()

		if isSaving {
			smartPositionEnabledBox.Disable()
			fixedPositionBox.Disable()
		} else {
			smartPositionEnabledBox.Enable()
			fixedPositionBox.Enable()
		}

		if !isSaving && smartPositionEnabledBox.Checked {
			minimumIntervalSelect.Enable()
			minimumDistanceEntry.Enable()
		} else {
			minimumIntervalSelect.Disable()
			minimumDistanceEntry.Disable()
		}

		if !isSaving && fixedPositionBox.Checked {
			fixedLatitudeEntry.Enable()
			fixedLongitudeEntry.Enable()
			fixedAltitudeEntry.Enable()
		} else {
			fixedLatitudeEntry.Disable()
			fixedLongitudeEntry.Disable()
			fixedAltitudeEntry.Disable()
		}

		if !isSaving && !fixedPositionBox.Checked {
			gpsModeSelect.Enable()
			gpsUpdateIntervalSelect.Enable()
		} else {
			gpsModeSelect.Disable()
			gpsUpdateIntervalSelect.Disable()
		}
	}

	updateButtons := func() {
		mu.Lock()
		activePage := ""
		if saveGate != nil {
			activePage = strings.TrimSpace(saveGate.ActivePage())
		}
		connected := isConnected()
		canSave := dep.Actions.NodeSettings != nil && connected && !saving && dirty && (activePage == "" || activePage == pageID)
		canCancel := !saving && dirty
		canReload := dep.Actions.NodeSettings != nil && !saving
		mu.Unlock()

		if canSave {
			saveButton.Enable()
		} else {
			saveButton.Disable()
		}
		if canCancel {
			cancelButton.Enable()
		} else {
			cancelButton.Disable()
		}
		if canReload {
			reloadButton.Enable()
		} else {
			reloadButton.Disable()
		}
		updateFieldAvailability()
	}

	markDirty := func() {
		if applyingForm.Load() {
			return
		}
		mu.Lock()
		dirty = readFormValues() != baselineFormValues
		mu.Unlock()
		updateButtons()
	}

	applyLoadedSettings := func(next app.NodePositionSettings) {
		mu.Lock()
		next = enrichNodePositionSettingsWithLocalCoordinates(next, dep)
		if next.FixedPosition {
			if next.FixedLatitude == nil && baseline.FixedLatitude != nil {
				v := *baseline.FixedLatitude
				next.FixedLatitude = &v
			}
			if next.FixedLongitude == nil && baseline.FixedLongitude != nil {
				v := *baseline.FixedLongitude
				next.FixedLongitude = &v
			}
			if next.FixedAltitude == nil && baseline.FixedAltitude != nil {
				v := *baseline.FixedAltitude
				next.FixedAltitude = &v
			}
		}
		baseline = cloneNodePositionSettings(next)
		baselineFormValues = nodePositionFormValuesFromSettings(baseline)
		dirty = false
		settings := cloneNodePositionSettings(baseline)
		mu.Unlock()
		applyForm(settings)
		updateButtons()
	}

	buildSettingsFromForm := func(target app.NodeSettingsTarget) (app.NodePositionSettings, error) {
		positionBroadcastSecs, err := nodePositionParseBroadcastIntervalLabel("position broadcast interval", positionBroadcastSecsSelect.Selected)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		broadcastSmartMinimumIntervalSecs, err := nodePositionParseSmartMinimumIntervalLabel("smart minimum interval", minimumIntervalSelect.Selected)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		broadcastSmartMinimumDistance, err := parseNodePositionUint32Field("smart minimum distance", minimumDistanceEntry.Text)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		gpsMode, err := nodePositionParseEnumLabel("GPS mode", gpsModeSelect.Selected, nodePositionGpsModeOptions)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		gpsUpdateInterval, err := nodePositionParseGpsUpdateIntervalLabel("GPS update interval", gpsUpdateIntervalSelect.Selected)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		rxGPIO, err := parseNodePositionUint32Field("GPS RX GPIO", rxGPIOEntry.Text)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		txGPIO, err := parseNodePositionUint32Field("GPS TX GPIO", txGPIOEntry.Text)
		if err != nil {
			return app.NodePositionSettings{}, err
		}
		gpsEnGPIO, err := parseNodePositionUint32Field("GPS EN GPIO", gpsEnGPIOEntry.Text)
		if err != nil {
			return app.NodePositionSettings{}, err
		}

		formValues := readFormValues()

		mu.Lock()
		next := cloneNodePositionSettings(baseline)
		mu.Unlock()
		previousFixed := next.FixedPosition

		next.NodeID = strings.TrimSpace(target.NodeID)
		next.PositionBroadcastSecs = positionBroadcastSecs
		next.PositionBroadcastSmartEnabled = smartPositionEnabledBox.Checked
		next.BroadcastSmartMinimumIntervalSecs = broadcastSmartMinimumIntervalSecs
		next.BroadcastSmartMinimumDistance = broadcastSmartMinimumDistance
		next.FixedPosition = fixedPositionBox.Checked
		next.RemoveFixedPosition = !next.FixedPosition && previousFixed
		next.FixedLatitude = nil
		next.FixedLongitude = nil
		next.FixedAltitude = nil
		if next.FixedPosition {
			fixedLatitude, err := parseNodePositionLatitudeField("fixed latitude", fixedLatitudeEntry.Text)
			if err != nil {
				return app.NodePositionSettings{}, err
			}
			fixedLongitude, err := parseNodePositionLongitudeField("fixed longitude", fixedLongitudeEntry.Text)
			if err != nil {
				return app.NodePositionSettings{}, err
			}
			fixedAltitude, err := parseNodePositionInt32Field("fixed altitude", fixedAltitudeEntry.Text)
			if err != nil {
				return app.NodePositionSettings{}, err
			}
			next.FixedLatitude = &fixedLatitude
			next.FixedLongitude = &fixedLongitude
			next.FixedAltitude = &fixedAltitude
			next.RemoveFixedPosition = false
		}
		next.GpsMode = gpsMode
		next.GpsUpdateInterval = gpsUpdateInterval
		next.PositionFlags = nodePositionComposeFlags(formValues)
		next.RxGPIO = rxGPIO
		next.TxGPIO = txGPIO
		next.GpsEnGPIO = gpsEnGPIO

		return next, nil
	}

	reloadFromDevice := func() {
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node position settings reload failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Reload failed: local node ID is not known yet.", 0, 1)

			return
		}
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node position settings reload unavailable: service is not configured", "page_id", pageID)
			controls.SetStatus("Reload is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node position settings reload blocked: device is disconnected", "page_id", pageID, "node_id", target.NodeID)
			controls.SetStatus("Reload from device is unavailable while disconnected.", 0, 2)

			return
		}

		nodeSettingsTabLogger.Info("reloading node position settings from device", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Reloading position settings from device…", 1, 2)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			loaded, err := dep.Actions.NodeSettings.LoadPositionSettings(ctx, target)
			fyne.Do(func() {
				if err != nil {
					nodeSettingsTabLogger.Warn("reloading node position settings from device failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Reload failed: "+err.Error(), 0, 2)
					updateButtons()

					return
				}
				applyLoadedSettings(loaded)
				nodeSettingsTabLogger.Info("reloaded node position settings from device", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Reloaded position settings from device.", 2, 2)
			})
		}()
	}

	positionBroadcastSecsSelect.OnChanged = func(_ string) { markDirty() }
	smartPositionEnabledBox.OnChanged = func(_ bool) { markDirty() }
	minimumIntervalSelect.OnChanged = func(_ string) { markDirty() }
	minimumDistanceEntry.OnChanged = func(_ string) { markDirty() }
	fixedPositionBox.OnChanged = func(_ bool) { markDirty() }
	fixedLatitudeEntry.OnChanged = func(_ string) { markDirty() }
	fixedLongitudeEntry.OnChanged = func(_ string) { markDirty() }
	fixedAltitudeEntry.OnChanged = func(_ string) { markDirty() }
	gpsModeSelect.OnChanged = func(_ string) { markDirty() }
	gpsUpdateIntervalSelect.OnChanged = func(_ string) { markDirty() }
	flagAltitudeBox.OnChanged = func(_ bool) { markDirty() }
	flagAltitudeMSLBox.OnChanged = func(_ bool) { markDirty() }
	flagGeoidalSeparationBox.OnChanged = func(_ bool) { markDirty() }
	flagDOPBox.OnChanged = func(_ bool) { markDirty() }
	flagHVDOPBox.OnChanged = func(_ bool) { markDirty() }
	flagSatInViewBox.OnChanged = func(_ bool) { markDirty() }
	flagSeqNoBox.OnChanged = func(_ bool) { markDirty() }
	flagTimestampBox.OnChanged = func(_ bool) { markDirty() }
	flagHeadingBox.OnChanged = func(_ bool) { markDirty() }
	flagSpeedBox.OnChanged = func(_ bool) { markDirty() }
	rxGPIOEntry.OnChanged = func(_ string) { markDirty() }
	txGPIOEntry.OnChanged = func(_ string) { markDirty() }
	gpsEnGPIOEntry.OnChanged = func(_ string) { markDirty() }

	cancelButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node position settings edit canceled", "page_id", pageID)
		mu.Lock()
		settings := cloneNodePositionSettings(baseline)
		dirty = false
		mu.Unlock()
		applyForm(settings)
		controls.SetStatus("Local edits reverted.", 1, 1)
		updateButtons()
	}

	saveButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node position settings save requested", "page_id", pageID)
		if dep.Actions.NodeSettings == nil {
			nodeSettingsTabLogger.Warn("node position settings save unavailable: service is not configured")
			controls.SetStatus("Save is unavailable: node settings service is not configured.", 0, 1)

			return
		}
		if !isConnected() {
			nodeSettingsTabLogger.Info("node position settings save blocked: device is disconnected", "page_id", pageID)
			controls.SetStatus("Save is unavailable while disconnected.", 0, 1)
			updateButtons()

			return
		}
		target, ok := localTarget()
		if !ok {
			nodeSettingsTabLogger.Warn("node position settings save failed: local node ID is unknown", "page_id", pageID)
			controls.SetStatus("Save failed: local node ID is not known yet.", 0, 1)

			return
		}
		if saveGate != nil && !saveGate.TryAcquire(pageID) {
			nodeSettingsTabLogger.Info("node position settings save blocked: another page save is active", "page_id", pageID, "active_page", saveGate.ActivePage())
			controls.SetStatus("Another settings save is in progress on a different page.", 0, 1)
			updateButtons()

			return
		}

		next, err := buildSettingsFromForm(target)
		if err != nil {
			if saveGate != nil {
				saveGate.Release(pageID)
			}
			nodeSettingsTabLogger.Warn("node position settings save failed: invalid form values", "page_id", pageID, "error", err)
			controls.SetStatus("Save failed: "+err.Error(), 0, 1)
			updateButtons()

			return
		}

		mu.Lock()
		saving = true
		mu.Unlock()
		nodeSettingsTabLogger.Info("saving node position settings", "page_id", pageID, "node_id", target.NodeID)
		controls.SetStatus("Saving position settings…", 1, 3)
		updateButtons()

		go func(settings app.NodePositionSettings) {
			ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout)
			defer cancel()
			err := dep.Actions.NodeSettings.SavePositionSettings(ctx, target, settings)
			fyne.Do(func() {
				mu.Lock()
				saving = false
				if saveGate != nil {
					saveGate.Release(pageID)
				}
				if err != nil {
					nodeSettingsTabLogger.Warn("saving node position settings failed", "page_id", pageID, "node_id", target.NodeID, "error", err)
					controls.SetStatus("Save failed: "+err.Error(), 0, 3)
					mu.Unlock()
					updateButtons()

					return
				}
				baseline = cloneNodePositionSettings(settings)
				baselineFormValues = nodePositionFormValuesFromSettings(baseline)
				dirty = false
				applied := cloneNodePositionSettings(baseline)
				mu.Unlock()

				applyForm(applied)
				nodeSettingsTabLogger.Info("saved node position settings", "page_id", pageID, "node_id", target.NodeID)
				controls.SetStatus("Saved position settings.", 3, 3)
				updateButtons()
			})
		}(next)
	}

	reloadButton.OnTapped = func() {
		nodeSettingsTabLogger.Info("node position settings reload requested", "page_id", pageID)
		reloadFromDevice()
	}

	initial := app.NodePositionSettings{
		GpsMode: int32(generated.Config_PositionConfig_DISABLED),
	}
	if target, ok := localTarget(); ok {
		initial.NodeID = target.NodeID
	}
	applyLoadedSettings(initial)
	if dep.Actions.NodeSettings == nil {
		controls.SetStatus("Position settings are unavailable: node settings service is not configured.", 0, 1)
	} else {
		controls.SetStatus("Position settings will load when this tab is opened.", 0, 1)
	}

	if dep.Data.Bus != nil {
		nodeSettingsTabLogger.Debug("starting node position settings page listener for connection status updates", "page_id", pageID)
		connSub := dep.Data.Bus.Subscribe(connectors.TopicConnStatus)
		go func() {
			for range connSub {
				fyne.Do(func() {
					nodeSettingsTabLogger.Debug("received connection status update for node position settings page", "page_id", pageID)
					updateButtons()
				})
			}
		}()
	}
	updateButtons()

	content := container.NewVBox(
		widget.NewLabel("Position settings are loaded from and saved to the connected local node."),
		form,
	)

	onTabOpened := func() {
		if dep.Actions.NodeSettings == nil {
			return
		}
		if !initialReloadStarted.CompareAndSwap(false, true) {
			return
		}
		nodeSettingsTabLogger.Debug("starting lazy initial load for node position settings", "page_id", pageID)
		reloadFromDevice()
	}

	return wrapNodeSettingsPage(content, controls), onTabOpened
}

type nodePositionSettingsFormValues struct {
	PositionBroadcastSecs             string
	PositionBroadcastSmartEnabled     bool
	BroadcastSmartMinimumIntervalSecs string
	BroadcastSmartMinimumDistance     string
	FixedPosition                     bool
	FixedLatitude                     string
	FixedLongitude                    string
	FixedAltitude                     string
	GpsMode                           string
	GpsUpdateInterval                 string
	FlagAltitude                      bool
	FlagAltitudeMSL                   bool
	FlagGeoidalSeparation             bool
	FlagDOP                           bool
	FlagHVDOP                         bool
	FlagSatInView                     bool
	FlagSeqNo                         bool
	FlagTimestamp                     bool
	FlagHeading                       bool
	FlagSpeed                         bool
	RxGPIO                            string
	TxGPIO                            string
	GpsEnGPIO                         string
}

type nodePositionEnumOption struct {
	Label string
	Value int32
}

var nodePositionGpsModeOptions = []nodePositionEnumOption{
	{Label: "Disabled", Value: int32(generated.Config_PositionConfig_DISABLED)},
	{Label: "Enabled", Value: int32(generated.Config_PositionConfig_ENABLED)},
	{Label: "Not present", Value: int32(generated.Config_PositionConfig_NOT_PRESENT)},
}

var nodePositionBroadcastIntervalOptions = []nodeSettingsUint32Option{
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastIntervalUnset, "Unset"), Value: nodePositionBroadcastIntervalUnset},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval1Min, "Unset"), Value: nodePositionBroadcastInterval1Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval90Sec, "Unset"), Value: nodePositionBroadcastInterval90Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval5Min, "Unset"), Value: nodePositionBroadcastInterval5Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval15Min, "Unset"), Value: nodePositionBroadcastInterval15Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval1Hour, "Unset"), Value: nodePositionBroadcastInterval1Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval2Hour, "Unset"), Value: nodePositionBroadcastInterval2Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval3Hour, "Unset"), Value: nodePositionBroadcastInterval3Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval4Hour, "Unset"), Value: nodePositionBroadcastInterval4Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval5Hour, "Unset"), Value: nodePositionBroadcastInterval5Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval6Hour, "Unset"), Value: nodePositionBroadcastInterval6Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval12Hour, "Unset"), Value: nodePositionBroadcastInterval12Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval18Hour, "Unset"), Value: nodePositionBroadcastInterval18Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval24Hour, "Unset"), Value: nodePositionBroadcastInterval24Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval36Hour, "Unset"), Value: nodePositionBroadcastInterval36Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval48Hour, "Unset"), Value: nodePositionBroadcastInterval48Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionBroadcastInterval72Hour, "Unset"), Value: nodePositionBroadcastInterval72Hour},
}

var nodePositionSmartMinimumIntervalOptions = []nodeSettingsUint32Option{
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval15Sec, "Unset"), Value: nodePositionSmartMinimumInterval15Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval30Sec, "Unset"), Value: nodePositionSmartMinimumInterval30Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval45Sec, "Unset"), Value: nodePositionSmartMinimumInterval45Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval1Min, "Unset"), Value: nodePositionSmartMinimumInterval1Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval5Min, "Unset"), Value: nodePositionSmartMinimumInterval5Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval10Min, "Unset"), Value: nodePositionSmartMinimumInterval10Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval15Min, "Unset"), Value: nodePositionSmartMinimumInterval15Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval30Min, "Unset"), Value: nodePositionSmartMinimumInterval30Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionSmartMinimumInterval1Hour, "Unset"), Value: nodePositionSmartMinimumInterval1Hour},
}

var nodePositionGpsUpdateIntervalOptions = []nodeSettingsUint32Option{
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateIntervalUnset, "Unset"), Value: nodePositionGpsUpdateIntervalUnset},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval8Sec, "Unset"), Value: nodePositionGpsUpdateInterval8Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval20Sec, "Unset"), Value: nodePositionGpsUpdateInterval20Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval40Sec, "Unset"), Value: nodePositionGpsUpdateInterval40Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval1Min, "Unset"), Value: nodePositionGpsUpdateInterval1Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval80Sec, "Unset"), Value: nodePositionGpsUpdateInterval80Sec},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval2Min, "Unset"), Value: nodePositionGpsUpdateInterval2Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval5Min, "Unset"), Value: nodePositionGpsUpdateInterval5Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval10Min, "Unset"), Value: nodePositionGpsUpdateInterval10Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval15Min, "Unset"), Value: nodePositionGpsUpdateInterval15Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval30Min, "Unset"), Value: nodePositionGpsUpdateInterval30Min},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval1Hour, "Unset"), Value: nodePositionGpsUpdateInterval1Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval6Hour, "Unset"), Value: nodePositionGpsUpdateInterval6Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval12Hour, "Unset"), Value: nodePositionGpsUpdateInterval12Hour},
	{Label: nodeSettingsSecondsKnownLabel(nodePositionGpsUpdateInterval24Hour, "Unset"), Value: nodePositionGpsUpdateInterval24Hour},
}

func nodePositionFormValuesFromSettings(settings app.NodePositionSettings) nodePositionSettingsFormValues {
	fixedLatitude := ""
	if settings.FixedLatitude != nil {
		fixedLatitude = strconv.FormatFloat(*settings.FixedLatitude, 'f', -1, 64)
	}
	fixedLongitude := ""
	if settings.FixedLongitude != nil {
		fixedLongitude = strconv.FormatFloat(*settings.FixedLongitude, 'f', -1, 64)
	}
	fixedAltitude := ""
	if settings.FixedAltitude != nil {
		fixedAltitude = strconv.FormatInt(int64(*settings.FixedAltitude), 10)
	}

	return nodePositionSettingsFormValues{
		PositionBroadcastSecs:             nodePositionBroadcastIntervalSelectLabel(settings.PositionBroadcastSecs),
		PositionBroadcastSmartEnabled:     settings.PositionBroadcastSmartEnabled,
		BroadcastSmartMinimumIntervalSecs: nodePositionSmartMinimumIntervalSelectLabel(settings.BroadcastSmartMinimumIntervalSecs),
		BroadcastSmartMinimumDistance:     strconv.FormatUint(uint64(settings.BroadcastSmartMinimumDistance), 10),
		FixedPosition:                     settings.FixedPosition,
		FixedLatitude:                     fixedLatitude,
		FixedLongitude:                    fixedLongitude,
		FixedAltitude:                     fixedAltitude,
		GpsMode:                           nodePositionEnumLabel(settings.GpsMode, nodePositionGpsModeOptions),
		GpsUpdateInterval:                 nodePositionGpsUpdateIntervalSelectLabel(settings.GpsUpdateInterval),
		FlagAltitude:                      nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_ALTITUDE)),
		FlagAltitudeMSL:                   nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_ALTITUDE_MSL)),
		FlagGeoidalSeparation:             nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_GEOIDAL_SEPARATION)),
		FlagDOP:                           nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_DOP)),
		FlagHVDOP:                         nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_HVDOP)),
		FlagSatInView:                     nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SATINVIEW)),
		FlagSeqNo:                         nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SEQ_NO)),
		FlagTimestamp:                     nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_TIMESTAMP)),
		FlagHeading:                       nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_HEADING)),
		FlagSpeed:                         nodePositionFlagSet(settings.PositionFlags, uint32(generated.Config_PositionConfig_SPEED)),
		RxGPIO:                            strconv.FormatUint(uint64(settings.RxGPIO), 10),
		TxGPIO:                            strconv.FormatUint(uint64(settings.TxGPIO), 10),
		GpsEnGPIO:                         strconv.FormatUint(uint64(settings.GpsEnGPIO), 10),
	}
}

func nodePositionComposeFlags(values nodePositionSettingsFormValues) uint32 {
	flags := uint32(0)
	if values.FlagAltitude {
		flags |= uint32(generated.Config_PositionConfig_ALTITUDE)
	}
	if values.FlagAltitudeMSL {
		flags |= uint32(generated.Config_PositionConfig_ALTITUDE_MSL)
	}
	if values.FlagGeoidalSeparation {
		flags |= uint32(generated.Config_PositionConfig_GEOIDAL_SEPARATION)
	}
	if values.FlagDOP {
		flags |= uint32(generated.Config_PositionConfig_DOP)
	}
	if values.FlagHVDOP {
		flags |= uint32(generated.Config_PositionConfig_HVDOP)
	}
	if values.FlagSatInView {
		flags |= uint32(generated.Config_PositionConfig_SATINVIEW)
	}
	if values.FlagSeqNo {
		flags |= uint32(generated.Config_PositionConfig_SEQ_NO)
	}
	if values.FlagTimestamp {
		flags |= uint32(generated.Config_PositionConfig_TIMESTAMP)
	}
	if values.FlagHeading {
		flags |= uint32(generated.Config_PositionConfig_HEADING)
	}
	if values.FlagSpeed {
		flags |= uint32(generated.Config_PositionConfig_SPEED)
	}

	return flags
}

func nodePositionFlagSet(flags, flag uint32) bool {
	return flags&flag != 0
}

func cloneNodePositionSettings(settings app.NodePositionSettings) app.NodePositionSettings {
	out := settings
	if settings.FixedLatitude != nil {
		v := *settings.FixedLatitude
		out.FixedLatitude = &v
	}
	if settings.FixedLongitude != nil {
		v := *settings.FixedLongitude
		out.FixedLongitude = &v
	}
	if settings.FixedAltitude != nil {
		v := *settings.FixedAltitude
		out.FixedAltitude = &v
	}

	return out
}

func parseNodePositionUint32Field(fieldName, raw string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a non-negative integer", fieldName)
	}

	return uint32(value), nil
}

func parseNodePositionInt32Field(fieldName, raw string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", fieldName)
	}

	return int32(value), nil
}

func parseNodePositionLatitudeField(fieldName, raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", fieldName)
	}
	if value < -90 || value > 90 {
		return 0, fmt.Errorf("%s must be between -90 and 90", fieldName)
	}

	return value, nil
}

func parseNodePositionLongitudeField(fieldName, raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", fieldName)
	}
	if value < -180 || value > 180 {
		return 0, fmt.Errorf("%s must be between -180 and 180", fieldName)
	}

	return value, nil
}

func nodePositionSetBroadcastIntervalSelect(selectWidget *widget.Select, value uint32) {
	nodeSettingsSetUint32Select(selectWidget, nodePositionBroadcastIntervalOptions, value, nodeSettingsCustomSecondsLabel)
}

func nodePositionSetSmartMinimumIntervalSelect(selectWidget *widget.Select, value uint32) {
	nodeSettingsSetUint32Select(selectWidget, nodePositionSmartMinimumIntervalOptions, value, nodeSettingsCustomSecondsLabel)
}

func nodePositionSetGpsUpdateIntervalSelect(selectWidget *widget.Select, value uint32) {
	nodeSettingsSetUint32Select(selectWidget, nodePositionGpsUpdateIntervalOptions, value, nodeSettingsCustomSecondsLabel)
}

func nodePositionBroadcastIntervalSelectLabel(value uint32) string {
	label := nodeSettingsUint32OptionLabel(value, nodePositionBroadcastIntervalOptions)
	if label != "" {
		return label
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodePositionSmartMinimumIntervalSelectLabel(value uint32) string {
	label := nodeSettingsUint32OptionLabel(value, nodePositionSmartMinimumIntervalOptions)
	if label != "" {
		return label
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodePositionGpsUpdateIntervalSelectLabel(value uint32) string {
	label := nodeSettingsUint32OptionLabel(value, nodePositionGpsUpdateIntervalOptions)
	if label != "" {
		return label
	}

	return nodeSettingsCustomSecondsLabel(value)
}

func nodePositionParseBroadcastIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeSettingsParseUint32SelectLabel(
		fieldName,
		selected,
		nodePositionBroadcastIntervalOptions,
		nodeSettingsCustomSecondsLabelSuffix,
	)
}

func nodePositionParseSmartMinimumIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeSettingsParseUint32SelectLabel(
		fieldName,
		selected,
		nodePositionSmartMinimumIntervalOptions,
		nodeSettingsCustomSecondsLabelSuffix,
	)
}

func nodePositionParseGpsUpdateIntervalLabel(fieldName, selected string) (uint32, error) {
	return nodeSettingsParseUint32SelectLabel(
		fieldName,
		selected,
		nodePositionGpsUpdateIntervalOptions,
		nodeSettingsCustomSecondsLabelSuffix,
	)
}

func enrichNodePositionSettingsWithLocalCoordinates(settings app.NodePositionSettings, dep RuntimeDependencies) app.NodePositionSettings {
	if dep.Data.NodeStore == nil {
		return settings
	}
	node, ok := localNodeSnapshot(dep.Data.NodeStore, dep.Data.LocalNodeID)
	if !ok {
		return settings
	}
	if node.Latitude != nil {
		v := *node.Latitude
		settings.FixedLatitude = &v
	}
	if node.Longitude != nil {
		v := *node.Longitude
		settings.FixedLongitude = &v
	}
	if node.Altitude != nil {
		v := *node.Altitude
		settings.FixedAltitude = &v
	}

	return settings
}

func nodePositionSetEnumSelect(selectWidget *widget.Select, options []nodePositionEnumOption, value int32) {
	optionLabels := nodePositionEnumOptionsLabels(options)
	selected := nodePositionEnumLabel(value, options)
	if strings.HasPrefix(selected, "Unknown (") {
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodePositionEnumLabel(value int32, options []nodePositionEnumOption) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return nodePositionUnknownEnumLabel(value)
}

func nodePositionEnumOptionsLabels(options []nodePositionEnumOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}

func nodePositionUnknownEnumLabel(value int32) string {
	return fmt.Sprintf("Unknown (%d)", value)
}

func nodePositionParseEnumLabel(fieldName, selected string, options []nodePositionEnumOption) (int32, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return 0, fmt.Errorf("%s must be selected", fieldName)
	}
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}

	const (
		prefix = "Unknown ("
		suffix = ")"
	)
	if strings.HasPrefix(selected, prefix) && strings.HasSuffix(selected, suffix) {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, prefix), suffix)
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return int32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}
