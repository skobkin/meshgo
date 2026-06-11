# MeshGo

[![build status](https://ci.skobk.in/api/badges/5/status.svg?events=push%2Ctag)](https://ci.skobk.in/repos/5)
[![latest release](https://git.skobk.in/skobkin/meshgo/badges/release.svg)](https://git.skobk.in/skobkin/meshgo/releases)

![logo](docs/logo_text.svg)

MeshGo is a desktop client for Meshtastic networks.

It focuses on a practical GUI for connecting to nodes, browsing network data, and interacting with messages.

[![Download latest release](docs/button_download.svg)](https://git.skobk.in/skobkin/meshgo/releases)

Check [Releases](https://git.skobk.in/skobkin/meshgo/releases) section for latest downloads available.

> [!NOTE]
> Releases are hosted on my private [Forgejo](https://en.wikipedia.org/wiki/Forgejo) instance,
> Github is used only to mirror the source code for better availability.

## Screenshots

![chat view](docs/screenshot_chat.png)

<details>
<summary>Chat view</summary>

![hop tooltip](docs/screenshot_chat_hop_tooltip.png)

![delivery indicator](docs/screenshot_chat_delivery_indicator.png)

</details>

<details>
<summary>Node list</summary>

![nodes view](docs/screenshot_nodes.png)

![node overview](docs/screenshot_node_overview.png)

![nodes traceroute](docs/screenshot_traceroute.png)

</details>
<details>
<summary>Map view</summary>

![map view](docs/screenshot_map.png)

![map tile loading status](docs/screenshot_map_loading_indicator.png)

</details>
<details>
<summary>Node settings</summary>

![node user settings](docs/screenshot_node_settings_device.png)

![channel settings](docs/screenshot_node_settings_channels.png)

![channel settings](docs/screenshot_node_settings_channels_edit.png)

</details>
<details>
<summary>App settings</summary>

![app settings](docs/screenshot_app_settings.png)

</details>
<details>
<summary>Update dialog</summary>

![update dialog](docs/screenshot_update_dialog.png)

</details>

## License

MeshGo's original source code is licensed under MIT unless otherwise noted.

The generated Meshtastic protobuf bindings in `internal/radio/meshtasticpb/`
are generated from Meshtastic protobuf schemas from
[`meshtastic/protobufs`](https://github.com/meshtastic/protobufs), which
are licensed under GPL-3.0. These generated files are kept separate from
MeshGo's original code. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)
for details and the upstream license reference.
