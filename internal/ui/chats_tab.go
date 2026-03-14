package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
	"github.com/skobkin/meshgo/internal/ui/widgets"
	chatlayout "github.com/skobkin/meshgo/internal/ui/widgets/layout"
)

var chatsLogger = slog.With("component", "ui.chats")

// messageRowMeasureEpsilon filters sub-pixel measurement jitter so list item
// heights are updated only on meaningful size changes.
const messageRowMeasureEpsilon float32 = 0.5

func newChatsTab(
	window fyne.Window,
	store *domain.ChatStore,
	sender MessageSender,
	nodeNameByID func(string) string,
	relayNodeNameByLastByte func(uint32) string,
	localNodeID func() string,
	nodeChanges <-chan struct{},
	initialSelectedKey string,
	openRequests <-chan string,
	onChatSelected func(string),
	onDeleteDMChat func(string) error,
	onShareChannel func(domain.Chat),
) fyne.CanvasObject {
	chats := store.ChatListSorted()
	previewsByKey := chatPreviewByKey(store, chats, nodeNameByID)
	selectedKey := strings.TrimSpace(initialSelectedKey)
	readIncomingUpToByKey := initialReadIncomingByChat(store, chats)
	unreadByKey := chatUnreadByKey(store, chats, readIncomingUpToByKey)
	if selectedKey != "" && len(chats) > 0 && !hasChat(chats, selectedKey) {
		selectedKey = ""
	}
	if selectedKey == "" && len(chats) > 0 {
		selectedKey = chats[0].Key
	}
	chatsLogger.Info(
		"building chats tab",
		"chat_count", len(chats),
		"initial_selected_chat", selectedKey,
	)
	markChatRead(store, readIncomingUpToByKey, selectedKey)
	unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
	messageView := buildChatMessageView(store.Messages(selectedKey), nodeNameByID, localNodeID)
	var messageList *widget.List
	var chatTitle *widget.Label
	var entry *widget.Entry
	var tooltipManager *widgets.HoverTooltipManager
	var replyLabel *widget.Label
	var replyIndicator *fyne.Container
	var sendStatusLabel *widget.Label
	var refreshReplyIndicator func()
	var ensureReplyShortcut func()
	pendingScrollChatKey := ""
	pendingScrollMinCount := 0
	replyToDeviceMessageID := ""
	hoveredReplyTargetDeviceMessageID := ""
	replyShortcutRegistered := false
	pendingRequestedChatKey := ""
	messageItemHeightByID := make(map[widget.ListItemID]float32)
	messageItemWidthByID := make(map[widget.ListItemID]float32)
	clearSelectionOnRefresh := false

	var chatList *widget.List
	chatList = widget.NewList(
		func() int { return len(chats) },
		func() fyne.CanvasObject {
			unreadLabel := widget.NewLabel("●")
			titleLabel := widget.NewLabel("chat")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			typeLabel := widget.NewLabel("type")
			previewLabel := widget.NewLabel("preview")

			return newChatRowItem(container.NewVBox(
				container.NewHBox(unreadLabel, titleLabel, layout.NewSpacer(), typeLabel),
				previewLabel,
			))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(chats) {
				return
			}
			chat := chats[id]
			rowItem, ok := obj.(*chatRowItem)
			if !ok {
				return
			}
			rowItem.onPrimaryTap = func() {
				chatList.Select(id)
			}
			rowItem.onSecondary = func(position fyne.Position) {
				showChatListContextMenu(canvasForObject(rowItem), position, chat, func(selected domain.Chat, action chatListAction) {
					switch action {
					case chatListActionShare:
						if onShareChannel != nil {
							onShareChannel(selected)
						}
					case chatListActionDelete:
						if !domain.IsDMChat(selected) || onDeleteDMChat == nil {
							return
						}
						if window == nil {
							chatsLogger.Warn("delete dm chat failed: active window unavailable", "chat_key", selected.Key)

							return
						}
						title := chatDisplayTitle(selected, nodeNameByID)
						if strings.TrimSpace(title) == "" {
							title = selected.Key
						}
						dialog.ShowConfirm(
							"Delete DM chat?",
							fmt.Sprintf("Delete local DM history for %s from this desktop app?", title),
							func(ok bool) {
								if !ok {
									return
								}
								if selected.Key == selectedKey {
									clearSelectionOnRefresh = true
								}
								if err := onDeleteDMChat(selected.Key); err != nil {
									if selected.Key == selectedKey {
										clearSelectionOnRefresh = false
									}
									chatsLogger.Warn("delete dm chat failed", "chat_key", selected.Key, "error", err)
									dialog.ShowError(err, window)
								}
							},
							window,
						)
					}
				})
			}
			root := rowItem.content.(*fyne.Container)
			line1 := root.Objects[0].(*fyne.Container)
			unreadLabel := line1.Objects[0].(*widget.Label)
			titleLabel := line1.Objects[1].(*widget.Label)
			typeLabel := line1.Objects[3].(*widget.Label)
			previewLabel := root.Objects[1].(*widget.Label)

			unreadLabel.SetText(chatUnreadMarker(unreadByKey[chat.Key]))
			titleLabel.SetText(chatDisplayTitle(chat, nodeNameByID))
			typeLabel.SetText(chatTypeLabel(chat))
			if preview, ok := previewsByKey[chat.Key]; ok {
				previewLabel.SetText(preview)
			} else {
				previewLabel.SetText("No messages yet")
			}
		},
	)
	chatList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(chats) {
			return
		}
		chatsLogger.Debug(
			"chat selected",
			"index", id,
			"chat_key", chats[id].Key,
			"chat_title", chats[id].Title,
		)
		tooltipManager.Hide(nil)
		selectedKey = chats[id].Key
		markChatRead(store, readIncomingUpToByKey, selectedKey)
		unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
		if onChatSelected != nil {
			onChatSelected(selectedKey)
		}
		messageView = buildChatMessageView(store.Messages(selectedKey), nodeNameByID, localNodeID)
		replyToDeviceMessageID = ""
		hoveredReplyTargetDeviceMessageID = ""
		clear(messageItemHeightByID)
		clear(messageItemWidthByID)
		refreshReplyIndicator()
		chatList.Refresh()
		messageList.Refresh()
		chatTitle.SetText(chatDisplayTitle(chats[id], nodeNameByID))
		scrollMessageListToEnd(messageList, len(messageView.Timeline))
		ensureReplyShortcut()
		focusEntry(entry)
	}

	chatTitle = widget.NewLabel("No chat selected")
	if selectedKey != "" {
		chatTitle.SetText(chatTitleByKey(chats, selectedKey, nodeNameByID))
	}
	statusTooltips := newMessageStatusTooltipCache()
	tooltipLayer := container.NewWithoutLayout()
	tooltipManager = widgets.NewHoverTooltipManager(tooltipLayer)

	refreshReplyIndicator = func() {
		if replyIndicator == nil || replyLabel == nil {
			return
		}
		replyID := strings.TrimSpace(replyToDeviceMessageID)
		if replyID == "" {
			replyIndicator.Hide()
			replyLabel.SetText("")
			replyIndicator.Refresh()

			return
		}
		replyLabel.SetText("Replying to message: original message unavailable")
		if original, ok := messageView.ByDeviceID[replyID]; ok {
			meta, hasMeta := parseMessageMeta(original.MetaJSON)
			sender, _, hasSender := messageTextParts(*original, meta, hasMeta, nodeNameByID, localNodeID)
			body := compactWhitespace(original.Body)
			if body == "" {
				body = "(empty)"
			}
			if hasSender {
				replyLabel.SetText(fmt.Sprintf("Replying to %s: %s", sender, body))
			} else {
				replyLabel.SetText(fmt.Sprintf("Replying to message: %s", body))
			}
		}
		replyIndicator.Show()
		replyIndicator.Refresh()
	}

	clearReplyTarget := func() {
		replyToDeviceMessageID = ""
		refreshReplyIndicator()
	}

	setReplyTarget := func(message *domain.ChatMessage) bool {
		if message == nil || !canReplyToMessage(*message) {
			if sendStatusLabel != nil {
				sendStatusLabel.SetText("Reply unavailable for this message")
			}

			return false
		}
		replyToDeviceMessageID = strings.TrimSpace(message.DeviceMessageID)
		refreshReplyIndicator()
		focusEntry(entry)

		return true
	}

	replyFromShortcut := func() {
		if entry == nil || !entry.Visible() {
			return
		}
		if hoveredID := strings.TrimSpace(hoveredReplyTargetDeviceMessageID); hoveredID != "" {
			if hovered, ok := messageView.ByDeviceID[hoveredID]; ok {
				if setReplyTarget(hovered) {
					return
				}
			}
		}
		if latest := latestReplyTarget(messageView.Timeline); latest != nil {
			_ = setReplyTarget(latest)
		}
	}

	ensureReplyShortcut = func() {
		if replyShortcutRegistered || entry == nil {
			return
		}
		fyneCanvas := canvasForObject(entry)
		if fyneCanvas == nil {
			return
		}
		shortcut := &desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierControl}
		fyneCanvas.AddShortcut(shortcut, func(fyne.Shortcut) {
			replyFromShortcut()
		})
		replyShortcutRegistered = true
	}

	messageList = widget.NewList(
		func() int { return len(messageView.Timeline) },
		func() fyne.CanvasObject {
			quoteText := widget.NewRichTextWithText("")
			quoteText.Wrapping = fyne.TextWrapWord
			quoteBar := canvas.NewRectangle(color.NRGBA{R: 120, G: 120, B: 120, A: 255})
			quoteBar.SetMinSize(fyne.NewSize(3, 1))
			quoteLine := container.NewBorder(
				nil,
				nil,
				container.NewHBox(
					quoteBar,
					horizontalSpacer(theme.Padding()/2),
				),
				nil,
				quoteText,
			)
			quoteLine.Hide()

			transportBadge := widgets.NewTooltipLabel("", "", tooltipManager)
			messageText := widget.NewRichTextWithText("message")
			messageText.Wrapping = fyne.TextWrapWord
			messageLine := container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(horizontalSpacer(theme.Padding()), transportBadge, horizontalSpacer(theme.Padding())),
				messageText,
			)
			metaParts := container.NewHBox(widget.NewRichTextWithText("meta"))
			statusBadge := widgets.NewTooltipLabel("", "", tooltipManager)
			timeLabel := widget.NewLabel("time")
			reactionsRow := container.NewHBox()
			reactionsRow.Hide()
			row := container.NewVBox(
				quoteLine,
				messageLine,
				container.NewHBox(
					metaParts,
					layout.NewSpacer(),
					container.NewHBox(statusBadge, horizontalSpacer(theme.Padding()/2), timeLabel, horizontalSpacer(0)),
				),
				reactionsRow,
			)
			bubbleBg := canvas.NewRectangle(chatBubbleFillColor(domain.MessageDirectionIn))
			bubbleBg.CornerRadius = 10
			bubble := container.NewStack(bubbleBg, container.NewPadded(row))

			return newChatMessageRowItem(container.New(chatlayout.NewChatRowLayout(false), bubble))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(messageView.Timeline) {
				return
			}
			msg := messageView.Timeline[id]
			meta, hasMeta := parseMessageMeta(msg.MetaJSON)
			rowItem, ok := obj.(*chatMessageRowItem)
			if !ok {
				return
			}
			message := msg
			rowItem.onSecondary = func(position fyne.Position) {
				showChatMessageContextMenu(canvasForObject(rowItem), position, message, func(message domain.ChatMessage, action ChatAction) {
					if action != ChatActionReply {
						return
					}
					_ = setReplyTarget(&message)
				})
			}
			rowItem.onHoverChange = func(hovered bool) {
				if hovered {
					if canReplyToMessage(message) {
						hoveredReplyTargetDeviceMessageID = message.DeviceMessageID
					}

					return
				}
				if hoveredReplyTargetDeviceMessageID == message.DeviceMessageID {
					hoveredReplyTargetDeviceMessageID = ""
				}
			}
			rowContainer, ok := rowItem.content.(*fyne.Container)
			if !ok {
				return
			}
			rowLayout, ok := rowContainer.Layout.(*chatlayout.ChatRowLayout)
			if ok {
				rowLayout.SetAlignRight(msg.Direction == domain.MessageDirectionOut)
			}
			bubble := rowContainer.Objects[0].(*fyne.Container)
			bubbleBg := bubble.Objects[0].(*canvas.Rectangle)
			bubbleBg.FillColor = chatBubbleFillColor(msg.Direction)
			bubbleBg.Refresh()
			box := bubble.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
			quoteLine := box.Objects[0].(*fyne.Container)
			quoteText := quoteLine.Objects[0].(*widget.RichText)
			replyID := strings.TrimSpace(msg.ReplyToDeviceMessageID)
			if replyID != "" {
				if original, ok := messageView.ByDeviceID[replyID]; ok {
					quoteText.Segments = quoteSegments(*original, nodeNameByID, localNodeID)
				} else {
					quoteText.Segments = []widget.RichTextSegment{
						&widget.TextSegment{Text: "Original message unavailable", Style: widget.RichTextStyleInline},
					}
				}
				quoteText.Refresh()
				quoteLine.Show()
			} else {
				quoteLine.Hide()
			}
			messageLine := box.Objects[1].(*fyne.Container)
			messageText := messageLine.Objects[0].(*widget.RichText)
			transportSlot := messageLine.Objects[1].(*fyne.Container)
			transportBadge := transportSlot.Objects[1].(*widgets.TooltipWidget)
			messageText.Segments = messageTextSegments(msg, meta, hasMeta, nodeNameByID, localNodeID)
			messageText.Wrapping = fyne.TextWrapWord
			messageText.Refresh()
			transportBadge.SetBadge(messageTransportBadge(msg, meta, hasMeta))
			metaRow := box.Objects[2].(*fyne.Container)
			metaParts := metaRow.Objects[0].(*fyne.Container)
			widgets.HideTooltipWidgets(metaParts.Objects)
			metaParts.Objects = messageMetaWidgets(msg, meta, hasMeta, nodeNameByID, relayNodeNameByLastByte, tooltipManager)
			metaParts.Refresh()
			metaRight := metaRow.Objects[2].(*fyne.Container)
			statusBadge := metaRight.Objects[0].(*widgets.TooltipWidget)
			statusText, statusTooltip := messageStatusBadge(msg)
			if statusTooltipContent := messageStatusTooltipContent(msg, statusTooltips); statusTooltipContent != nil {
				statusBadge.SetBadgeWithContent(statusText, statusTooltipContent)
			} else {
				statusBadge.SetBadge(statusText, statusTooltip)
			}
			metaRight.Objects[2].(*widget.Label).SetText(messageTimeLabel(msg.At))
			reactionsRow := box.Objects[3].(*fyne.Container)
			widgets.HideTooltipWidgets(reactionsRow.Objects)
			reactionsRow.Objects = messageReactionWidgets(
				msg.DeviceMessageID,
				messageView.ReactionsByTargetDeviceID,
				tooltipManager,
			)
			if len(reactionsRow.Objects) == 0 {
				reactionsRow.Hide()
			} else {
				reactionsRow.Show()
				reactionsRow.Refresh()
			}

			// During early startup refreshes list width can still be zero. Skip
			// height caching in that state to avoid overestimating wrapped row height.
			if messageList.Size().Width <= 1 || rowContainer.Size().Width <= 1 {
				rowContainer.Refresh()

				return
			}

			// Chat rows can have different heights (e.g. multiline message text or
			// reaction rows). Update item height when meaningful change is observed.
			rowHeight := rowContainer.MinSize().Height
			rowWidth := rowContainer.Size().Width
			prevHeight, hasPrev := messageItemHeightByID[id]
			prevWidth := messageItemWidthByID[id]
			if shouldUpdateMessageItemHeight(hasPrev, prevHeight, prevWidth, rowHeight, rowWidth) {
				messageItemHeightByID[id] = rowHeight
				messageItemWidthByID[id] = rowWidth
				messageList.SetItemHeight(id, rowHeight)
			}
			rowContainer.Refresh()
		},
	)

	entry = widget.NewEntry()
	entry.SetPlaceHolder("Type message (max 200 bytes)")
	counterLabel := widget.NewLabel("0/200 bytes")
	sendStatusLabel = widget.NewLabel("")
	sendStatusLabel.Truncation = fyne.TextTruncateEllipsis
	sendButton := widget.NewButton("Send", nil)
	replyLabel = widget.NewLabel("")
	replyLabel.Truncation = fyne.TextTruncateEllipsis
	replyCancelButton := widget.NewButton("Cancel", func() {
		clearReplyTarget()
		focusEntry(entry)
	})
	replyIndicator = container.NewBorder(
		nil,
		nil,
		nil,
		replyCancelButton,
		replyLabel,
	)
	replyIndicator.Hide()

	updateCounter := func(text string) {
		count := len([]byte(text))
		counterLabel.SetText(fmt.Sprintf("%d/200 bytes", count))
	}
	entry.OnChanged = updateCounter
	isSending := false

	applyComposerState := func() {
		canSend := !isSending && selectedKey != "" && sender != nil
		if canSend {
			entry.Enable()
			sendButton.Enable()
		} else {
			entry.Disable()
			sendButton.Disable()
		}
	}

	setSending := func(inFlight bool) {
		isSending = inFlight
		applyComposerState()
	}

	sendCurrent := func() {
		text := strings.TrimSpace(entry.Text)
		if selectedKey == "" {
			chatsLogger.Debug("send ignored: no selected chat")

			return
		}
		if text == "" {
			chatsLogger.Debug("send ignored: empty message text")

			return
		}
		if len([]byte(text)) > 200 {
			chatsLogger.Info("send blocked: message exceeds 200 bytes", "chat_key", selectedKey, "bytes", len([]byte(text)))

			return
		}

		chatsLogger.Info("sending chat message", "chat_key", selectedKey, "bytes", len([]byte(text)))
		pendingScrollChatKey = selectedKey
		pendingScrollMinCount = len(messageView.Timeline) + 1
		sendStatusLabel.SetText("")
		opts := radio.TextSendOptions{ReplyToDeviceMessageID: strings.TrimSpace(replyToDeviceMessageID)}
		setSending(true)
		go func(chatKey, body string, sendOpts radio.TextSendOptions) {
			res := <-sender.SendText(chatKey, body, sendOpts)
			if res.Err != nil {
				fyne.Do(func() {
					chatsLogger.Warn("chat message send failed", "chat_key", chatKey, "bytes", len([]byte(body)), "error", res.Err)
					if pendingScrollChatKey == chatKey {
						pendingScrollChatKey = ""
						pendingScrollMinCount = 0
					}
					sendStatusLabel.SetText("Send failed: " + res.Err.Error())
					setSending(false)
				})

				return
			}
			fyne.Do(func() {
				chatsLogger.Info("chat message sent", "chat_key", chatKey, "bytes", len([]byte(body)))
				sendStatusLabel.SetText("")
				entry.SetText("")
				clearReplyTarget()
				setSending(false)
			})
		}(selectedKey, text, opts)
	}

	entry.OnSubmitted = func(_ string) { sendCurrent() }
	sendButton.OnTapped = sendCurrent

	composer := container.NewBorder(nil, nil, nil, sendButton, entry)
	composerStatusRow := container.NewHBox(counterLabel, layout.NewSpacer(), sendStatusLabel)
	right := container.NewBorder(
		chatTitle,
		container.NewVBox(replyIndicator, composerStatusRow, composer),
		nil,
		nil,
		messageList,
	)

	split := container.NewHSplit(
		container.NewBorder(nil, nil, nil, nil, chatList),
		right,
	)
	split.Offset = 0.32

	var refreshFromStore func()
	openRequestedChat := func(chatKey string) {
		requested := strings.TrimSpace(chatKey)
		if requested == "" {
			return
		}
		pendingRequestedChatKey = requested
		if selectedIndex := chatIndexByKey(chats, requested); selectedIndex >= 0 {
			chatList.Select(selectedIndex)
			pendingRequestedChatKey = ""
			focusEntry(entry)

			return
		}
		if refreshFromStore != nil {
			refreshFromStore()
		}
	}

	refreshFromStore = func() {
		chatsLogger.Debug(
			"refreshing chats tab from store",
			"selected_chat", selectedKey,
			"chat_count", len(chats),
		)
		updatedChats := store.ChatListSorted()
		nextSelectedKey := selectedKey
		requestedChatKey := strings.TrimSpace(pendingRequestedChatKey)
		switch {
		case requestedChatKey != "" && hasChat(updatedChats, requestedChatKey):
			nextSelectedKey = requestedChatKey
			pendingRequestedChatKey = ""
		case clearSelectionOnRefresh:
			nextSelectedKey = ""
			clearSelectionOnRefresh = false
		case nextSelectedKey == "" && len(updatedChats) > 0:
			nextSelectedKey = updatedChats[0].Key
		}
		if nextSelectedKey != "" && !hasChat(updatedChats, nextSelectedKey) {
			wasSelectedDM := domain.IsDMKey(nextSelectedKey)
			nextSelectedKey = ""
			if len(updatedChats) > 0 && !clearSelectionOnRefresh && !wasSelectedDM {
				nextSelectedKey = updatedChats[0].Key
			}
			clearSelectionOnRefresh = false
		}
		updatedView := buildChatMessageView(store.Messages(nextSelectedKey), nodeNameByID, localNodeID)
		if slices.Equal(chats, updatedChats) &&
			nextSelectedKey == selectedKey &&
			slices.Equal(messageView.Timeline, updatedView.Timeline) &&
			reactionMapEqual(messageView.ReactionsByTargetDeviceID, updatedView.ReactionsByTargetDeviceID) {
			chatsLogger.Debug(
				"skipping chat refresh: store snapshot unchanged",
				"selected_chat", selectedKey,
				"chat_count", len(chats),
			)

			return
		}

		tooltipManager.Hide(nil)
		chats = updatedChats
		pruneReadIncomingByChat(readIncomingUpToByKey, chats)
		previewsByKey = chatPreviewByKey(store, chats, nodeNameByID)
		if nextSelectedKey != selectedKey {
			replyToDeviceMessageID = ""
			hoveredReplyTargetDeviceMessageID = ""
		}
		selectedKey = nextSelectedKey
		selectedIndex := chatIndexByKey(chats, selectedKey)
		messageView = updatedView
		clear(messageItemHeightByID)
		clear(messageItemWidthByID)
		markChatRead(store, readIncomingUpToByKey, selectedKey)
		unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
		if selectedKey == "" {
			chatTitle.SetText("No chat selected")
			entry.SetText("")
			sendStatusLabel.SetText("")
			pendingScrollChatKey = ""
			pendingScrollMinCount = 0
		} else {
			chatTitle.SetText(chatTitleByKey(chats, selectedKey, nodeNameByID))
		}
		if replyToDeviceMessageID != "" {
			if _, ok := messageView.ByDeviceID[replyToDeviceMessageID]; !ok {
				replyToDeviceMessageID = ""
			}
		}
		refreshReplyIndicator()
		applyComposerState()
		chatList.Refresh()
		messageList.Refresh()
		ensureReplyShortcut()
		if selectedIndex >= 0 {
			chatList.Select(selectedIndex)
		} else {
			chatList.UnselectAll()
		}
		if pendingScrollChatKey != "" &&
			selectedKey == pendingScrollChatKey &&
			len(messageView.Timeline) >= pendingScrollMinCount {
			chatsLogger.Debug(
				"auto-scrolling to latest message after send",
				"chat_key", selectedKey,
				"message_count", len(messageView.Timeline),
			)
			scrollMessageListToEnd(messageList, len(messageView.Timeline))
			pendingScrollChatKey = ""
			pendingScrollMinCount = 0
		}
	}

	if selectedIndex := chatIndexByKey(chats, selectedKey); selectedIndex >= 0 {
		chatList.Select(selectedIndex)
		fyne.Do(func() {
			refreshReplyIndicator()
			applyComposerState()
			ensureReplyShortcut()
			messageList.Refresh()
		})
	} else if len(chats) > 0 {
		chatList.Select(0)
		fyne.Do(func() {
			refreshReplyIndicator()
			applyComposerState()
			ensureReplyShortcut()
			messageList.Refresh()
		})
	} else {
		fyne.Do(func() {
			applyComposerState()
			refreshReplyIndicator()
			messageList.Refresh()
		})
	}

	chatsLogger.Debug("starting chat store change listener")
	go func() {
		for range store.Changes() {
			fyne.Do(func() {
				refreshFromStore()
			})
		}
	}()
	if nodeChanges != nil {
		chatsLogger.Debug("starting node change listener for chat labels")
		go func() {
			for range nodeChanges {
				fyne.Do(func() {
					tooltipManager.Hide(nil)
					previewsByKey = chatPreviewByKey(store, chats, nodeNameByID)
					chatTitle.SetText(chatTitleByKey(chats, selectedKey, nodeNameByID))
					refreshReplyIndicator()
					chatList.Refresh()
					messageList.Refresh()
				})
			}
		}()
	}
	if openRequests != nil {
		chatsLogger.Debug("starting chat open request listener")
		go func() {
			for chatKey := range openRequests {
				requested := chatKey
				fyne.Do(func() {
					openRequestedChat(requested)
				})
			}
		}()
	}

	return container.New(layout.NewStackLayout(), split, tooltipLayer)
}

func focusEntry(entry *widget.Entry) {
	if entry == nil {
		return
	}
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	fyneCanvas := app.Driver().CanvasForObject(entry)
	if fyneCanvas == nil {
		return
	}
	fyneCanvas.Focus(entry)
}

func scrollMessageListToEnd(list *widget.List, length int) {
	if list == nil || length <= 0 {
		return
	}
	list.ScrollTo(length - 1)
	list.ScrollToBottom()
}

func shouldUpdateMessageItemHeight(hasPrev bool, prevHeight, prevWidth, rowHeight, rowWidth float32) bool {
	if !hasPrev {
		return true
	}
	if rowHeight > prevHeight+messageRowMeasureEpsilon {
		return true
	}
	if rowHeight < prevHeight-messageRowMeasureEpsilon {
		if rowWidth > prevWidth+messageRowMeasureEpsilon {
			return true
		}
	}

	return false
}

func chatUnreadMarker(hasUnread bool) string {
	if hasUnread {
		return "●"
	}

	return " "
}

func chatPreviewByKey(store *domain.ChatStore, chats []domain.Chat, nodeNameByID func(string) string) map[string]string {
	previews := make(map[string]string, len(chats))
	for _, chat := range chats {
		previews[chat.Key] = chatPreviewLine(store.Messages(chat.Key), nodeNameByID)
	}

	return previews
}

func chatPreviewLine(messages []domain.ChatMessage, nodeNameByID func(string) string) string {
	last, ok := latestNonReactionMessage(messages)
	if !ok {
		return "No messages yet"
	}
	body := compactWhitespace(last.Body)
	if body == "" {
		body = "(empty)"
	}

	return truncatePreview(fmt.Sprintf("%s: %s", previewSender(last, nodeNameByID), body), 72)
}

func previewSender(msg domain.ChatMessage, nodeNameByID func(string) string) string {
	if msg.Direction == domain.MessageDirectionOut {
		return "you"
	}
	meta, hasMeta := parseMessageMeta(msg.MetaJSON)
	if !hasMeta {
		return "someone"
	}
	if sender := domain.NormalizeNodeID(meta.From); sender != "" {
		return displaySender(sender, nodeNameByID)
	}

	return "someone"
}

func compactWhitespace(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

func truncatePreview(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	if limit <= 3 {
		return string(runes[:limit])
	}

	return string(runes[:limit-3]) + "..."
}

type messageMeta struct {
	From      string   `json:"from"`
	To        string   `json:"to"`
	ReplyID   *uint32  `json:"reply_id"`
	Emoji     *uint32  `json:"emoji"`
	Hops      *int     `json:"hops"`
	HopStart  *uint32  `json:"hop_start"`
	HopLimit  *uint32  `json:"hop_limit"`
	RelayNode *uint32  `json:"relay_node"`
	RxRSSI    *int     `json:"rx_rssi"`
	RxSNR     *float64 `json:"rx_snr"`
	ViaMQTT   bool     `json:"via_mqtt"`
	Transport string   `json:"transport"`
}

func parseMessageMeta(raw string) (messageMeta, bool) {
	if strings.TrimSpace(raw) == "" {
		return messageMeta{}, false
	}
	var meta messageMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return messageMeta{}, false
	}

	return meta, true
}

func messageTextLine(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) string {
	sender, body, hasSender := messageTextParts(m, meta, hasMeta, nodeNameByID, localNodeID)
	if hasSender {
		return fmt.Sprintf("%s: %s", sender, body)
	}

	return body
}

func messageTextParts(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) (sender, body string, hasSender bool) {
	if m.Direction == domain.MessageDirectionOut {
		if hasMeta {
			if localID := domain.NormalizeNodeID(meta.From); localID != "" {
				return displaySender(localID, nodeNameByID), m.Body, true
			}
		}
		if localNodeID != nil {
			if localID := domain.NormalizeNodeID(localNodeID()); localID != "" {
				return displaySender(localID, nodeNameByID), m.Body, true
			}
		}

		return "you", m.Body, true
	}
	if hasMeta {
		if sender := domain.NormalizeNodeID(meta.From); sender != "" {
			return displaySender(sender, nodeNameByID), m.Body, true
		}
	}

	return "", m.Body, false
}

func messageTextSegments(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) []widget.RichTextSegment {
	sender, body, hasSender := messageTextParts(m, meta, hasMeta, nodeNameByID, localNodeID)
	if hasSender {
		return []widget.RichTextSegment{
			&widget.TextSegment{Text: sender, Style: widget.RichTextStyleStrong},
			&widget.TextSegment{Text: ": " + body, Style: widget.RichTextStyleInline},
		}
	}

	return []widget.RichTextSegment{
		&widget.TextSegment{Text: body, Style: widget.RichTextStyleInline},
	}
}

func messageMetaLine(m domain.ChatMessage, meta messageMeta, hasMeta bool) string {
	chunks := messageMetaChunks(m, meta, hasMeta)
	var b strings.Builder
	for i, chunk := range chunks {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(chunk.PlainText)
	}

	return b.String()
}

func messageMetaSegments(m domain.ChatMessage, meta messageMeta, hasMeta bool) []widget.RichTextSegment {
	chunks := messageMetaChunks(m, meta, hasMeta)
	segments := make([]widget.RichTextSegment, 0, len(chunks)*2)
	for i, chunk := range chunks {
		if i > 0 {
			segments = append(segments, &widget.TextSegment{Text: " ", Style: widget.RichTextStyleInline})
		}
		segments = append(segments, chunk.Segments...)
	}

	return segments
}

func messageMetaWidgets(
	m domain.ChatMessage,
	meta messageMeta,
	hasMeta bool,
	nodeNameByID func(string) string,
	relayNodeNameByLastByte func(uint32) string,
	tooltipManager *widgets.HoverTooltipManager,
) []fyne.CanvasObject {
	chunks := messageMetaChunksWithContext(m, meta, hasMeta, nodeNameByID, relayNodeNameByLastByte)
	widgetsList := make([]fyne.CanvasObject, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Tooltip) > 0 {
			widgetsList = append(widgetsList, widgets.NewTooltipRichText(chunk.Segments, chunk.Tooltip, tooltipManager))

			continue
		}
		widgetsList = append(widgetsList, widget.NewRichText(chunk.Segments...))
	}

	return widgetsList
}

var hopBadges = [...]string{
	"⓪", "①", "②", "③", "④", "⑤", "⑥", "⑦",
}

type messageMetaChunk struct {
	PlainText string
	Segments  []widget.RichTextSegment
	Tooltip   []widget.RichTextSegment
}

func newMetaChunkInline(text string) messageMetaChunk {
	return messageMetaChunk{
		PlainText: text,
		Segments:  []widget.RichTextSegment{&widget.TextSegment{Text: text, Style: widget.RichTextStyleInline}},
	}
}

func messageMetaChunks(m domain.ChatMessage, meta messageMeta, hasMeta bool) []messageMetaChunk {
	return messageMetaChunksWithContext(m, meta, hasMeta, nil, nil)
}

func messageMetaChunksWithContext(
	m domain.ChatMessage,
	meta messageMeta,
	hasMeta bool,
	nodeNameByID func(string) string,
	relayNodeNameByLastByte func(uint32) string,
) []messageMetaChunk {
	hops, hopsKnown := messageHops(meta, hasMeta)
	parts := make([]messageMetaChunk, 0, 2)
	if hopsKnown && hops > 0 {
		hopChunk := newMetaChunkInline(hopBadge(hops))
		if m.Direction == domain.MessageDirectionIn {
			hopChunk.Tooltip = hopTooltipSegments(hops, meta, hasMeta, nodeNameByID, relayNodeNameByLastByte)
		}
		parts = append(parts, hopChunk)
	}
	if isMessageFromMQTT(meta, hasMeta) {
		return parts
	}

	if m.Direction == domain.MessageDirectionIn && hopsKnown && hops == 0 {
		if quality, ok := signalQualityFromMetrics(meta.RxRSSI, meta.RxSNR); ok {
			bars := signalBarsForQuality(quality)
			parts = append(parts, messageMetaChunk{
				PlainText: bars,
				Segments: []widget.RichTextSegment{
					&widget.TextSegment{
						Text:  bars,
						Style: signalRichTextStyle(signalThemeColorForQuality(quality), true),
					},
				},
				Tooltip: signalTooltipSegments(meta),
			})
		}
	}

	return parts
}

func hopTooltipSegments(
	hops int,
	meta messageMeta,
	hasMeta bool,
	nodeNameByID func(string) string,
	relayNodeNameByLastByte func(uint32) string,
) []widget.RichTextSegment {
	if hops < 0 {
		return nil
	}

	lines := []string{fmt.Sprintf("Hops: %d", hops)}
	if relay := resolveRelayNodeDisplay(meta, hasMeta, nodeNameByID, relayNodeNameByLastByte); relay != "" {
		lines = append(lines, fmt.Sprintf("Received from: %s (last relay node)", relay))
	}
	if isMessageFromMQTT(meta, hasMeta) {
		lines = append(lines, "MQTT involved")
	}

	return []widget.RichTextSegment{
		&widget.TextSegment{Text: strings.Join(lines, "\n"), Style: widget.RichTextStyleInline},
	}
}

func signalTooltipSegments(meta messageMeta) []widget.RichTextSegment {
	segments := make([]widget.RichTextSegment, 0, 6)
	if meta.RxRSSI != nil {
		rssiText := fmt.Sprintf("%d", *meta.RxRSSI)
		segments = append(segments,
			&widget.TextSegment{Text: "RSSI: ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{
				Text:  rssiText,
				Style: signalRichTextStyle(signalThemeColorForRSSI(*meta.RxRSSI), false),
			},
		)
	}
	if meta.RxSNR != nil {
		if len(segments) > 0 {
			segments = append(segments, &widget.TextSegment{Text: " ", Style: widget.RichTextStyleInline})
		}
		snrText := fmt.Sprintf("%.2f", *meta.RxSNR)
		segments = append(segments,
			&widget.TextSegment{Text: "SNR: ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{
				Text:  snrText,
				Style: signalRichTextStyle(signalThemeColorForSNR(*meta.RxSNR), false),
			},
		)
	}

	return segments
}

func hopBadge(hops int) string {
	if hops < 0 {
		return "?"
	}
	if hops < len(hopBadges) {
		return hopBadges[hops]
	}

	return fmt.Sprintf("h%d", hops)
}

func signalRichTextStyle(colorName fyne.ThemeColorName, monospace bool) widget.RichTextStyle {
	style := widget.RichTextStyleInline
	style.ColorName = colorName
	if monospace {
		style.TextStyle.Monospace = true
	}

	return style
}

func messageStatusBadge(m domain.ChatMessage) (text, tooltip string) {
	if m.Direction != domain.MessageDirectionOut {
		return "", ""
	}
	switch m.Status {
	case domain.MessageStatusPending:
		return "◷", messageStatusPendingTooltipText
	case domain.MessageStatusSent:
		if domain.IsDMKey(m.ChatKey) {
			return "✓", messageStatusSentDMTooltipText
		}

		return "✓", messageStatusSentChannelTooltipText
	case domain.MessageStatusAcked:
		if domain.IsDMKey(m.ChatKey) {
			return "✓✓", messageStatusAckedDMTooltipText
		}

		return "✓✓", messageStatusAckedChannelTooltipText
	case domain.MessageStatusFailed:
		reason := compactWhitespace(strings.TrimSpace(m.StatusReason))
		if reason == "" {
			return "⚠", messageStatusFailedTooltipText
		}

		return "⚠", fmt.Sprintf("%s\nReason: %s.", messageStatusFailedTooltipText, reason)
	default:
		return "", ""
	}
}

type messageStatusTooltipCache struct {
	pending       fyne.CanvasObject
	sentChannel   fyne.CanvasObject
	sentDM        fyne.CanvasObject
	ackedChannel  fyne.CanvasObject
	ackedDM       fyne.CanvasObject
	failedGeneric fyne.CanvasObject
}

const (
	messageStatusPendingTooltipText = `Sent from PC to device.
Waiting for mesh confirmation.`
	messageStatusSentChannelTooltipText = `Sent from PC to device.
Transmitted over radio.
Heard by at least one neighbor node.`
	messageStatusSentDMTooltipText = `Sent from PC to device.
Transmitted over radio.
Relayed in mesh; waiting target ack.`
	messageStatusAckedChannelTooltipText = `Sent from PC to device.
Transmitted over radio.
Mesh ack received.`
	messageStatusAckedDMTooltipText = `Sent from PC to device.
Transmitted over radio.
Delivered to target node.`
	messageStatusFailedTooltipText = `Sent from PC to device.
Transmission or delivery failed.`
)

func newMessageStatusTooltipCache() messageStatusTooltipCache {
	return messageStatusTooltipCache{
		pending:       widget.NewLabel(messageStatusPendingTooltipText),
		sentChannel:   widget.NewLabel(messageStatusSentChannelTooltipText),
		sentDM:        widget.NewLabel(messageStatusSentDMTooltipText),
		ackedChannel:  widget.NewLabel(messageStatusAckedChannelTooltipText),
		ackedDM:       widget.NewLabel(messageStatusAckedDMTooltipText),
		failedGeneric: widget.NewLabel(messageStatusFailedTooltipText),
	}
}

func messageStatusTooltipContent(m domain.ChatMessage, cache messageStatusTooltipCache) fyne.CanvasObject {
	if m.Direction != domain.MessageDirectionOut {
		return nil
	}

	switch m.Status {
	case domain.MessageStatusPending:
		return cache.pending
	case domain.MessageStatusSent:
		if domain.IsDMKey(m.ChatKey) {
			return cache.sentDM
		}

		return cache.sentChannel
	case domain.MessageStatusAcked:
		if domain.IsDMKey(m.ChatKey) {
			return cache.ackedDM
		}

		return cache.ackedChannel
	case domain.MessageStatusFailed:
		if strings.TrimSpace(m.StatusReason) == "" {
			return cache.failedGeneric
		}

		return nil
	default:
		return nil
	}
}

func messageHops(meta messageMeta, hasMeta bool) (int, bool) {
	if !hasMeta {
		return 0, false
	}
	if meta.Hops != nil {
		return *meta.Hops, true
	}
	if meta.HopStart == nil || meta.HopLimit == nil {
		return 0, false
	}
	if *meta.HopStart == 0 && *meta.HopLimit == 0 {
		return 0, false
	}
	if *meta.HopStart < *meta.HopLimit {
		return 0, false
	}

	return int(*meta.HopStart - *meta.HopLimit), true
}

func isMessageFromMQTT(meta messageMeta, hasMeta bool) bool {
	if !hasMeta {
		return false
	}
	if meta.ViaMQTT {
		return true
	}

	return strings.EqualFold(strings.TrimSpace(meta.Transport), "TRANSPORT_MQTT")
}

func messageTransportBadge(m domain.ChatMessage, meta messageMeta, hasMeta bool) (text, tooltip string) {
	if m.Direction != domain.MessageDirectionIn {
		return "", ""
	}
	if isMessageFromMQTT(meta, hasMeta) {
		return "☁", "via MQTT"
	}

	return "📡", "via Radio"
}

func messageTimeLabel(at time.Time) string {
	if at.IsZero() {
		return ""
	}

	return at.Local().Format("15:04")
}

func chatBubbleFillColor(direction domain.MessageDirection) color.Color {
	app := fyne.CurrentApp()
	if app == nil {
		if direction == domain.MessageDirectionOut {
			return color.NRGBA{R: 72, G: 92, B: 123, A: 255}
		}

		return color.NRGBA{R: 48, G: 48, B: 48, A: 255}
	}

	th := app.Settings().Theme()
	variant := app.Settings().ThemeVariant()
	base := toNRGBA(th.Color(theme.ColorNameInputBackground, variant))
	incoming := colorWithContrastOffset(base)
	if direction != domain.MessageDirectionOut {
		return incoming
	}

	primary := toNRGBA(th.Color(theme.ColorNamePrimary, variant))
	if isDarkColor(incoming) {
		return mixNRGBA(incoming, primary, 0.25)
	}

	return mixNRGBA(incoming, primary, 0.18)
}

func toNRGBA(c color.Color) color.NRGBA {
	nrgba, ok := color.NRGBAModel.Convert(c).(color.NRGBA)
	if ok {
		return nrgba
	}

	return color.NRGBA{}
}

func mixNRGBA(a, b color.NRGBA, bWeight float32) color.NRGBA {
	if bWeight < 0 {
		bWeight = 0
	}
	if bWeight > 1 {
		bWeight = 1
	}
	aWeight := 1 - bWeight

	return color.NRGBA{
		R: uint8(float32(a.R)*aWeight + float32(b.R)*bWeight),
		G: uint8(float32(a.G)*aWeight + float32(b.G)*bWeight),
		B: uint8(float32(a.B)*aWeight + float32(b.B)*bWeight),
		A: uint8(float32(a.A)*aWeight + float32(b.A)*bWeight),
	}
}

func horizontalSpacer(width float32) fyne.CanvasObject {
	if width < 0 {
		width = 0
	}
	rect := canvas.NewRectangle(color.Transparent)
	rect.SetMinSize(fyne.NewSize(width, 1))

	return rect
}

func colorWithContrastOffset(c color.NRGBA) color.NRGBA {
	if isDarkColor(c) {
		return mixNRGBA(c, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, 0.08)
	}

	return mixNRGBA(c, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, 0.06)
}

func isDarkColor(c color.NRGBA) bool {
	return colorLuma(c) < 0.5
}

func colorLuma(c color.NRGBA) float32 {
	return (0.299*float32(c.R) + 0.587*float32(c.G) + 0.114*float32(c.B)) / 255
}

type chatMessageView struct {
	Timeline                  []domain.ChatMessage
	ByDeviceID                map[string]*domain.ChatMessage
	ReactionsByTargetDeviceID map[string][]reactionChip
}

type reactionChip struct {
	Emoji   string
	Senders []string
}

func buildChatMessageView(
	messages []domain.ChatMessage,
	nodeNameByID func(string) string,
	localNodeID func() string,
) chatMessageView {
	view := chatMessageView{
		Timeline:                  make([]domain.ChatMessage, 0, len(messages)),
		ByDeviceID:                make(map[string]*domain.ChatMessage),
		ReactionsByTargetDeviceID: make(map[string][]reactionChip),
	}
	reactionSenderSetByTargetAndEmoji := make(map[string]map[string]map[string]string)
	reactionEmojiOrderByTarget := make(map[string][]string)
	for _, msg := range messages {
		if isReactionMessage(msg) {
			targetID := strings.TrimSpace(msg.ReplyToDeviceMessageID)
			if targetID == "" {
				continue
			}
			emoji := strings.TrimSpace(msg.Body)
			if emoji == "" {
				continue
			}
			meta, hasMeta := parseMessageMeta(msg.MetaJSON)
			senderKey, senderLabel := reactionSenderKeyAndLabel(msg, meta, hasMeta, nodeNameByID, localNodeID)
			targetMap, ok := reactionSenderSetByTargetAndEmoji[targetID]
			if !ok {
				targetMap = make(map[string]map[string]string)
				reactionSenderSetByTargetAndEmoji[targetID] = targetMap
			}
			senderSet, ok := targetMap[emoji]
			if !ok {
				senderSet = make(map[string]string)
				targetMap[emoji] = senderSet
				reactionEmojiOrderByTarget[targetID] = append(reactionEmojiOrderByTarget[targetID], emoji)
			}
			if senderKey == "" {
				senderKey = senderLabel
			}
			if senderKey == "" {
				senderKey = "unknown"
				senderLabel = "someone"
			}
			senderSet[senderKey] = senderLabel

			continue
		}

		view.Timeline = append(view.Timeline, msg)
	}
	for i := range view.Timeline {
		deviceID := strings.TrimSpace(view.Timeline[i].DeviceMessageID)
		if deviceID == "" {
			continue
		}
		view.ByDeviceID[deviceID] = &view.Timeline[i]
	}
	for targetID, byEmoji := range reactionSenderSetByTargetAndEmoji {
		emojiOrder := reactionEmojiOrderByTarget[targetID]
		chips := make([]reactionChip, 0, len(byEmoji))
		for _, emoji := range emojiOrder {
			senderSet := byEmoji[emoji]
			if len(senderSet) == 0 {
				continue
			}
			senders := make([]string, 0, len(senderSet))
			for _, sender := range senderSet {
				senders = append(senders, sender)
			}
			sort.Strings(senders)
			chips = append(chips, reactionChip{Emoji: emoji, Senders: senders})
		}
		if len(chips) > 0 {
			view.ReactionsByTargetDeviceID[targetID] = chips
		}
	}

	return view
}

func isReactionMessage(message domain.ChatMessage) bool {
	return strings.TrimSpace(message.ReplyToDeviceMessageID) != "" && message.Emoji != 0
}

func canReplyToMessage(message domain.ChatMessage) bool {
	if isReactionMessage(message) {
		return false
	}

	return strings.TrimSpace(message.DeviceMessageID) != ""
}

func latestReplyTarget(messages []domain.ChatMessage) *domain.ChatMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if !canReplyToMessage(messages[i]) {
			continue
		}
		target := messages[i]

		return &target
	}

	return nil
}

func latestNonReactionMessage(messages []domain.ChatMessage) (domain.ChatMessage, bool) {
	for i := len(messages) - 1; i >= 0; i-- {
		if isReactionMessage(messages[i]) {
			continue
		}

		return messages[i], true
	}

	return domain.ChatMessage{}, false
}

func quoteSegments(message domain.ChatMessage, nodeNameByID func(string) string, localNodeID func() string) []widget.RichTextSegment {
	meta, hasMeta := parseMessageMeta(message.MetaJSON)
	sender, body, hasSender := messageTextParts(message, meta, hasMeta, nodeNameByID, localNodeID)
	if !hasSender {
		return []widget.RichTextSegment{
			&widget.TextSegment{Text: body, Style: widget.RichTextStyleInline},
		}
	}

	return []widget.RichTextSegment{
		&widget.TextSegment{Text: sender, Style: widget.RichTextStyleStrong},
		&widget.TextSegment{Text: "\n" + body, Style: widget.RichTextStyleInline},
	}
}

func messageReactionWidgets(
	targetDeviceMessageID string,
	reactionsByTarget map[string][]reactionChip,
	tooltipManager *widgets.HoverTooltipManager,
) []fyne.CanvasObject {
	targetDeviceMessageID = strings.TrimSpace(targetDeviceMessageID)
	if targetDeviceMessageID == "" {
		return nil
	}
	chips := reactionsByTarget[targetDeviceMessageID]
	if len(chips) == 0 {
		return nil
	}
	objects := make([]fyne.CanvasObject, 0, len(chips))
	for _, chip := range chips {
		segments := reactionChipSegments(chip)
		tooltip := []widget.RichTextSegment{
			&widget.TextSegment{
				Text:  strings.Join(chip.Senders, "\n"),
				Style: widget.RichTextStyleInline,
			},
		}
		objects = append(objects, widgets.NewTooltipRichText(segments, tooltip, tooltipManager))
	}

	return objects
}

func reactionChipSegments(chip reactionChip) []widget.RichTextSegment {
	emojiStyle := widget.RichTextStyleInline
	emojiStyle.SizeName = theme.SizeNameHeadingText
	segments := []widget.RichTextSegment{
		&widget.TextSegment{Text: chip.Emoji, Style: emojiStyle},
	}
	if len(chip.Senders) > 1 {
		segments = append(segments,
			&widget.TextSegment{Text: " ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: fmt.Sprintf("%d", len(chip.Senders)), Style: widget.RichTextStyleInline},
		)
	}

	return segments
}

func reactionSenderKeyAndLabel(
	message domain.ChatMessage,
	meta messageMeta,
	hasMeta bool,
	nodeNameByID func(string) string,
	localNodeID func() string,
) (string, string) {
	if hasMeta {
		if sender := domain.NormalizeNodeID(meta.From); sender != "" {
			return sender, displaySender(sender, nodeNameByID)
		}
	}
	if message.Direction == domain.MessageDirectionOut {
		if localNodeID != nil {
			if local := domain.NormalizeNodeID(localNodeID()); local != "" {
				return local, displaySender(local, nodeNameByID)
			}
		}

		return "you", "you"
	}

	return "someone", "someone"
}

func reactionMapEqual(a, b map[string][]reactionChip) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		right, ok := b[key]
		if !ok {
			return false
		}
		if len(left) != len(right) {
			return false
		}
		for i := range left {
			if left[i].Emoji != right[i].Emoji {
				return false
			}
			if !slices.Equal(left[i].Senders, right[i].Senders) {
				return false
			}
		}
	}

	return true
}

func canvasForObject(obj fyne.CanvasObject) fyne.Canvas {
	if obj == nil {
		return nil
	}
	app := fyne.CurrentApp()
	if app == nil {
		return nil
	}

	return app.Driver().CanvasForObject(obj)
}

func displaySender(nodeID string, nodeNameByID func(string) string) string {
	if nodeNameByID == nil {
		return nodeID
	}
	if v := strings.TrimSpace(nodeNameByID(nodeID)); v != "" {
		return v
	}

	return nodeID
}

func hasChat(chats []domain.Chat, key string) bool {
	for _, c := range chats {
		if c.Key == key {
			return true
		}
	}

	return false
}

func chatIndexByKey(chats []domain.Chat, key string) int {
	if key == "" {
		return -1
	}
	for i, c := range chats {
		if c.Key == key {
			return i
		}
	}

	return -1
}

func initialReadIncomingByChat(store *domain.ChatStore, chats []domain.Chat) map[string]time.Time {
	readIncomingUpToByKey := make(map[string]time.Time, len(chats))
	for _, chat := range chats {
		readIncomingUpToByKey[chat.Key] = latestIncomingAt(store.Messages(chat.Key))
	}

	return readIncomingUpToByKey
}

func markChatRead(store *domain.ChatStore, readIncomingUpToByKey map[string]time.Time, chatKey string) {
	chatKey = strings.TrimSpace(chatKey)
	if chatKey == "" {
		return
	}
	readIncomingUpToByKey[chatKey] = latestIncomingAt(store.Messages(chatKey))
}

func pruneReadIncomingByChat(readIncomingUpToByKey map[string]time.Time, chats []domain.Chat) {
	keep := make(map[string]struct{}, len(chats))
	for _, chat := range chats {
		keep[chat.Key] = struct{}{}
	}
	for key := range readIncomingUpToByKey {
		if _, ok := keep[key]; ok {
			continue
		}
		delete(readIncomingUpToByKey, key)
	}
}

func chatUnreadByKey(store *domain.ChatStore, chats []domain.Chat, readIncomingUpToByKey map[string]time.Time) map[string]bool {
	unreadByKey := make(map[string]bool, len(chats))
	for _, chat := range chats {
		latestIncoming := latestIncomingAt(store.Messages(chat.Key))
		lastReadIncoming := readIncomingUpToByKey[chat.Key]
		unreadByKey[chat.Key] = latestIncoming.After(lastReadIncoming)
	}

	return unreadByKey
}

func latestIncomingAt(messages []domain.ChatMessage) time.Time {
	var latest time.Time
	for _, msg := range messages {
		if msg.Direction != domain.MessageDirectionIn {
			continue
		}
		if msg.At.After(latest) {
			latest = msg.At
		}
	}

	return latest
}
