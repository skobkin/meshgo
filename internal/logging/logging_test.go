package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/skobkin/meshgo/internal/config"
)

func TestFanoutWriter_ContinuesWhenOneDestinationFails(t *testing.T) {
	var dst bytes.Buffer
	w := newFanoutWriter(errorWriter{err: errors.New("broken stdout")}, &dst)

	n, err := w.Write([]byte("test"))
	if err != nil {
		t.Fatalf("write returned error: %v", err)
	}
	if n != len("test") {
		t.Fatalf("unexpected bytes written: got %d, want %d", n, len("test"))
	}
	if got := dst.String(); got != "test" {
		t.Fatalf("unexpected destination contents: got %q", got)
	}
}

func TestManagerConfigure_LogFileStillReceivesLogsWhenStdoutFails(t *testing.T) {
	origDefault := slog.Default()
	t.Cleanup(func() { slog.SetDefault(origDefault) })

	origStdout := os.Stdout
	t.Cleanup(func() { os.Stdout = origStdout })

	brokenStdout, err := os.CreateTemp(t.TempDir(), "broken-stdout-*")
	if err != nil {
		t.Fatalf("create broken stdout: %v", err)
	}
	if err := brokenStdout.Close(); err != nil {
		t.Fatalf("close broken stdout: %v", err)
	}
	os.Stdout = brokenStdout

	logPath := filepath.Join(t.TempDir(), "app.log")
	m := NewManager()
	t.Cleanup(func() { _ = m.Close() })

	if err := m.Configure(config.LoggingConfig{Level: "debug", LogToFile: true}, logPath); err != nil {
		t.Fatalf("configure manager: %v", err)
	}

	slog.Info("file must receive this message")

	if err := m.Close(); err != nil {
		t.Fatalf("close manager: %v", err)
	}

	cleanLogPath := filepath.Clean(logPath)
	// #nosec G304 -- logPath is created from t.TempDir() in this test.
	raw, err := os.ReadFile(cleanLogPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !bytes.Contains(raw, []byte("file must receive this message")) {
		t.Fatalf("log file does not contain test message, contents: %q", string(raw))
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write(_ []byte) (int, error) {
	return 0, w.err
}
