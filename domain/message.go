package domain

import "time"

// Message represents a chat message persisted in the database.
type Message struct {
	ID        int64
	ChatID    string
	SenderID  string
	PortNum   int
	Text      string
	RxSNR     float64
	RxRSSI    int
	Timestamp time.Time
	IsUnread  bool
}
