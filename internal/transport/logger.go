package transport

import "log/slog"

func transportLogger(name string, attrs ...any) *slog.Logger {
	logger := slog.With("component", "transport", "transport", name)
	if len(attrs) == 0 {
		return logger
	}

	return logger.With(attrs...)
}
