# Meshtastic Protocol Buffer Definitions

This directory contains the official Meshtastic protocol buffer Go definitions, sourced from:

**Origin:** https://github.com/lmatte7/goMesh/tree/main/github.com/meshtastic/gomeshproto

These are properly generated protobuf Go files that provide:
- Correct protobuf reflection metadata
- All Meshtastic message types (FromRadio, ToRadio, MeshPacket, etc.)
- Proper enum definitions
- Full compatibility with google.golang.org/protobuf

## Key Files:
- `mesh.pb.go` - Core Meshtastic protocol messages
- `portnums.pb.go` - Port number definitions
- `config.pb.go` - Configuration messages
- `telemetry.pb.go` - Device metrics and telemetry
- And many others for specific Meshtastic functionality

## License
These files are part of the Meshtastic project and follow the same licensing terms.

## Attribution
Generated protobuf code from the lmatte7/goMesh project, which provides working Go protobuf definitions for the Meshtastic protocol.