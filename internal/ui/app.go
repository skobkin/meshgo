package ui

import (
	"log/slog"
	"strings"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/domain"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

var appLogger = slog.With("component", "ui.app")

func Run(dep RuntimeDependencies) error {
	return runWithApp(dep, newFyneApp())
}

func initialConnStatus(dep RuntimeDependencies) busmsg.ConnectionStatus {
	return meshapp.ConnectionStatusFromConfig(dep.Data.Config.Connection)
}

func resolveInitialConnStatus(dep RuntimeDependencies) busmsg.ConnectionStatus {
	fallback := initialConnStatus(dep)
	status, ok := currentConnStatus(dep)
	if !ok || status.State == "" {
		return fallback
	}
	if strings.TrimSpace(status.TransportName) == "" {
		status.TransportName = fallback.TransportName
	}
	if strings.TrimSpace(status.Target) == "" {
		status.Target = fallback.Target
	}

	return status
}

func currentConnStatus(dep RuntimeDependencies) (busmsg.ConnectionStatus, bool) {
	if dep.Data.CurrentConnStatus == nil {
		return busmsg.ConnectionStatus{}, false
	}

	return dep.Data.CurrentConnStatus()
}

func nodeChanges(store *domain.NodeStore) <-chan struct{} {
	if store == nil {
		return nil
	}

	return store.Changes()
}
