package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
)

type channelShareOption struct {
	Index    int
	Title    string
	Settings meshapp.NodeChannelSettings
}

func handleChannelShareAction(window fyne.Window, dep RuntimeDependencies, chat domain.Chat) {
	if window == nil {
		window = currentRuntimeWindow(dep)
	}
	if window == nil {
		return
	}
	if dep.Actions.NodeSettings == nil {
		showErrorModal(dep, fmt.Errorf("channel sharing is unavailable: node settings service is not configured"))

		return
	}
	if !isNodeSettingsConnected(dep) {
		showInfoModal(dep, "Channel sharing", "Channel sharing is available only while connected to a device.")

		return
	}

	target, ok := localNodeSettingsTarget(dep)
	if !ok {
		showErrorModal(dep, fmt.Errorf("channel sharing is unavailable: local node ID is not known yet"))

		return
	}

	loading := showBusyDialog(window, "Channel sharing", "Loading current channel and LoRa settings from the connected device…")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), nodeSettingsOpTimeout*3)
		defer cancel()

		loadedChannels, err := dep.Actions.NodeSettings.LoadChannelSettings(ctx, target)
		if err != nil {
			fyne.Do(func() {
				if loading != nil {
					loading.Hide()
				}
				showErrorModal(dep, fmt.Errorf("load channel settings for sharing: %w", err))
			})

			return
		}

		loraSettings, err := dep.Actions.NodeSettings.LoadLoRaSettings(ctx, target)
		if err != nil {
			fyne.Do(func() {
				if loading != nil {
					loading.Hide()
				}
				showErrorModal(dep, fmt.Errorf("load LoRa settings for sharing: %w", err))
			})

			return
		}

		fyne.Do(func() {
			if loading != nil {
				loading.Hide()
			}
			showChannelShareConfigDialog(window, dep, chat, loadedChannels, loraSettings)
		})
	}()
}

func showChannelShareConfigDialog(
	window fyne.Window,
	dep RuntimeDependencies,
	chat domain.Chat,
	loaded meshapp.NodeChannelSettingsList,
	lora meshapp.NodeLoRaSettings,
) {
	if window == nil {
		return
	}

	presetTitle := nodeLoRaPrimaryChannelTitle(lora, "")
	options := buildChannelShareOptions(loaded.Channels, presetTitle)
	if len(options) == 0 {
		showInfoModal(dep, "Channel sharing", "There are no channels available to share.")

		return
	}

	selected := initialChannelShareSelection(options, channelIndexFromChatKey(chat.Key))
	modeGroup := widget.NewRadioGroup([]string{"Replace", "Add"}, nil)
	modeGroup.Horizontal = true
	modeGroup.SetSelected("Replace")

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	channelList := container.NewVBox()
	for _, option := range options {
		option := option
		title := option.Title
		if title == "" {
			title = fmt.Sprintf("Channel %d", option.Index+1)
		}
		check := widget.NewCheck(fmt.Sprintf("%d. %s", option.Index+1, title), func(checked bool) {
			selected[option.Index] = checked
		})
		check.SetChecked(selected[option.Index])
		channelList.Add(check)
	}

	presetValue := strings.TrimSpace(presetTitle)
	if presetValue == "" {
		presetValue = "Custom"
	}
	modeHint := widget.NewLabel("Replace includes radio settings in the shared payload. Add keeps the receiver's current radio settings.")
	modeHint.Wrapping = fyne.TextWrapWord

	generateButton := widget.NewButton("Generate", nil)
	closeButton := widget.NewButton("Close", nil)

	content := container.NewBorder(
		nil,
		container.NewHBox(statusLabel, layout.NewSpacer(), closeButton, generateButton),
		nil,
		nil,
		container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Mode", modeGroup),
				widget.NewFormItem("LoRa preset", widget.NewLabel(presetValue)),
			),
			modeHint,
			widget.NewSeparator(),
			widget.NewLabel("Channels to share"),
			channelList,
		),
	)

	modal := dialog.NewCustomWithoutButtons("Share channels", content, window)
	closeButton.OnTapped = modal.Hide
	generateButton.OnTapped = func() {
		selectedChannels := make([]meshapp.NodeChannelSettings, 0, len(options))
		for _, option := range options {
			if selected[option.Index] {
				selectedChannels = append(selectedChannels, option.Settings)
			}
		}
		if len(selectedChannels) == 0 {
			statusLabel.SetText("Select at least one channel to share.")

			return
		}

		rawURL, err := meshapp.BuildChannelShareURL(selectedChannels, lora, modeGroup.Selected == "Add")
		if err != nil {
			statusLabel.SetText("Generate failed: " + err.Error())

			return
		}

		modal.Hide()
		showQRCodeShareModal(window, qrShareModalPayload{
			Title: "Share channels QR code",
			URL:   rawURL,
		})
	}

	modal.Resize(fyne.NewSize(520, 460))
	modal.Show()
}

func buildChannelShareOptions(channels []meshapp.NodeChannelSettings, presetTitle string) []channelShareOption {
	options := make([]channelShareOption, 0, len(channels))
	for index, item := range channels {
		options = append(options, channelShareOption{
			Index:    index,
			Title:    nodeChannelDisplayTitle(item, index, presetTitle),
			Settings: cloneNodeChannelSettingsEntry(item),
		})
	}

	return options
}

func initialChannelShareSelection(options []channelShareOption, preferredIndex int) map[int]bool {
	selected := make(map[int]bool, len(options))
	if preferredIndex >= 0 {
		for _, option := range options {
			selected[option.Index] = option.Index == preferredIndex
		}

		return selected
	}

	for _, option := range options {
		selected[option.Index] = true
	}

	return selected
}

func channelIndexFromChatKey(chatKey string) int {
	chatKey = strings.TrimSpace(chatKey)
	if !strings.HasPrefix(chatKey, "channel:") {
		return -1
	}

	value, err := strconv.Atoi(strings.TrimPrefix(chatKey, "channel:"))
	if err != nil {
		return -1
	}

	return value
}
