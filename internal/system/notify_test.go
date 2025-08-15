package system

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestNewNotifier(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	if notifier == nil {
		t.Fatal("NewNotifier returned nil")
	}

	if notifier.logger != logger {
		t.Error("Logger not set correctly")
	}

	// Default state should be enabled
	if !notifier.IsEnabled() {
		t.Error("Notifier should be enabled by default")
	}
}

func TestNotifier_SetEnabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	// Test enabling
	notifier.SetEnabled(true)
	if !notifier.IsEnabled() {
		t.Error("Notifier should be enabled after SetEnabled(true)")
	}

	// Test disabling
	notifier.SetEnabled(false)
	if notifier.IsEnabled() {
		t.Error("Notifier should be disabled after SetEnabled(false)")
	}
}

func TestNotifier_NotifyNewMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	// Test with notifications enabled
	notifier.SetEnabled(true)
	err := notifier.NotifyNewMessage("chat1", "Test Chat", "Hello, World!", time.Now())
	// Note: This might fail on headless systems, so we just check it doesn't panic
	if err != nil {
		t.Logf("NotifyNewMessage failed (this may be expected on headless systems): %v", err)
	}

	// Test with notifications disabled
	notifier.SetEnabled(false)
	err = notifier.NotifyNewMessage("chat1", "Test Chat", "Hello, World!", time.Now())
	if err != nil {
		t.Errorf("NotifyNewMessage should not fail when disabled, got: %v", err)
	}
}

func TestNotifier_Alert(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	// Test with notifications enabled
	notifier.SetEnabled(true)
	err := notifier.Alert("Test Title", "Test Message")
	// Note: This might fail on headless systems, so we just check it doesn't panic
	if err != nil {
		t.Logf("Alert failed (this may be expected on headless systems): %v", err)
	}

	// Test with notifications disabled
	notifier.SetEnabled(false)
	err = notifier.Alert("Test Title", "Test Message")
	if err != nil {
		t.Errorf("Alert should not fail when disabled, got: %v", err)
	}
}

func TestNotifier_Beep(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	// Test with notifications enabled
	notifier.SetEnabled(true)
	err := notifier.Beep()
	// Note: This might fail on headless systems, so we just check it doesn't panic
	if err != nil {
		t.Logf("Beep failed (this may be expected on headless systems): %v", err)
	}

	// Test with notifications disabled
	notifier.SetEnabled(false)
	err = notifier.Beep()
	if err != nil {
		t.Errorf("Beep should not fail when disabled, got: %v", err)
	}
}

func TestNotifier_ConcurrentAccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	notifier := NewNotifier(logger)

	// Test concurrent enable/disable operations
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			notifier.SetEnabled(true)
			notifier.SetEnabled(false)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = notifier.IsEnabled()
			_ = notifier.NotifyNewMessage("test", "Test", "Message", time.Now())
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Test should complete without data races or panics
}