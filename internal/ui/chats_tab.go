package ui

import (
	"encoding/json"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func newChatsTab(store *domain.ChatStore, sender interface {
	SendText(chatKey, text string) <-chan radio.SendResult
}, nodeNameByID func(string) string, localNodeID func() string, nodeChanges <-chan struct{}, initialSelectedKey string, onChatSelected func(string)) fyne.CanvasObject {
	chats := store.ChatListSorted()
	previewsByKey := chatPreviewByKey(store, chats, nodeNameByID)
	selectedKey := strings.TrimSpace(initialSelectedKey)
	if selectedKey != "" && len(chats) > 0 && !hasChat(chats, selectedKey) {
		selectedKey = ""
	}
	if selectedKey == "" && len(chats) > 0 {
		selectedKey = chats[0].Key
	}
	messages := store.Messages(selectedKey)
	var messageList *widget.List
	var chatTitle *widget.Label
	var entry *widget.Entry
	pendingScrollChatKey := ""
	pendingScrollMinCount := 0

	chatList := widget.NewList(
		func() int { return len(chats) },
		func() fyne.CanvasObject {
			titleLabel := widget.NewLabel("chat")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			typeLabel := widget.NewLabel("type")
			previewLabel := widget.NewLabel("preview")
			return container.NewVBox(
				container.NewHBox(titleLabel, layout.NewSpacer(), typeLabel),
				previewLabel,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(chats) {
				return
			}
			chat := chats[id]
			root := obj.(*fyne.Container)
			line1 := root.Objects[0].(*fyne.Container)
			titleLabel := line1.Objects[0].(*widget.Label)
			typeLabel := line1.Objects[2].(*widget.Label)
			previewLabel := root.Objects[1].(*widget.Label)

			titleLabel.SetText(chat.Title)
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
		selectedKey = chats[id].Key
		if onChatSelected != nil {
			onChatSelected(selectedKey)
		}
		messages = store.Messages(selectedKey)
		messageList.Refresh()
		chatTitle.SetText(chats[id].Title)
		scrollMessageListToEnd(messageList, len(messages))
		focusEntry(entry)
	}

	chatTitle = widget.NewLabel("No chat selected")
	if selectedKey != "" {
		chatTitle.SetText(chatTitleByKey(chats, selectedKey))
	}

	messageList = widget.NewList(
		func() int { return len(messages) },
		func() fyne.CanvasObject {
			return container.NewVBox(
				widget.NewRichTextWithText("message"),
				container.NewHBox(widget.NewRichTextWithText("meta"), layout.NewSpacer(), widget.NewLabel("status")),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(messages) {
				return
			}
			msg := messages[id]
			meta, hasMeta := parseMessageMeta(msg.MetaJSON)
			box := obj.(*fyne.Container)
			messageText := box.Objects[0].(*widget.RichText)
			messageText.Segments = messageTextSegments(msg, meta, hasMeta, nodeNameByID, localNodeID)
			messageText.Refresh()
			metaRow := box.Objects[1].(*fyne.Container)
			metaText := metaRow.Objects[0].(*widget.RichText)
			metaText.Segments = messageMetaSegments(msg, meta, hasMeta)
			metaText.Refresh()
			metaRow.Objects[2].(*widget.Label).SetText(messageStatusLine(msg))
		},
	)

	entry = widget.NewEntry()
	entry.SetPlaceHolder("Type message (max 200 bytes)")
	counterLabel := widget.NewLabel("0/200 bytes")
	sendButton := widget.NewButton("Send", nil)

	updateCounter := func(text string) {
		count := len([]byte(text))
		counterLabel.SetText(fmt.Sprintf("%d/200 bytes", count))
	}
	entry.OnChanged = updateCounter

	setSending := func(inFlight bool) {
		if inFlight {
			entry.Disable()
			sendButton.Disable()
			return
		}
		entry.Enable()
		sendButton.Enable()
	}

	sendCurrent := func() {
		text := strings.TrimSpace(entry.Text)
		if selectedKey == "" {
			return
		}
		if text == "" {
			return
		}
		if len([]byte(text)) > 200 {
			return
		}

		pendingScrollChatKey = selectedKey
		pendingScrollMinCount = len(messages) + 1
		setSending(true)
		go func(chatKey, body string) {
			res := <-sender.SendText(chatKey, body)
			if res.Err != nil {
				fyne.Do(func() {
					if pendingScrollChatKey == chatKey {
						pendingScrollChatKey = ""
						pendingScrollMinCount = 0
					}
					setSending(false)
				})
				return
			}
			fyne.Do(func() {
				entry.SetText("")
				setSending(false)
			})
		}(selectedKey, text)
	}

	entry.OnSubmitted = func(_ string) { sendCurrent() }
	sendButton.OnTapped = sendCurrent

	composer := container.NewBorder(nil, nil, nil, sendButton, entry)
	right := container.NewBorder(
		chatTitle,
		container.NewVBox(counterLabel, composer),
		nil,
		nil,
		messageList,
	)

	split := container.NewHSplit(
		container.NewBorder(nil, nil, nil, nil, chatList),
		right,
	)
	split.Offset = 0.32

	refreshFromStore := func() {
		updatedChats := store.ChatListSorted()
		chats = updatedChats
		previewsByKey = chatPreviewByKey(store, chats, nodeNameByID)
		if selectedKey == "" && len(chats) > 0 {
			selectedKey = chats[0].Key
		}
		if selectedKey != "" && !hasChat(chats, selectedKey) {
			selectedKey = ""
			if len(chats) > 0 {
				selectedKey = chats[0].Key
			}
		}
		selectedIndex := chatIndexByKey(chats, selectedKey)
		messages = store.Messages(selectedKey)
		chatTitle.SetText(chatTitleByKey(chats, selectedKey))
		chatList.Refresh()
		messageList.Refresh()
		if selectedIndex >= 0 {
			chatList.Select(selectedIndex)
		} else {
			chatList.UnselectAll()
		}
		if pendingScrollChatKey != "" &&
			selectedKey == pendingScrollChatKey &&
			len(messages) >= pendingScrollMinCount {
			scrollMessageListToEnd(messageList, len(messages))
			pendingScrollChatKey = ""
			pendingScrollMinCount = 0
		}
	}

	go func() {
		for range store.Changes() {
			fyne.Do(func() {
				refreshFromStore()
			})
		}
	}()
	if nodeChanges != nil {
		go func() {
			for range nodeChanges {
				fyne.Do(func() {
					previewsByKey = chatPreviewByKey(store, chats, nodeNameByID)
					chatList.Refresh()
					messageList.Refresh()
				})
			}
		}()
	}

	if selectedIndex := chatIndexByKey(chats, selectedKey); selectedIndex >= 0 {
		chatList.Select(selectedIndex)
	} else if len(chats) > 0 {
		chatList.Select(0)
	}

	return container.New(layout.NewStackLayout(), split)
}

func focusEntry(entry *widget.Entry) {
	if entry == nil {
		return
	}
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	canvas := app.Driver().CanvasForObject(entry)
	if canvas == nil {
		return
	}
	canvas.Focus(entry)
}

func scrollMessageListToEnd(list *widget.List, length int) {
	if list == nil || length <= 0 {
		return
	}
	list.ScrollTo(length - 1)
	list.ScrollToBottom()
}

func chatTypeLabel(c domain.Chat) string {
	if c.Type == domain.ChatTypeDM {
		return "DM"
	}
	return "Channel"
}

func chatPreviewByKey(store *domain.ChatStore, chats []domain.Chat, nodeNameByID func(string) string) map[string]string {
	previews := make(map[string]string, len(chats))
	for _, chat := range chats {
		previews[chat.Key] = chatPreviewLine(store.Messages(chat.Key), nodeNameByID)
	}
	return previews
}

func chatPreviewLine(messages []domain.ChatMessage, nodeNameByID func(string) string) string {
	if len(messages) == 0 {
		return "No messages yet"
	}
	last := messages[len(messages)-1]
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
	if sender := normalizeNodeID(meta.From); sender != "" {
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
	Hops      *int     `json:"hops"`
	HopStart  *uint32  `json:"hop_start"`
	HopLimit  *uint32  `json:"hop_limit"`
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
	prefix, sender, body, hasSender := messageTextParts(m, meta, hasMeta, nodeNameByID, localNodeID)
	if hasSender {
		return fmt.Sprintf("%s %s: %s", prefix, sender, body)
	}
	return fmt.Sprintf("%s %s", prefix, body)
}

func messageTextParts(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) (prefix, sender, body string, hasSender bool) {
	prefix = "<"
	if m.Direction == domain.MessageDirectionOut {
		prefix = ">"
		if hasMeta {
			if localID := normalizeNodeID(meta.From); localID != "" {
				return prefix, displaySender(localID, nodeNameByID), m.Body, true
			}
		}
		if localNodeID != nil {
			if localID := normalizeNodeID(localNodeID()); localID != "" {
				return prefix, displaySender(localID, nodeNameByID), m.Body, true
			}
		}
		return prefix, "you", m.Body, true
	}
	if hasMeta {
		if sender := normalizeNodeID(meta.From); sender != "" {
			return prefix, displaySender(sender, nodeNameByID), m.Body, true
		}
	}
	return prefix, "", m.Body, false
}

func messageTextSegments(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) []widget.RichTextSegment {
	prefix, sender, body, hasSender := messageTextParts(m, meta, hasMeta, nodeNameByID, localNodeID)
	if hasSender {
		return []widget.RichTextSegment{
			&widget.TextSegment{Text: prefix + " ", Style: widget.RichTextStyleInline},
			&widget.TextSegment{Text: sender, Style: widget.RichTextStyleStrong},
			&widget.TextSegment{Text: ": " + body, Style: widget.RichTextStyleInline},
		}
	}
	return []widget.RichTextSegment{
		&widget.TextSegment{Text: prefix + " " + body, Style: widget.RichTextStyleInline},
	}
}

func messageMetaLine(m domain.ChatMessage, meta messageMeta, hasMeta bool) string {
	fragments := messageMetaFragments(m, meta, hasMeta)
	var b strings.Builder
	for _, f := range fragments {
		b.WriteString(f.Text)
	}
	return b.String()
}

func messageMetaSegments(m domain.ChatMessage, meta messageMeta, hasMeta bool) []widget.RichTextSegment {
	fragments := messageMetaFragments(m, meta, hasMeta)
	segments := make([]widget.RichTextSegment, 0, len(fragments))
	for _, f := range fragments {
		segments = append(segments, &widget.TextSegment{Text: f.Text, Style: f.Style})
	}
	return segments
}

type messageMetaFragment struct {
	Text  string
	Style widget.RichTextStyle
}

var hopBadges = [...]string{
	"⓪", "①", "②", "③", "④", "⑤", "⑥", "⑦",
}

func messageMetaFragments(m domain.ChatMessage, meta messageMeta, hasMeta bool) []messageMetaFragment {
	hops, hopsKnown := messageHops(meta, hasMeta)
	hopsLine := "?"
	if hopsKnown {
		hopsLine = hopBadge(hops)
	}

	parts := []messageMetaFragment{{Text: hopsLine, Style: widget.RichTextStyleInline}}
	if isMessageFromMQTT(meta, hasMeta) {
		parts = appendMetaSeparator(parts)
		parts = append(parts, messageMetaFragment{Text: "[MQTT]", Style: widget.RichTextStyleInline})
		return parts
	}

	if m.Direction == domain.MessageDirectionIn && hopsKnown && hops == 0 {
		if quality, ok := signalQualityFromMetrics(meta.RxRSSI, meta.RxSNR); ok {
			parts = appendMetaSeparator(parts)
			parts = append(parts, messageMetaFragment{
				Text:  signalBarsForQuality(quality),
				Style: signalRichTextStyle(signalThemeColorForQuality(quality), true),
			})
		}
		if meta.RxRSSI != nil {
			parts = appendMetaSeparator(parts)
			parts = append(parts, messageMetaFragment{Text: "RSSI: ", Style: widget.RichTextStyleInline})
			parts = append(parts, messageMetaFragment{
				Text:  fmt.Sprintf("%d", *meta.RxRSSI),
				Style: signalRichTextStyle(signalThemeColorForRSSI(*meta.RxRSSI), false),
			})
		}
		if meta.RxSNR != nil {
			parts = appendMetaSeparator(parts)
			parts = append(parts, messageMetaFragment{Text: "SNR: ", Style: widget.RichTextStyleInline})
			parts = append(parts, messageMetaFragment{
				Text:  fmt.Sprintf("%.2f", *meta.RxSNR),
				Style: signalRichTextStyle(signalThemeColorForSNR(*meta.RxSNR), false),
			})
		}
	}

	return parts
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

func appendMetaSeparator(parts []messageMetaFragment) []messageMetaFragment {
	return append(parts, messageMetaFragment{Text: " | ", Style: widget.RichTextStyleInline})
}

func signalRichTextStyle(colorName fyne.ThemeColorName, monospace bool) widget.RichTextStyle {
	style := widget.RichTextStyleInline
	style.ColorName = colorName
	if monospace {
		style.TextStyle.Monospace = true
	}
	return style
}

func messageStatusLine(m domain.ChatMessage) string {
	if m.Direction != domain.MessageDirectionOut {
		return ""
	}
	switch m.Status {
	case domain.MessageStatusPending:
		return "Pending"
	case domain.MessageStatusSent:
		return "Sent"
	case domain.MessageStatusAcked:
		return "Acked"
	case domain.MessageStatusFailed:
		return "Failed"
	default:
		return ""
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

func normalizeNodeID(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" || strings.EqualFold(v, "unknown") || v == "!ffffffff" {
		return ""
	}
	return v
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

func chatTitleByKey(chats []domain.Chat, key string) string {
	if key == "" {
		return "No chat selected"
	}
	for _, c := range chats {
		if c.Key == key {
			return c.Title
		}
	}
	return key
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
