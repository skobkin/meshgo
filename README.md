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

### Installation & Usage

```bash
# Build from source
git clone <repository-url>
cd meshgo
make build
./build/meshgo

# Or download from releases
# https://github.com/your-repo/meshgo/releases
```

**Console Mode** (debugging):
```bash
./build/meshgo --console
```

Console commands: `connect serial <port>`, `connect ip <host:port>`, `send <chat> <message>`, `nodes`, `chats`, `exit`

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

## Development

```bash
make test      # Run tests
make fmt       # Format code
make build-all # Build for all platforms
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