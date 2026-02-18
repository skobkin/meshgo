package app

import (
	"strings"

	"github.com/skobkin/meshgo/internal/config"
	"github.com/skobkin/meshgo/internal/radio/busmsg"
)

func TransportNameFromType(transport config.TransportType) string {
	switch transport {
	case config.TransportIP:
		return "ip"
	case config.TransportSerial:
		return "serial"
	case config.TransportBluetooth:
		return "bluetooth"
	default:
		if value := strings.TrimSpace(string(transport)); value != "" {
			return value
		}

		return "unknown"
	}
}

func ConnectionTarget(cfg config.ConnectionConfig) string {
	switch cfg.Transport {
	case config.TransportIP:
		return strings.TrimSpace(cfg.Host)
	case config.TransportSerial:
		return strings.TrimSpace(cfg.SerialPort)
	case config.TransportBluetooth:
		return strings.TrimSpace(cfg.BluetoothAddress)
	default:
		return ""
	}
}

func ConnectionStatusFromConfig(cfg config.ConnectionConfig) busmsg.ConnectionStatus {
	status := busmsg.ConnectionStatus{
		State:         busmsg.ConnectionStateDisconnected,
		TransportName: TransportNameFromType(cfg.Transport),
		Target:        ConnectionTarget(cfg),
	}
	if status.Target != "" {
		status.State = busmsg.ConnectionStateConnecting
	}

	return status
}
