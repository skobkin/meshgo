package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"
)

const (
	nodeSettingsCustomLabelPrefix        = "Custom ("
	nodeSettingsCustomSecondsLabelSuffix = " seconds)"
)

type nodeSettingsUint32Option struct {
	Label string
	Value uint32
}

func nodeSettingsSecondsKnownLabel(seconds uint32, zeroLabel string) string {
	if seconds == 0 {
		if zeroLabel != "" {
			return zeroLabel
		}

		return "0 seconds"
	}
	if seconds%3600 == 0 {
		hours := seconds / 3600
		if hours == 1 {
			return "1 hour"
		}

		return fmt.Sprintf("%d hours", hours)
	}
	if seconds%60 == 0 {
		minutes := seconds / 60
		if minutes == 1 {
			return "1 minute"
		}

		return fmt.Sprintf("%d minutes", minutes)
	}
	if seconds == 1 {
		return "1 second"
	}

	return fmt.Sprintf("%d seconds", seconds)
}

func nodeSettingsCustomSecondsLabel(seconds uint32) string {
	return fmt.Sprintf("%s%d%s", nodeSettingsCustomLabelPrefix, seconds, nodeSettingsCustomSecondsLabelSuffix)
}

func nodeSettingsSetUint32Select(
	selectWidget *widget.Select,
	options []nodeSettingsUint32Option,
	value uint32,
	customLabel func(uint32) string,
) {
	optionLabels := nodeSettingsUint32OptionLabels(options)
	selected := nodeSettingsUint32OptionLabel(value, options)
	if selected == "" {
		selected = customLabel(value)
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeSettingsParseUint32SelectLabel(
	fieldName string,
	selected string,
	options []nodeSettingsUint32Option,
	customSuffix string,
) (uint32, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return 0, fmt.Errorf("%s must be selected", fieldName)
	}
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}
	if strings.HasPrefix(selected, nodeSettingsCustomLabelPrefix) && strings.HasSuffix(selected, customSuffix) {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, nodeSettingsCustomLabelPrefix), customSuffix)
		value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return uint32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}

func nodeSettingsUint32OptionLabel(value uint32, options []nodeSettingsUint32Option) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return ""
}

func nodeSettingsUint32OptionLabels(options []nodeSettingsUint32Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}
