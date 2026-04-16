package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/app"
)

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
