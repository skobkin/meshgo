package ui

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
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
	readIncomingUpToByKey := initialReadIncomingByChat(store, chats)
	unreadByKey := chatUnreadByKey(store, chats, readIncomingUpToByKey)
	if selectedKey != "" && len(chats) > 0 && !hasChat(chats, selectedKey) {
		selectedKey = ""
	}
	if selectedKey == "" && len(chats) > 0 {
		selectedKey = chats[0].Key
	}
	markChatRead(store, readIncomingUpToByKey, selectedKey)
	unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
	messages := store.Messages(selectedKey)
	var messageList *widget.List
	var chatTitle *widget.Label
	var entry *widget.Entry
	pendingScrollChatKey := ""
	pendingScrollMinCount := 0

	chatList := widget.NewList(
		func() int { return len(chats) },
		func() fyne.CanvasObject {
			unreadLabel := widget.NewLabel("●")
			titleLabel := widget.NewLabel("chat")
			titleLabel.TextStyle = fyne.TextStyle{Bold: true}
			typeLabel := widget.NewLabel("type")
			previewLabel := widget.NewLabel("preview")

			return container.NewVBox(
				container.NewHBox(unreadLabel, titleLabel, layout.NewSpacer(), typeLabel),
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
			unreadLabel := line1.Objects[0].(*widget.Label)
			titleLabel := line1.Objects[1].(*widget.Label)
			typeLabel := line1.Objects[3].(*widget.Label)
			previewLabel := root.Objects[1].(*widget.Label)

			unreadLabel.SetText(chatUnreadMarker(unreadByKey[chat.Key]))
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
		markChatRead(store, readIncomingUpToByKey, selectedKey)
		unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
		if onChatSelected != nil {
			onChatSelected(selectedKey)
		}
		messages = store.Messages(selectedKey)
		chatList.Refresh()
		messageList.Refresh()
		chatTitle.SetText(chats[id].Title)
		scrollMessageListToEnd(messageList, len(messages))
		focusEntry(entry)
	}

	chatTitle = widget.NewLabel("No chat selected")
	if selectedKey != "" {
		chatTitle.SetText(chatTitleByKey(chats, selectedKey))
	}
	tooltipLayer := container.NewWithoutLayout()
	tooltipManager := newHoverTooltipManager(tooltipLayer)

	messageList = widget.NewList(
		func() int { return len(messages) },
		func() fyne.CanvasObject {
			transportBadge := newTooltipLabel("", "", tooltipManager)
			messageLine := container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(horizontalSpacer(theme.Padding()), transportBadge, horizontalSpacer(theme.Padding())),
				widget.NewRichTextWithText("message"),
			)
			metaParts := container.NewHBox(widget.NewRichTextWithText("meta"))
			statusBadge := newTooltipLabel("", "", tooltipManager)
			timeLabel := widget.NewLabel("time")
			row := container.NewVBox(
				messageLine,
				container.NewHBox(
					metaParts,
					layout.NewSpacer(),
					container.NewHBox(statusBadge, horizontalSpacer(theme.Padding()/2), timeLabel, horizontalSpacer(0)),
				),
			)
			bubbleBg := canvas.NewRectangle(chatBubbleFillColor(domain.MessageDirectionIn))
			bubbleBg.CornerRadius = 10
			bubble := container.NewStack(bubbleBg, container.NewPadded(row))

			return container.New(newChatRowLayout(false), bubble)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(messages) {
				return
			}
			msg := messages[id]
			meta, hasMeta := parseMessageMeta(msg.MetaJSON)
			rowContainer := obj.(*fyne.Container)
			rowLayout, ok := rowContainer.Layout.(*chatRowLayout)
			if ok {
				rowLayout.SetAlignRight(msg.Direction == domain.MessageDirectionOut)
			}
			bubble := rowContainer.Objects[0].(*fyne.Container)
			bubbleBg := bubble.Objects[0].(*canvas.Rectangle)
			bubbleBg.FillColor = chatBubbleFillColor(msg.Direction)
			bubbleBg.Refresh()
			box := bubble.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
			messageLine := box.Objects[0].(*fyne.Container)
			messageText := messageLine.Objects[0].(*widget.RichText)
			transportSlot := messageLine.Objects[1].(*fyne.Container)
			transportBadge := transportSlot.Objects[1].(*tooltipWidget)
			messageText.Segments = messageTextSegments(msg, meta, hasMeta, nodeNameByID, localNodeID)
			messageText.Refresh()
			transportBadge.SetBadge(messageTransportBadge(msg, meta, hasMeta))
			metaRow := box.Objects[1].(*fyne.Container)
			metaParts := metaRow.Objects[0].(*fyne.Container)
			metaParts.Objects = messageMetaWidgets(msg, meta, hasMeta, tooltipManager)
			metaParts.Refresh()
			metaRight := metaRow.Objects[2].(*fyne.Container)
			metaRight.Objects[0].(*tooltipWidget).SetBadge(messageStatusBadge(msg))
			metaRight.Objects[2].(*widget.Label).SetText(messageTimeLabel(msg.At))

			// Chat rows can have different heights (e.g. multiline message text),
			// so we must update list item height per message.
			messageList.SetItemHeight(id, rowContainer.MinSize().Height)
			rowContainer.Refresh()
		},
	)

	entry = widget.NewEntry()
	entry.SetPlaceHolder("Type message (max 200 bytes)")
	counterLabel := widget.NewLabel("0/200 bytes")
	sendStatusLabel := widget.NewLabel("")
	sendStatusLabel.Truncation = fyne.TextTruncateEllipsis
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
		sendStatusLabel.SetText("")
		setSending(true)
		go func(chatKey, body string) {
			res := <-sender.SendText(chatKey, body)
			if res.Err != nil {
				fyne.Do(func() {
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
				sendStatusLabel.SetText("")
				entry.SetText("")
				setSending(false)
			})
		}(selectedKey, text)
	}

	entry.OnSubmitted = func(_ string) { sendCurrent() }
	sendButton.OnTapped = sendCurrent

	composer := container.NewBorder(nil, nil, nil, sendButton, entry)
	composerStatusRow := container.NewHBox(counterLabel, layout.NewSpacer(), sendStatusLabel)
	right := container.NewBorder(
		chatTitle,
		container.NewVBox(composerStatusRow, composer),
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
		pruneReadIncomingByChat(readIncomingUpToByKey, chats)
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
		markChatRead(store, readIncomingUpToByKey, selectedKey)
		unreadByKey = chatUnreadByKey(store, chats, readIncomingUpToByKey)
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

func chatTypeLabel(c domain.Chat) string {
	if c.Type == domain.ChatTypeDM {
		return "DM"
	}

	return "Channel"
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
	sender, body, hasSender := messageTextParts(m, meta, hasMeta, nodeNameByID, localNodeID)
	if hasSender {
		return fmt.Sprintf("%s: %s", sender, body)
	}

	return body
}

func messageTextParts(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string, localNodeID func() string) (sender, body string, hasSender bool) {
	if m.Direction == domain.MessageDirectionOut {
		if hasMeta {
			if localID := normalizeNodeID(meta.From); localID != "" {
				return displaySender(localID, nodeNameByID), m.Body, true
			}
		}
		if localNodeID != nil {
			if localID := normalizeNodeID(localNodeID()); localID != "" {
				return displaySender(localID, nodeNameByID), m.Body, true
			}
		}

		return "you", m.Body, true
	}
	if hasMeta {
		if sender := normalizeNodeID(meta.From); sender != "" {
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

func messageMetaWidgets(m domain.ChatMessage, meta messageMeta, hasMeta bool, tooltipManager *hoverTooltipManager) []fyne.CanvasObject {
	chunks := messageMetaChunks(m, meta, hasMeta)
	widgets := make([]fyne.CanvasObject, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Tooltip) > 0 {
			widgets = append(widgets, newTooltipRichText(chunk.Segments, chunk.Tooltip, tooltipManager))

			continue
		}
		widgets = append(widgets, widget.NewRichText(chunk.Segments...))
	}

	return widgets
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
	hops, hopsKnown := messageHops(meta, hasMeta)
	parts := make([]messageMetaChunk, 0, 2)
	if hopsKnown && hops > 0 {
		parts = append(parts, newMetaChunkInline(hopBadge(hops)))
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
		return "◷", "Pending"
	case domain.MessageStatusSent:
		return "✓", "Sent"
	case domain.MessageStatusAcked:
		return "✓✓", "Acked"
	case domain.MessageStatusFailed:
		reason := compactWhitespace(strings.TrimSpace(m.StatusReason))
		if reason == "" {
			return "⚠", "Failed"
		}

		return "⚠", "Failed: " + reason
	default:
		return "", ""
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
