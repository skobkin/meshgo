package tray

import "testing"

func TestNoopCallbacks(t *testing.T) {
	n := &Noop{}
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
	n.exit()
	if !exited {
		t.Fatalf("exit callback not invoked")
	}

	// Run and Quit should be no-ops
	n.Run()
	n.Quit()
}
