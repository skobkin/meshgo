package ui

import (
	"testing"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	meshapp "github.com/skobkin/meshgo/internal/app"
)

func TestUpdateIndicatorApplySnapshotAndTap(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	var opened meshapp.UpdateSnapshot
	var openCalls int
	indicator := newUpdateIndicator(theme.VariantLight, meshapp.UpdateSnapshot{}, false, func(snapshot meshapp.UpdateSnapshot) {
		openCalls++
		opened = snapshot
	})

	if indicator.Button().Visible() {
		t.Fatalf("expected hidden update button for unknown snapshot")
	}
	if indicator.Button().text != "" {
		t.Fatalf("expected empty update button text, got %q", indicator.Button().text)
	}

	indicator.onTap()
	if openCalls != 0 {
		t.Fatalf("expected no tap callback when update is unknown")
	}

	snapshot := meshapp.UpdateSnapshot{
		CurrentVersion:  "0.6.0",
		UpdateAvailable: true,
		Latest: meshapp.ReleaseInfo{
			Version: "0.7.0",
		},
	}
	indicator.ApplySnapshot(snapshot)

	if !indicator.Button().Visible() {
		t.Fatalf("expected visible update button when update is available")
	}
	if indicator.Button().text != "0.7.0" {
		t.Fatalf("expected latest version on button, got %q", indicator.Button().text)
	}

	indicator.onTap()
	if openCalls != 1 {
		t.Fatalf("expected tap callback once, got %d", openCalls)
	}
	if opened.Latest.Version != "0.7.0" {
		t.Fatalf("expected tapped snapshot latest version, got %q", opened.Latest.Version)
	}
}

func TestUpdateIndicatorApplyThemeAndHideWhenNoUpdate(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	indicator := newUpdateIndicator(theme.VariantLight, meshapp.UpdateSnapshot{}, true, nil)
	indicator.ApplyTheme(theme.VariantDark)
	if indicator.Button().icon == nil {
		t.Fatalf("expected themed update icon")
	}

	indicator.ApplySnapshot(meshapp.UpdateSnapshot{
		CurrentVersion:  "0.7.0",
		UpdateAvailable: false,
	})
	if indicator.Button().Visible() {
		t.Fatalf("expected hidden update button when update is unavailable")
	}
	if indicator.Button().text != "" {
		t.Fatalf("expected empty text when update is unavailable, got %q", indicator.Button().text)
	}
}
