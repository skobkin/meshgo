package ui

import (
	"strings"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/resources"
)

func TestFormatWindowTitle(t *testing.T) {
	got := formatWindowTitle(connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
		TransportName: "ip",
	}, "")
	want := "MeshGo " + meshapp.BuildVersion() + " - IP connected"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatConnStatus_WithTargetAndLocalNodeName(t *testing.T) {
	got := formatConnStatus(connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
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
	connected := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateConnected,
	})
	if connected != resources.UIIconConnected {
		t.Fatalf("expected connected icon, got %q", connected)
	}

	disconnected := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateDisconnected,
	})
	if disconnected != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon, got %q", disconnected)
	}

	connecting := sidebarStatusIcon(connectors.ConnectionStatus{
		State: connectors.ConnectionStateConnecting,
	})
	if connecting != resources.UIIconDisconnected {
		t.Fatalf("expected disconnected icon for connecting state, got %q", connecting)
	}
}

func TestLocalNodeDisplayName(t *testing.T) {
	store := domain.NewNodeStore()
	store.Upsert(domain.Node{NodeID: "!11111111", ShortName: "ABCD", LongName: "Alpha Bravo"})
	store.Upsert(domain.Node{NodeID: "!22222222", ShortName: "EFGH"})
	store.Upsert(domain.Node{NodeID: "!33333333"})

	if got := localNodeDisplayName(func() string { return "!11111111" }, store); got != "Alpha Bravo" {
		t.Fatalf("expected long name, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!22222222" }, store); got != "EFGH" {
		t.Fatalf("expected short name fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!33333333" }, store); got != "!33333333" {
		t.Fatalf("expected node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!44444444" }, store); got != "!44444444" {
		t.Fatalf("expected unknown node id fallback, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "" }, store); got != "" {
		t.Fatalf("expected empty for empty node id, got %q", got)
	}
	if got := localNodeDisplayName(nil, store); got != "" {
		t.Fatalf("expected empty for nil localNodeID, got %q", got)
	}
	if got := localNodeDisplayName(func() string { return "!11111111" }, nil); got != "!11111111" {
		t.Fatalf("expected node id fallback for nil store, got %q", got)
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
		connectors.ConnectionStatus{
			State:         connectors.ConnectionStateConnecting,
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

	presenter.Set(connectors.ConnectionStatus{
		State:         connectors.ConnectionStateConnected,
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
	applyConnStatusUI(nil, nil, nil, connectors.ConnectionStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: "serial",
	}, theme.VariantLight, "")
}
