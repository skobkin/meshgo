package notifications

// Payload is a generic user-facing notification payload.
type Payload struct {
	Title   string
	Content string
}

// Sender sends notifications using a platform-specific backend.
type Sender interface {
	Send(payload Payload)
}
