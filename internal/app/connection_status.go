package app

import (
	"strings"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/connectors"
)

func TransportNameFromConnector(connector config.ConnectorType) string {
	switch connector {
	case config.ConnectorIP:
		return "ip"
	case config.ConnectorSerial:
		return "serial"
	case config.ConnectorBluetooth:
		return "bluetooth"
	default:
		if value := strings.TrimSpace(string(connector)); value != "" {
			return value
		}
		return "unknown"
	}
}

func ConnectionTarget(cfg config.ConnectionConfig) string {
	switch cfg.Connector {
	case config.ConnectorIP:
		return strings.TrimSpace(cfg.Host)
	case config.ConnectorSerial:
		return strings.TrimSpace(cfg.SerialPort)
	case config.ConnectorBluetooth:
		return strings.TrimSpace(cfg.BluetoothAddress)
	default:
		return ""
	}
}

func ConnectionStatusFromConfig(cfg config.ConnectionConfig) connectors.ConnectionStatus {
	status := connectors.ConnectionStatus{
		State:         connectors.ConnectionStateDisconnected,
		TransportName: TransportNameFromConnector(cfg.Connector),
		Target:        ConnectionTarget(cfg),
	}
	if status.Target != "" {
		status.State = connectors.ConnectionStateConnecting
	}

	return status
}
