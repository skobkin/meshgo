# MeshGo

A cross-platform desktop GUI application for managing Meshtastic mesh networks, built in Go with clean architecture principles.

## Features

✅ **Core Functionality**
- Serial and TCP/IP connectivity to Meshtastic devices
- Send and receive text messages on channels and direct messages
- Node database with signal quality indicators (Good/Fair/Bad)
- Persistent message history with SQLite storage
- Cross-platform system tray with unread indicators
- Native notifications for new messages
- Auto-reconnection with exponential backoff

✅ **Architecture**
- Clean separation between UI, core logic, and transport layers
- Framework-agnostic UI adapter (console UI implemented, GUI ready)
- Event-driven design with proper error handling
- SQLite persistence with migrations
- Configurable settings with JSON storage

🚧 **Planned Features**
- Qt/Wails GUI implementation
- Traceroute visualization
- Channel management
- Node favorites and filtering
- Map integration
- Bluetooth transport

## Quick Start

### Prerequisites
- Go 1.24 or later
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/your-username/meshgo.git
cd meshgo

# Build and run
make build
./build/meshgo
```

The application now uses Fyne for both GUI and system tray integration, requiring no additional system dependencies.

### Usage

The application is a cross-platform GUI application with integrated system tray support:

- **GUI Mode** (default): `make build && ./build/meshgo`
  - Full Fyne GUI window with connection controls, node lists, and messaging
  - Integrated system tray with right-click menu
  - Cross-platform notifications
  - Clean shutdown via window close or system tray
  
- **Headless Mode** (`make build-no-gui`): For servers or systems without GUI
  - No GUI dependencies
  - Background daemon mode
  - Use Ctrl+C or system signals to exit
  
- **Console Mode** (debugging): Add `--console` flag to use interactive console
  ```bash
  ./build/meshgo --console
  ```
  Available console commands:
  ```
  help                    - Show help
  connect serial <port>   - Connect via serial (e.g., connect serial /dev/ttyUSB0)
  connect ip <host:port>  - Connect via TCP/IP (e.g., connect ip 192.168.1.100:4403)
  disconnect              - Disconnect
  status                  - Show connection status
  nodes                   - List discovered nodes
  chats                   - List chats
  send <chat> <message>   - Send message
  trace <nodeID>          - Traceroute to node
  favorite <nodeID>       - Toggle node favorite
  exit                    - Exit application
  ```

The application currently uses an event-driven architecture with the following capabilities:
- Automatic device discovery and connection
- Background message processing and notifications
- Persistent storage of messages and node information
- System tray integration with unread message indicators

## Architecture

### Directory Structure

```
meshgo/
├── cmd/meshgo/           # Main application entry point
├── internal/
│   ├── core/            # Domain models, business logic, interfaces
│   ├── transport/       # Serial/TCP transport implementations  
│   ├── protocol/        # Meshtastic protocol handling
│   ├── storage/         # SQLite persistence layer
│   ├── system/          # Notifications, tray, platform integration
│   └── ui/              # UI adapter and implementations
├── migrations/          # Database schema migrations
└── assets/             # Icons, resources
```

### Key Components

**Transport Layer**: Abstracts connectivity (serial, TCP, future Bluetooth)
```go
type Transport interface {
    Connect(ctx context.Context) error
    ReadPacket(ctx context.Context) ([]byte, error)
    WritePacket(ctx context.Context, []byte) error
    IsConnected() bool
}
```

**Protocol Layer**: Handles Meshtastic packet encoding/decoding
- MeshPacket framing and routing
- Port-based message handling (text, position, routing, etc.)
- Node database management

**Storage Layer**: SQLite-based persistence
- Messages with chat organization
- Node information with signal metrics
- Settings with JSON configuration

**UI Layer**: Framework-agnostic adapter pattern
- Console UI implemented
- Qt/Wails GUI ready for integration
- Event-driven updates

## Building

### Development
```bash
# Full development cycle
make dev

# Run tests
make test

# Format code
make fmt

# Run with coverage
make test-coverage
```

### Production Builds
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Create distribution packages
make package
```

## Configuration

Configuration is stored in `~/.config/meshgo/config.json`:

```json
{
  "connection": {
    "type": "serial",
    "serial": {
      "port": "/dev/ttyUSB0", 
      "baud": 115200
    },
    "ip": {
      "host": "192.168.1.100",
      "port": 4403
    }
  },
  "reconnect": {
    "initial_millis": 1000,
    "max_millis": 60000,
    "multiplier": 1.6,
    "jitter": 0.2
  },
  "notifications": {
    "enabled": true
  },
  "logging": {
    "enabled": false,
    "level": "info"
  },
  "ui": {
    "start_minimized": false,
    "theme": "system"
  }
}
```

## Database

SQLite database is stored at `~/.config/meshgo/meshgo.db` with tables:
- `messages` - Chat messages with metadata
- `nodes` - Node information and metrics
- `chats` - Chat/channel definitions
- `channels` - Channel configurations  
- `settings` - Key-value settings storage

## Signal Quality

Node signal quality is calculated from RSSI and SNR:
- **Good**: SNR ≥ 8dB AND RSSI ≥ -95dBm
- **Fair**: SNR ≥ 2dB AND RSSI ≥ -110dBm
- **Bad**: All other cases

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Meshtastic](https://meshtastic.org/) for the amazing mesh networking project
- Go community for excellent libraries and tools