package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/skobkin/meshgo/internal/config"
)

// Manager owns app logger configuration and optional log file lifecycle.
type Manager struct {
	mu     sync.RWMutex
	logger *slog.Logger
	file   *os.File
}

func NewManager() *Manager {
	m := &Manager{}
	m.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	return m
}

func (m *Manager) Configure(cfg config.LoggingConfig, filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.file != nil {
		_ = m.file.Close()
		m.file = nil
	}

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return err
	}

	writer := io.Writer(os.Stdout)
	if cfg.LogToFile {
		cleanPath := filepath.Clean(filePath)
		// #nosec G304 -- path is resolved by app runtime and points to user config dir.
		file, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		m.file = file
		writer = newFanoutWriter(os.Stdout, file)
	}

	h := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	m.logger = slog.New(h)
	slog.SetDefault(m.logger)

	return nil
}

func (m *Manager) Logger(component string) *slog.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.logger.With("component", component)
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.file != nil {
		if err := m.file.Close(); err != nil {
			return err
		}
		m.file = nil
	}

	return nil
}

func parseLevel(raw string) (slog.Leveler, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return nil, fmt.Errorf("unsupported log level: %q", raw)
	}
}

type fanoutWriter struct {
	writers []io.Writer
}

func newFanoutWriter(writers ...io.Writer) io.Writer {
	filtered := make([]io.Writer, 0, len(writers))
	for _, w := range writers {
		if w != nil {
			filtered = append(filtered, w)
		}
	}

	return &fanoutWriter{writers: filtered}
}

func (w *fanoutWriter) Write(p []byte) (int, error) {
	var (
		wroteAny bool
		firstErr error
	)

	for _, dst := range w.writers {
		n, err := dst.Write(p)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}

			continue
		}
		if n != len(p) {
			if firstErr == nil {
				firstErr = io.ErrShortWrite
			}

			continue
		}
		wroteAny = true
	}

	if wroteAny {
		return len(p), nil
	}
	if firstErr != nil {
		return 0, firstErr
	}

	return len(p), nil
}
