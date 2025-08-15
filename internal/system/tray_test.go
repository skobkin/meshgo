package system

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewSystemTray(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	if tray == nil {
		t.Fatal("NewSystemTray returned nil")
	}

	if tray.logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestSystemTray_SetUnread(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// These calls should not panic even on headless systems
	tray.SetUnread(true)
	tray.SetUnread(false)
}

func TestSystemTray_SetWindowVisible(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// These calls should not panic even on headless systems
	tray.SetWindowVisible(true)
	tray.SetWindowVisible(false)
}

func TestSystemTray_Callbacks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// Test setting callbacks
	var showHideCalled, notificationsCalled, exitCalled bool

	tray.OnShowHide(func() {
		showHideCalled = true
	})

	tray.OnToggleNotifications(func(enabled bool) {
		notificationsCalled = true
	})

	tray.OnExit(func() {
		exitCalled = true
	})

	// Callbacks are set but won't be invoked in this test environment
	// Just verify they can be set without panicking
	if showHideCalled || notificationsCalled || exitCalled {
		t.Error("Callbacks should not be invoked just by setting them")
	}
}

func TestSystemTray_RunAndQuit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// Test that Run() and Quit() don't panic on headless systems
	// Run() should return immediately on stub implementation
	done := make(chan bool, 1)
	go func() {
		tray.Run()
		done <- true
	}()

	// Give it a moment to potentially start
	select {
	case <-done:
		// Expected for stub implementation
	case <-time.After(100 * time.Millisecond):
		// If it's still running, try to quit
		tray.Quit()
		select {
		case <-done:
			// Good, it quit
		case <-time.After(100 * time.Millisecond):
			t.Error("Tray should have quit when requested")
		}
	}
}

func TestSystemTray_MultipleCallbacks(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// Test that setting multiple callbacks works (last one should win)
	var counter int

	tray.OnShowHide(func() {
		counter = 1
	})

	tray.OnShowHide(func() {
		counter = 2
	})

	// The callbacks won't be invoked in test environment
	// but they should be settable without panicking
	_ = counter // Suppress unused variable warning
}

func TestSystemTray_ConcurrentOperations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	tray := NewSystemTray(logger)

	// Test concurrent operations
	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 50; i++ {
			tray.SetUnread(i%2 == 0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			tray.SetWindowVisible(i%2 == 0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			tray.OnShowHide(func() {})
		}
		done <- true
	}()

	// Wait for all operations to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Good
		case <-time.After(1 * time.Second):
			t.Error("Concurrent operations took too long")
		}
	}
}