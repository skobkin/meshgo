# MeshGo

A modern desktop GUI application for Meshtastic mesh networks, built in Go with Fyne.

## Features

- **Cross-platform GUI**: Native desktop application with system tray integration
- **Multiple connections**: Serial and TCP/IP connectivity to Meshtastic devices  
- **Messaging**: Send/receive text messages on channels and direct messages
- **Node management**: Real-time node discovery with signal quality indicators
- **Persistent storage**: SQLite database for message history and node information
- **Notifications**: Native desktop notifications for new messages
- **CI/CD ready**: Automated builds and releases with Drone CI

## Quick Start

### Requirements
- Go 1.24+
- Linux/Windows/macOS with graphics support

### Installation

```bash
git clone <repository-url>
cd meshgo
go build -o meshgo ./cmd/meshgo
./meshgo
```

Or download pre-built binaries from the [releases page](../../releases).

### Usage

**GUI Mode** (default):
```bash
./meshgo
```

**Console Mode** (debugging):
```bash
./meshgo --console
```

Console commands: `connect serial <port>`, `connect ip <host:port>`, `send <chat> <message>`, `nodes`, `chats`, `exit`

## Architecture

```
cmd/meshgo/           # Main application
internal/
├── core/            # Business logic and interfaces
├── transport/       # Serial/TCP connectivity
├── protocol/        # Meshtastic protocol handling
├── storage/         # SQLite persistence
├── system/          # Notifications and system tray
└── ui/              # Fyne GUI implementation
```

The application uses clean architecture with:
- **Transport abstraction**: Pluggable connectivity (serial, TCP, future Bluetooth)
- **Event-driven design**: Reactive UI updates via events
- **Protocol layer**: Proper Meshtastic protobuf handling with gomeshproto
- **Persistent storage**: SQLite with automatic migrations

## Configuration

Config stored at `~/.config/meshgo/config.json`:

```json
{
  "connection": {
    "type": "serial",
    "serial": {"port": "/dev/ttyUSB0", "baud": 115200},
    "ip": {"host": "192.168.1.100", "port": 4403}
  },
  "ui": {"start_minimized": false, "theme": "system"},
  "notifications": {"enabled": true}
}
```

## Building

```bash
# Development
make build
make test
make fmt

# All platforms
make build-all

# With CI/CD
# Drone CI automatically builds and releases on git tags
```

## Signal Quality

Node signal strength calculated from RSSI/SNR:
- **Good**: SNR ≥ 8dB AND RSSI ≥ -95dBm  
- **Fair**: SNR ≥ 2dB AND RSSI ≥ -110dBm
- **Bad**: Below fair thresholds

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Meshtastic](https://meshtastic.org/) project
- [lmatte7/goMesh](https://github.com/lmatte7/goMesh) for protobuf definitions