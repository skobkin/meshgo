package domain

// Chat represents a conversation, either a channel or direct message.
type Chat struct {
	ID            string
	Title         string
	Encryption    int
	LastMessageTS int64
}
