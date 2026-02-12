package app

import (
	"path/filepath"
	"testing"

	"github.com/skobkin/meshgo/internal/config"
)

func TestRuntimeRememberMapViewport_PersistsConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	rt := &Runtime{
		Core: RuntimeCore{
			Paths: Paths{ConfigFile: configPath},
			Config: config.AppConfig{
				Connection: config.ConnectionConfig{
					Connector: config.ConnectorIP,
					Host:      "192.168.1.1",
				},
			},
		},
	}

	rt.RememberMapViewport(12, 34, -56)

	if !rt.Core.Config.UI.MapViewport.Set {
		t.Fatalf("expected map viewport flag to be set")
	}
	if rt.Core.Config.UI.MapViewport.Zoom != 12 || rt.Core.Config.UI.MapViewport.X != 34 || rt.Core.Config.UI.MapViewport.Y != -56 {
		t.Fatalf("unexpected runtime map viewport: %+v", rt.Core.Config.UI.MapViewport)
	}

	saved, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if !saved.UI.MapViewport.Set {
		t.Fatalf("expected saved map viewport flag to be set")
	}
	if saved.UI.MapViewport.Zoom != 12 || saved.UI.MapViewport.X != 34 || saved.UI.MapViewport.Y != -56 {
		t.Fatalf("unexpected saved viewport: %+v", saved.UI.MapViewport)
	}
}
