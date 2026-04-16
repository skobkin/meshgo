package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"

	"github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
	generated "github.com/skobkin/meshgo/internal/radio/meshtasticpb"
)

func TestNodeTabIncludesImportExportAndMaintenanceTabs(t *testing.T) {
	tab := newNodeTabWithOnShow(RuntimeDependencies{})
	topTabs, ok := tab.(*container.AppTabs)
	if !ok {
		t.Fatalf("expected node tab root to be app tabs, got %T", tab)
	}
	if len(topTabs.Items) < 6 {
		t.Fatalf("expected all top-level tabs to be present, got %d", len(topTabs.Items))
	}
	if topTabs.Items[4].Text != "Import/Export" {
		t.Fatalf("expected Import/Export tab at index 4, got %q", topTabs.Items[4].Text)
	}
	if topTabs.Items[5].Text != "Maintenance" {
		t.Fatalf("expected Maintenance tab at index 5, got %q", topTabs.Items[5].Text)
	}
}

func TestNodeImportExportPageDisablesActionsWithoutNodeSettingsService(t *testing.T) {
	page := newNodeImportExportPage(RuntimeDependencies{})

	exportButton := mustFindButtonByText(t, page, "Export profile…")
	importButton := mustFindButtonByText(t, page, "Import profile…")
	if !exportButton.Disabled() {
		t.Fatalf("expected export button to be disabled without node settings service")
	}
	if !importButton.Disabled() {
		t.Fatalf("expected import button to be disabled without node settings service")
	}
}

func TestNodeMaintenancePageDisablesActionsWithoutNodeSettingsService(t *testing.T) {
	page := newNodeMaintenancePage(RuntimeDependencies{})

	for _, title := range []string{"Reboot", "Shutdown", "Factory reset", "Reset node DB"} {
		if button := mustFindButtonByText(t, page, title); !button.Disabled() {
			t.Fatalf("expected %s button to be disabled without node settings service", title)
		}
	}
}

func TestDefaultNodeSettingsProfileFilenameUsesSanitizedLocalNodeName(t *testing.T) {
	filename := defaultNodeSettingsProfileFilename(RuntimeDependencies{
		Data: DataDependencies{
			LocalNodeSnapshot: func() app.LocalNodeSnapshot {
				return app.LocalNodeSnapshot{
					Present: true,
					Node: domain.Node{
						LongName: "Base / Alpha",
					},
				}
			},
		},
	})

	if !strings.HasPrefix(filename, "Meshtastic_Base___Alpha_") {
		t.Fatalf("unexpected filename prefix: %q", filename)
	}
	if !strings.HasSuffix(filename, "_nodeConfig.cfg") {
		t.Fatalf("unexpected filename suffix: %q", filename)
	}
}

func TestSanitizeProfileFilenamePart(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "TrimAndKeepSafe", input: " Alpha-42 ", want: "Alpha-42"},
		{name: "ReplaceUnsafe", input: "A/B:C", want: "A_B_C"},
		{name: "CollapseToNode", input: " / ", want: "node"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitizeProfileFilenamePart(tc.input); got != tc.want {
				t.Fatalf("unexpected sanitized name: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestBuildDeviceProfileSummary(t *testing.T) {
	longName := "Desk Node"
	shortName := "DN"
	summary := buildDeviceProfileSummary(&generated.DeviceProfile{
		LongName:       &longName,
		ShortName:      &shortName,
		Ringtone:       &shortName,
		CannedMessages: &longName,
		Config:         &generated.LocalConfig{Device: &generated.Config_DeviceConfig{}},
		ModuleConfig:   &generated.LocalModuleConfig{Audio: &generated.ModuleConfig_AudioConfig{}},
	})

	for _, fragment := range []string{"Desk Node", "DN", "Config sections: 1", "Module sections: 1", "Ringtone: true", "Canned messages: true"} {
		if !strings.Contains(summary, fragment) {
			t.Fatalf("expected profile summary to contain %q, got %q", fragment, summary)
		}
	}
}
