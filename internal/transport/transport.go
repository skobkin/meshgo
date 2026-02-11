package transport

import "context"

type Transport interface {
	Name() string
	Connect(ctx context.Context) error
	Close() error
	ReadFrame(ctx context.Context) ([]byte, error)
	WriteFrame(ctx context.Context, payload []byte) error
}

type StatusTargetResolver interface {
	StatusTarget() string
}
