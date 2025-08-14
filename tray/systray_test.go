package tray

import "testing"

func TestSystrayCallbacks(t *testing.T) {
	s := NewSystray(true)
	calledShow := false
	s.OnShowHide(func() { calledShow = true })
	if s.showHide == nil {
		t.Fatalf("showHide callback not set")
	}
	s.showHide()
	if !calledShow {
		t.Fatalf("showHide callback not invoked")
	}

	var enabled bool
	s.OnToggleNotifications(func(e bool) { enabled = e })
	if s.toggle == nil {
		t.Fatalf("toggle callback not set")
	}
	s.toggle(true)
	if !enabled {
		t.Fatalf("toggle callback not invoked")
	}

	exited := false
	s.OnExit(func() { exited = true })
	if s.exit == nil {
		t.Fatalf("exit callback not set")
	}
	s.exit()
	if !exited {
		t.Fatalf("exit callback not invoked")
	}
}
