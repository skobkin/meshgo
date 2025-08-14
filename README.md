# meshgo

Meshgo is a cross-platform desktop application written in Go for chatting with and managing [Meshtastic](https://meshtastic.org/) nodes. The project aims to provide a native look and feel with a small footprint while keeping the radio/transport logic separate from the graphical user interface.

## Goals
- Cross-platform: Linux, Windows, macOS.
- Two transports at launch: serial (USB/TTY/COM) and TCP/IP over LAN.
- Automatic reconnect with configurable backoff.
- Native notifications per OS and a tray icon with an unread indicator.
- Persistent storage of settings, logs and message history in SQLite.
- Architecture that allows the GUI toolkit and transports to be swapped later.

## Architecture Overview
The codebase is structured in layers to keep concerns separated:

- **UI Layer**: Responsible for presenting view models and user interaction. The UI backend is swappable via a `ui/adapter` package.
- **App/Core**: Implements use cases, domain models (Node, Chat, Message), state management and an event bus.
- **Protocols/Transports**: Provides a `Transport` interface with serial and TCP implementations. Future transports can be added without changing the core.
- **Meshtastic Service**: `RadioClient` handles protobuf framing, MeshPacket encoding/decoding and reconnect policy.
- **Persistence**: Settings stored as JSON/TOML and chats/nodes in SQLite.
- **System Integration**: `Notifier` for native notifications and `Tray` for a system tray icon.

## Data Storage
Settings and the SQLite database are stored beneath the user configuration directory (e.g. `~/.config/meshgo` on Linux). Message history, node information and channels are persisted with migrations to evolve the schema.

## Licensing
This project is licensed under the [MIT License](LICENSE). All dependencies are chosen from MIT, Apache-2.0 or BSD-compatible licenses.

## Building
### Linux prerequisites
Building with tray support on Linux requires the Ayatana appindicator development packages. Install them with:

```sh
sudo apt-get install -y libayatana-appindicator3-dev
```

Meshgo targets Go 1.24.x. A minimal implementation exists with transport selection, a reconnecting radio client and persistence scaffolding. You can build the current command-line prototype with:

```sh
go build ./cmd/meshgo
```

The prototype exposes a few flags for inspecting stored data:

```sh
./meshgo -list-chats   # list chats with unread counts
./meshgo -list-nodes   # list known nodes
./meshgo -list-channels        # list channels and PSK class
./meshgo -list-messages <chat>   # list recent messages in a chat
```

## Roadmap
- **v1**: serial and IP transport, chat and node management, notifications, tray integration and SQLite persistence.
- **v1.1+**: quality-of-life improvements, richer node details and additional transports like Bluetooth.

## Contributing
Contributions are welcome! Please open issues or pull requests to discuss improvements or report bugs.

