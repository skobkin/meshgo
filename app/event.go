package app

import (
	"time"

	"meshgo/domain"
)

type EventType int

const (
	EventMessage EventType = iota
	EventNode
	EventConnecting
	EventConnected
	EventDisconnected
	EventRetrying
)

type Event struct {
	Type    EventType
	Message *domain.Message
	Node    *domain.Node
	Err     error
	Delay   time.Duration
}

// Events returns a channel receiving application events. It is closed when the
// app's event loop stops.
func (a *App) Events() <-chan Event { return a.events }
