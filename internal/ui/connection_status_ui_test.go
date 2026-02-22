package ui

import (
	"strings"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
	"github.com/skobkin/meshgo/internal/resources"
)

func TestFormatWindowTitle(t *testing.T) {
	got := formatWindowTitle(busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateConnected,
		TransportName: "ip",
	}, "")
	want := "MeshGo " + meshapp.BuildVersion() + " - IP connected"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatConnStatus_WithTargetAndLocalNodeName(t *testing.T) {
	got := formatConnStatus(busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateConnected,
		TransportName: "serial",
		Target:        "/dev/ttyACM2",
	}, "ABCD")
	want := "Serial connected (/dev/ttyACM2, ABCD)"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestTransportDisplayName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ip", in: "ip", want: "IP"},
		{name: "serial", in: "serial", want: "Serial"},
		{name: "bluetooth", in: "bluetooth", want: "Bluetooth LE (unstable)"},
		{name: "fallback", in: "custom", want: "custom"},
		{name: "empty", in: " ", want: ""},
	}

	for _, tt := range tests {
		got := transportDisplayName(tt.in)
		if got != tt.want {
			t.Fatalf("%s: expected %q, got %q", tt.name, tt.want, got)
		}
	}
}

func TestSidebarStatusIcon(t *testing.T) {
	connected := sidebarStatusIcon(busmsg.ConnectionStatus{
		State: busmsg.ConnectionStateConnected,
	})
	if connected != resources.UIIconConnected {
		t.Fatalf("expected connected icon, got %q", connected)
	}

	disconnected := sidebarStatusIcon(busmsg.ConnectionStatus{
		State: busmsg.ConnectionStateDisconnected,
	})
	if disconnected != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon, got %q", disconnected)
	}

	connecting := sidebarStatusIcon(busmsg.ConnectionStatus{
		State: busmsg.ConnectionStateConnecting,
	})
	if connecting != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon for connecting state, got %q", connecting)
	}
}

func TestLocalNodeDisplayName(t *testing.T) {
	if got := localNodeDisplayName(func() meshapp.LocalNodeSnapshot {
		return meshapp.LocalNodeSnapshot{
			ID:   "!11111111",
			Node: domain.Node{NodeID: "!11111111", ShortName: "ABCD", LongName: "Alpha Bravo"},
		}
	}); got != "Alpha Bravo" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := localNodeDisplayName(func() meshapp.LocalNodeSnapshot {
		return meshapp.LocalNodeSnapshot{
			ID:   "!22222222",
			Node: domain.Node{NodeID: "!22222222", ShortName: "EFGH"},
		}
	}); got != "EFGH" {
		t.Fatalf("expected short name fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() meshapp.LocalNodeSnapshot {
		return meshapp.LocalNodeSnapshot{
			ID:   "!33333333",
			Node: domain.Node{NodeID: "!33333333"},
		}
	}); got != "!33333333" {
		t.Fatalf("expected node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() meshapp.LocalNodeSnapshot {
		return meshapp.LocalNodeSnapshot{
			ID:   "!44444444",
			Node: domain.Node{NodeID: "!44444444"},
		}
	}); got != "!44444444" {
		t.Fatalf("expected unknown node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() meshapp.LocalNodeSnapshot {
		return meshapp.LocalNodeSnapshot{}
	}); got != "" {
		t.Fatalf("expected empty for empty node id, got %q", got)
	}
	if got := localNodeDisplayName(nil); got != "" {
		t.Fatalf("expected empty for nil local node snapshot provider, got %q", got)
	}
}

func TestConnectionStatusPresenterSetRefreshAndApplyTheme(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	window := app.NewWindow("status")
	label := widget.NewLabel("")
	var localNameCalls int
	presenter := newConnectionStatusPresenter(
		window,
		label,
		busmsg.ConnectionStatus{
			State:         busmsg.ConnectionStateConnecting,
			TransportName: "ip",
		},
		theme.VariantLight,
		func() string {
			localNameCalls++

			return "ABCD"
		},
	)

	if presenter.SidebarIcon() == nil {
		t.Fatalf("expected sidebar icon")
	}
	if !strings.Contains(label.Text, "ABCD") {
		t.Fatalf("expected initial label to include local name, got %q", label.Text)
	}
	if !strings.Contains(window.Title(), "ABCD") {
		t.Fatalf("expected initial title to include local name, got %q", window.Title())
	}

	presenter.Set(busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateConnected,
		TransportName: "serial",
		Target:        "/dev/ttyACM0",
	}, theme.VariantDark)
	if !strings.Contains(label.Text, "connected") || !strings.Contains(label.Text, "/dev/ttyACM0") {
		t.Fatalf("expected connected status in label, got %q", label.Text)
	}
	presenter.Refresh(theme.VariantLight)
	presenter.ApplyTheme(theme.VariantDark)
	if presenter.SidebarIcon().Resource == nil {
		t.Fatalf("expected sidebar icon resource after theme application")
	}
	if localNameCalls == 0 {
		t.Fatalf("expected local name callback to be used")
	}
}

func TestApplyConnStatusUIHandlesNilTargets(t *testing.T) {
	applyConnStatusUI(nil, nil, nil, busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateDisconnected,
		TransportName: "serial",
	}, theme.VariantLight, "")
}
