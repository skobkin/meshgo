package tray

import "testing"

func TestNoopCallbacks(t *testing.T) {
	n := &Noop{quit: make(chan struct{})}
	calledShow := false
	n.OnShowHide(func() { calledShow = true })
	if n.showHide == nil {
		t.Fatalf("showHide callback not set")
	}
	n.showHide()
	if !calledShow {
		t.Fatalf("showHide callback not invoked")
	}

	var enabled bool
	n.OnToggleNotifications(func(e bool) { enabled = e })
	if n.toggle == nil {
		t.Fatalf("toggle callback not set")
	}
	n.toggle(true)
	if !enabled {
		t.Fatalf("toggle callback not invoked")
	}

	exited := false
	n.OnExit(func() { exited = true })
	if n.exit == nil {
		t.Fatalf("exit callback not set")
	}

	// Run should block until Quit is called and trigger the exit callback.
	done := make(chan struct{})
	go func() {
		n.Run()
		close(done)
	}()
	n.Quit()
	<-done
	if !exited {
		t.Fatalf("exit callback not invoked")
	}
}
