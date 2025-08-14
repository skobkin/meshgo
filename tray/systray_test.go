//go:build cgo

package tray

import "testing"

func TestSystrayCallbacks(t *testing.T) {
	tr := NewSystray(true)
	s, ok := tr.(*Systray)
	if !ok {
		t.Fatalf("expected *Systray, got %T", tr)
	}
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

	ready := false
	s.OnReady(func() { ready = true })
	if s.ready == nil {
		t.Fatalf("ready callback not set")
	}
	s.ready()
	if !ready {
		t.Fatalf("ready callback not invoked")
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
