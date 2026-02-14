package ui

import (
	"log/slog"
	"strings"

	meshapp "github.com/skobkin/meshgo/internal/app"
	"github.com/skobkin/meshgo/internal/connectors"
	"github.com/skobkin/meshgo/internal/domain"
)

var appLogger = slog.With("component", "ui.app")

func Run(dep RuntimeDependencies) error {
	return runWithApp(dep, newFyneApp())
}

func initialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
	return meshapp.ConnectionStatusFromConfig(dep.Data.Config.Connection)
}

func resolveInitialConnStatus(dep RuntimeDependencies) connectors.ConnectionStatus {
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

func currentConnStatus(dep RuntimeDependencies) (connectors.ConnectionStatus, bool) {
	if dep.Data.CurrentConnStatus == nil {
		return connectors.ConnectionStatus{}, false
	}

	return dep.Data.CurrentConnStatus()
}

func nodeChanges(store *domain.NodeStore) <-chan struct{} {
	if store == nil {
		return nil
	}

	return store.Changes()
}
