package transport

import "context"

// Transport is the common framed I/O contract for connector implementations.
type Transport interface {
	Name() string
	Connect(ctx context.Context) error
	Close() error
	ReadFrame(ctx context.Context) ([]byte, error)
	WriteFrame(ctx context.Context, payload []byte) error
}

// StatusTargetResolver exposes a human-readable endpoint shown in UI status.
type StatusTargetResolver interface {
	StatusTarget() string
}
