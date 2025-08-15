# MeshGo Agent Instructions

## Project Overview
MeshGo is a cross-platform desktop GUI application for managing Meshtastic mesh networks. It provides chat functionality, node management, and device connectivity through serial and TCP/IP transports.

## Key Architecture Principles

### Clean Architecture Layers
1. **UI Layer**: Framework-agnostic view models and presentation (Fyne)
2. **App/Core**: Business logic, use cases, domain models, event bus
3. **Protocols/Transports**: Transport interface with Serial/TCP implementations
4. **Persistence**: SQLite-based storage for messages, nodes, settings

### Core Interfaces
```go
// Transport abstraction for serial/TCP connectivity
type Transport interface {
    Connect(ctx context.Context) error
    Close() error
    ReadPacket(ctx context.Context) ([]byte, error)
    WritePacket(ctx context.Context, []byte) error
    IsConnected() bool
    Endpoint() string
}

// UI abstraction for framework swapping
type UI interface {
    Run()
    ShowMain()
    SetTrayBadge(hasUnread bool)
    Notify(chatID, title, body string)
    UpdateChats(chats []Chat)
    UpdateNodes(nodes []Node)
}
```

### Signal Quality Mapping
- **Good**: SNR ≥ 8dB AND RSSI ≥ -95dBm
- **Fair**: SNR ≥ 2dB AND RSSI ≥ -110dBm  
- **Bad**: All other cases

### Directory Structure
```
meshgo/
├── cmd/meshgo/           # Main application entry point
├── internal/
│   ├── core/            # Domain models, use cases, interfaces
│   ├── transport/       # Serial/TCP transport implementations
│   ├── protocol/        # Meshtastic protocol handling
│   ├── storage/         # SQLite persistence layer
│   ├── system/          # Notifications, tray, platform integration
│   └── ui/              # UI adapter and implementations
├── pkg/                 # Public APIs (if needed)
├── migrations/          # SQLite schema migrations
└── assets/             # Icons, resources
```

## Key Requirements

### Connectivity
- **Serial**: USB/TTY with device detection by VID/PID
- **TCP/IP**: Network connection to Meshtastic device
- **Auto-reconnect**: Exponential backoff (1s → 60s, ±20% jitter)

### Data Model
- **SQLite Schema**: messages, chats, nodes, channels, settings tables
- **Encryption States**: None (0), Default Key (1), Custom Key (2)
- **Persistence**: Config in `~/.config/meshgo/`, WAL mode, migrations

### UI Structure
- **Vertical Tab Rail**: Chats, Nodes, Settings (like Telegram Desktop)
- **Chats**: Channel/DM list with unread counts, message bubbles
- **Nodes**: Filterable list with signal quality, battery, context menus
- **Settings**: Connection, notifications, logging configuration

### System Integration
- **Tray Icon**: With unread badge, show/hide, notifications toggle
- **Notifications**: Native per-platform, suppress when focused
- **Cross-platform**: Linux (libnotify), Windows (toast), macOS (NSUserNotification)

## Dependencies & Libraries

### Core Dependencies
- **GUI**: Qt (therecipe/qt) preferred, Wails/Fyne as alternatives
- **SQLite**: `modernc.org/sqlite` (CGO-free) or `mattn/go-sqlite3`
- **Serial**: `go.bug.st/serial` (MIT license)
- **Tray**: `github.com/getlantern/systray` (Apache-2.0)
- **Notifications**: `github.com/gen2brain/beeep` (MIT)

### Protocol Handling
- Use protobuf for MeshPacket encoding/decoding
- Port routing: TEXT_MESSAGE_APP (chat), POSITION_APP, ROUTING_APP (traceroute)
- Track NodeDB from NodeInfo packets (names, positions, metrics)

## Implementation Guidelines

### Error Handling
- Transport errors → reconnect with backoff
- Decode errors → log and drop packet
- DB failures → retry, degrade to memory queue
- Graceful shutdown → close all transports

### Concurrency
- One read loop per transport connection
- Packet decode on worker pool
- UI updates marshaled to GUI thread
- Event bus with channels for loose coupling

### Security & Privacy
- No PSK storage in logs
- Explicit user consent for data export
- Redact personal IDs in diagnostics

## Testing Strategy
- **Unit**: Transport, protocol, persistence layers
- **Integration**: End-to-end with fake transport
- **UI**: View model state, golden screenshots

## Build & Release
- **Target**: Go 1.24+
- **Platforms**: Linux, Windows, macOS
- **Packaging**: AppImage/.deb/.rpm (Linux), MSIX (Windows), .app bundle (macOS)
- **Versioning**: Semantic versioning with git describe

## Common Patterns

### Event Bus Usage
```go
type Event struct {
    Type EventType
    Data any
}

// Emit events for UI updates
bus.Emit(Event{Type: MessageReceived, Data: msg})
bus.Emit(Event{Type: NodeUpdated, Data: node})
bus.Emit(Event{Type: ConnectionStateChanged, Data: state})
```

### Settings Management
- JSON config file with validation
- Runtime reload without restart
- Migration for config schema changes

### Persistence Patterns
- Batch DB writes for performance
- WAL mode for concurrent reads
- Migrations with schema_version table
- Backup/restore for corruption recovery

## Future Considerations
- v1.1: Per-chat mute, quick replies, improved traceroute
- v1.2: Channel management, richer node details  
- v2: Bluetooth transport, map view, MQTT bridge

## Development Notes
- Always check existing patterns before implementing new features
- Follow Go conventions and clean architecture principles
- Maintain UI framework abstraction for future swapping
- Test on all target platforms regularly
- Document public APIs and complex algorithms

## Original specification
If you're unsure of what app should be able to do, then check the original PRD document in the root directory.
If you can't find it, request it from the user.
