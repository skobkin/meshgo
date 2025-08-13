package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// New creates a logger that writes JSON logs to the given path when enabled.
// It returns the logger and a closer to release any resources.
func New(path string, enabled bool) (*slog.Logger, io.Closer, error) {
	if !enabled {
		return slog.New(slog.NewTextHandler(io.Discard, nil)), noopCloser{}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{AddSource: true})
	return slog.New(handler), f, nil
}

type noopCloser struct{}

func (noopCloser) Close() error { return nil }
