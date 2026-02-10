package persistence

import (
	"context"
	"log/slog"
	"time"
)

type writeCmd struct {
	name string
	fn   func(context.Context) error
}

type WriterQueue struct {
	logger *slog.Logger
	queue  chan writeCmd
}

func NewWriterQueue(logger *slog.Logger, capacity int) *WriterQueue {
	if capacity <= 0 {
		capacity = 256
	}
	return &WriterQueue{
		logger: logger,
		queue:  make(chan writeCmd, capacity),
	}
}

func (w *WriterQueue) Enqueue(name string, fn func(context.Context) error) {
	cmd := writeCmd{name: name, fn: fn}
	select {
	case w.queue <- cmd:
	default:
		go func() { w.queue <- cmd }()
	}
}

func (w *WriterQueue) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cmd := <-w.queue:
				w.runWithRetry(ctx, cmd)
			}
		}
	}()
}

func (w *WriterQueue) runWithRetry(ctx context.Context, cmd writeCmd) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := cmd.fn(ctx); err != nil {
			w.logger.Error("db write failed", "cmd", cmd.name, "attempt", attempt, "error", err)
			if attempt == maxAttempts {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(attempt) * 300 * time.Millisecond):
			}
			continue
		}
		return
	}
}
