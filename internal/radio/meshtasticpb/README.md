# Meshtastic protobuf generation

## License

Files in this directory are generated from Meshtastic's official protobuf
schemas. Per the Protocol Buffers license, generated code is owned by the
owner of the input file. The [`meshtastic/protobufs`](https://github.com/meshtastic/protobufs)
repository is licensed under **GPL-3.0**, so these generated `.pb.go`
files inherit the GPL-3.0 license and are kept separate from MeshGo's
MIT-licensed original source code.

See [`THIRD_PARTY_NOTICES.md`](../../../THIRD_PARTY_NOTICES.md) for the
upstream license reference and the rationale for the inherited license.

## Source

These files are generated from Meshtastic official protobuf schemas.

Pinned source:
- Repo: `https://github.com/meshtastic/protobufs`
- Commit: `59cb394dcfc4432cb216358ca26e861c7d13f462`
- Commit date/subject: `2026-05-18 Add T_ECHO_CARD identifier to mesh.proto (#919)`

Generation command (requires `protoc` and `protoc-gen-go`):

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
PATH="$(go env GOPATH)/bin:$PATH" \
  protoc -I /path/to/protobufs \
  --go_out=/tmp/meshtastic-pb-out \
  /path/to/protobufs/nanopb.proto \
  /path/to/protobufs/meshtastic/*.proto
cp /tmp/meshtastic-pb-out/github.com/meshtastic/go/generated/*.pb.go internal/radio/meshtasticpb/
```
