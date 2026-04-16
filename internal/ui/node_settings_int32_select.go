package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/widget"
)

const nodeSettingsCustomInt32LabelPrefix = "Custom ("

type nodeSettingsInt32Option struct {
	Label string
	Value int32
}

func nodeSettingsCustomInt32Label(value int32) string {
	return fmt.Sprintf("%s%d)", nodeSettingsCustomInt32LabelPrefix, value)
}

func nodeSettingsSetInt32Select(
	selectWidget *widget.Select,
	options []nodeSettingsInt32Option,
	value int32,
	customLabel func(int32) string,
) {
	optionLabels := nodeSettingsInt32OptionLabels(options)
	selected := nodeSettingsInt32OptionLabel(value, options)
	if selected == "" {
		selected = customLabel(value)
		optionLabels = append(optionLabels, selected)
	}
	selectWidget.SetOptions(optionLabels)
	selectWidget.SetSelected(selected)
}

func nodeSettingsParseInt32SelectLabel(
	fieldName string,
	selected string,
	options []nodeSettingsInt32Option,
) (int32, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return 0, fmt.Errorf("%s must be selected", fieldName)
	}
	for _, option := range options {
		if option.Label == selected {
			return option.Value, nil
		}
	}
	if strings.HasPrefix(selected, nodeSettingsCustomInt32LabelPrefix) && strings.HasSuffix(selected, ")") {
		raw := strings.TrimSuffix(strings.TrimPrefix(selected, nodeSettingsCustomInt32LabelPrefix), ")")
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("%s has invalid value", fieldName)
		}

		return int32(value), nil
	}

	return 0, fmt.Errorf("%s has unsupported value", fieldName)
}

func nodeSettingsInt32OptionLabel(value int32, options []nodeSettingsInt32Option) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}

	return ""
}

func nodeSettingsInt32OptionLabels(options []nodeSettingsInt32Option) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		out = append(out, option.Label)
	}

	return out
}
