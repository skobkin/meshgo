package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio"
)

func newChatsTab(store *domain.ChatStore, sender interface {
	SendText(chatKey, text string) <-chan radio.SendResult
}, nodeNameByID func(string) string) fyne.CanvasObject {
	chats := store.ChatListSorted()
	selectedKey := ""
	if len(chats) > 0 {
		selectedKey = chats[0].Key
	}
	messages := store.Messages(selectedKey)
	var messageList *widget.List
	var chatTitle *widget.Label

	chatList := widget.NewList(
		func() int { return len(chats) },
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel("chat"), widget.NewLabel("meta"))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(chats) {
				return
			}
			chat := chats[id]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(chat.Title)
			box.Objects[1].(*widget.Label).SetText(chatMetaLine(chat))
		},
	)
	chatList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(chats) {
			return
		}
		selectedKey = chats[id].Key
		messages = store.Messages(selectedKey)
		messageList.Refresh()
		chatTitle.SetText(chats[id].Title)
	}

	chatTitle = widget.NewLabel("No chat selected")
	if selectedKey != "" {
		chatTitle.SetText(chatTitleByKey(chats, selectedKey))
	}

	messageList = widget.NewList(
		func() int { return len(messages) },
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel("message"), widget.NewLabel("meta"))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(messages) {
				return
			}
			msg := messages[id]
			meta, hasMeta := parseMessageMeta(msg.MetaJSON)
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(messageTextLine(msg, meta, hasMeta, nodeNameByID))
			box.Objects[1].(*widget.Label).SetText(messageMetaLine(msg, meta, hasMeta))
		},
	)

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Type message (max 200 bytes)")
	counterLabel := widget.NewLabel("0/200 bytes")
	statusLabel := widget.NewLabel("")
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
			statusLabel.SetText("Select a chat first")
			return
		}
		if text == "" {
			statusLabel.SetText("Message is empty")
			return
		}
		if len([]byte(text)) > 200 {
			statusLabel.SetText("Message exceeds 200-byte limit")
			return
		}

		setSending(true)
		statusLabel.SetText("Sending...")
		go func(chatKey, body string) {
			res := <-sender.SendText(chatKey, body)
			if res.Err != nil {
				fyne.Do(func() {
					statusLabel.SetText("Send failed: " + res.Err.Error())
					setSending(false)
				})
				return
			}
			fyne.Do(func() {
				entry.SetText("")
				statusLabel.SetText("Sent")
				setSending(false)
			})
		}(selectedKey, text)
	}

	entry.OnSubmitted = func(_ string) { sendCurrent() }
	sendButton.OnTapped = sendCurrent

	composer := container.NewBorder(nil, nil, nil, sendButton, entry)
	right := container.NewBorder(
		chatTitle,
		container.NewVBox(counterLabel, statusLabel, composer),
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
		if selectedKey == "" && len(chats) > 0 {
			selectedKey = chats[0].Key
		}
		if selectedKey != "" && !hasChat(chats, selectedKey) {
			selectedKey = ""
			if len(chats) > 0 {
				selectedKey = chats[0].Key
			}
		}
		messages = store.Messages(selectedKey)
		chatTitle.SetText(chatTitleByKey(chats, selectedKey))
		chatList.Refresh()
		messageList.Refresh()
	}

	go func() {
		for range store.Changes() {
			fyne.Do(func() {
				refreshFromStore()
			})
		}
	}()

	if len(chats) > 0 {
		chatList.Select(0)
	}

	return container.New(layout.NewStackLayout(), split)
}

func chatMetaLine(c domain.Chat) string {
	typeLabel := "Channel"
	if c.Type == domain.ChatTypeDM {
		typeLabel = "DM"
	}
	if c.LastSentByMeAt.IsZero() {
		return typeLabel
	}
	return fmt.Sprintf("%s | last sent by me %s", typeLabel, c.LastSentByMeAt.Format(time.RFC3339))
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

func messageTextLine(m domain.ChatMessage, meta messageMeta, hasMeta bool, nodeNameByID func(string) string) string {
	prefix := "<"
	if m.Direction == domain.MessageDirectionOut {
		prefix = ">"
	}
	if m.Direction == domain.MessageDirectionIn && hasMeta {
		if sender := normalizeNodeID(meta.From); sender != "" {
			return fmt.Sprintf("%s %s: %s", prefix, displaySender(sender, nodeNameByID), m.Body)
		}
	}
	return fmt.Sprintf("%s %s", prefix, m.Body)
}

func messageMetaLine(m domain.ChatMessage, meta messageMeta, hasMeta bool) string {
	hops, hopsKnown := messageHops(meta, hasMeta)
	hopsLine := "hops: ?"
	if hopsKnown {
		hopsLine = fmt.Sprintf("hops: %d", hops)
	}

	parts := []string{hopsLine}
	if isMessageFromMQTT(meta, hasMeta) {
		parts = append(parts, "[MQTT]")
		return strings.Join(parts, " | ")
	}

	if m.Direction == domain.MessageDirectionIn && hopsKnown && hops == 0 {
		if meta.RxRSSI != nil {
			parts = append(parts, fmt.Sprintf("RSSI: %d", *meta.RxRSSI))
		}
		if meta.RxSNR != nil {
			parts = append(parts, fmt.Sprintf("SNR: %.2f", *meta.RxSNR))
		}
	}

	return strings.Join(parts, " | ")
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
