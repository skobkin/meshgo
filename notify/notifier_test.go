package notify

import (
	"testing"
	"time"
)

func TestBeeepNotifier(t *testing.T) {
	var called bool
	n := NewBeeep(true)
	n.notifyFunc = func(title, message string, appIcon any) error {
		called = true
		if title != "t" || message != "b" || appIcon != "" {
			t.Fatalf("unexpected params: %q %q %q", title, message, appIcon)
		}
		return nil
	}
	if err := n.NotifyNewMessage("chat", "t", "b", time.Now()); err != nil {
		t.Fatalf("NotifyNewMessage returned error: %v", err)
	}
	if !called {
		t.Fatal("notifyFunc not called")
	}

	// When disabled it should not invoke notifyFunc.
	n.SetEnabled(false)
	called = false
	if err := n.NotifyNewMessage("chat", "t", "b", time.Now()); err != nil {
		t.Fatalf("NotifyNewMessage returned error: %v", err)
	}
	if called {
		t.Fatal("notifyFunc called when notifier disabled")
	}
}
