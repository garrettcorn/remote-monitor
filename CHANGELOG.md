# Changelog

All notable changes to this project will be documented in this file.

## [1.3.0] - 2026-11-19

### Changed
- Move audio controls and restart/shut down into a new Control Panel section ([`aeaff72`](https://github.com/garrettcorn/remote-monitor/commit/aeaff72))

### Added
- Add Stream Deck-style buttons for media and volume keys, mic mute, lock screen, keyboard shortcuts, opening web pages and launching apps ([`aeaff72`](https://github.com/garrettcorn/remote-monitor/commit/aeaff72))
- Add a button editor to add, edit, delete and reorder buttons with custom labels and colours ([`aeaff72`](https://github.com/garrettcorn/remote-monitor/commit/aeaff72))
- Add `~/.pc_monitor/actions.json` allowlist of apps that buttons may launch ([`aeaff72`](https://github.com/garrettcorn/remote-monitor/commit/aeaff72))
- Add a Customize dialog to show or hide dashboard sections ([`aeaff72`](https://github.com/garrettcorn/remote-monitor/commit/aeaff72))

## [1.2.1] - 2026-11-19

### Changed
- Save the pairing token and paired PIN sessions in `~/.pc_monitor` so dashboards reconnect after a server restart ([`07601f7`](https://github.com/garrettcorn/remote-monitor/commit/07601f7))
- Show a "pairing revoked" status in the dashboard instead of retrying a rejected pairing token ([`07601f7`](https://github.com/garrettcorn/remote-monitor/commit/07601f7))

### Added
- Add `-reset-pairing` flag to revoke all paired dashboards ([`07601f7`](https://github.com/garrettcorn/remote-monitor/commit/07601f7))

### Removed
- Remove the second terminal QR code for the direct LAN URL ([`07601f7`](https://github.com/garrettcorn/remote-monitor/commit/07601f7))

### Fixed
- Fix paired dashboards requiring unpairing and a new PIN after every server restart ([`07601f7`](https://github.com/garrettcorn/remote-monitor/commit/07601f7))

## [1.2.0] - 2026-11-19

### Changed
- **Breaking:** require a one-time 4-digit PIN, shown on the server console, for dashboards to connect when remote control is enabled (the default); use `-read-only` to keep PIN-free monitoring ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Restyle the dashboard with a Liquid Glass theme supporting light and dark appearance ([`ebc806f`](https://github.com/garrettcorn/remote-monitor/commit/ebc806f))
- Escape host-provided names (processes, devices, drives) before rendering them in the dashboard ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))

### Added
- Add remote audio control for output/input device selection, volume and mute ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add remote shutdown and restart with confirmation in the dashboard ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add `-read-only` flag to disable remote control ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add drive health, with SMART temperature, wear, power-on hours and error counts when the server runs elevated ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add a connected devices list for USB, Bluetooth and display devices ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add per-drive read/write speed and active time, and per-adapter rates and link speed ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add GPU fan speed, core/memory clocks and power limit ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add battery status and signed-in users ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))
- Add status badges and a live dashboard link to the README ([`8dc2ccf`](https://github.com/garrettcorn/remote-monitor/commit/8dc2ccf))

### Removed
- Remove the fabricated placeholder drive reported on unsupported platforms ([`85f231b`](https://github.com/garrettcorn/remote-monitor/commit/85f231b))

### Fixed
- Fix the dashboard background flickering ([`609d316`](https://github.com/garrettcorn/remote-monitor/commit/609d316), [`d4d8687`](https://github.com/garrettcorn/remote-monitor/commit/d4d8687))

## [1.1.0] - 2026-11-19

### Changed
- Report OS uptime and boot time instead of time since the server started ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))
- Report the total process count instead of the length of the top-10 list ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))
- Rank top processes by CPU usage, then memory, on Windows ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))
- Widen the terminal QR code quiet zone to the required 4 modules ([`739aaa2`](https://github.com/garrettcorn/remote-monitor/commit/739aaa2))

### Added
- Add a second terminal QR code for the direct LAN URL ([`739aaa2`](https://github.com/garrettcorn/remote-monitor/commit/739aaa2))
- Add ACPI thermal zone temperatures on Windows without requiring admin rights ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))
- Add per-process CPU %, memory % and working-set memory on Windows ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))
- Add process memory (RSS) on Linux and macOS ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))

### Removed
- Remove hard-coded fallback temperatures shown when no sensor is available ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))

### Fixed
- Fix pairing QR codes that could not be scanned due to missing Reed-Solomon error correction ([`739aaa2`](https://github.com/garrettcorn/remote-monitor/commit/739aaa2))
- Fix QR encoder failing for payloads longer than 134 bytes ([`739aaa2`](https://github.com/garrettcorn/remote-monitor/commit/739aaa2))
- Fix dashboard thermals, GPU, storage, adapters and processes never rendering ([`1066534`](https://github.com/garrettcorn/remote-monitor/commit/1066534))
- Fix network totals and rates always reading zero on Windows ([`1066534`](https://github.com/garrettcorn/remote-monitor/commit/1066534))
- Fix per-core CPU usage on Windows being synthesised from overall load ([`f144cce`](https://github.com/garrettcorn/remote-monitor/commit/f144cce))

## [1.0.0] - 2026-11-19

### Added
- Add cross-platform server daemon in Go for Windows, Linux, macOS, Android (Termux), and iOS (#1)
- Add deep system telemetry engine gathering CPU, GPU, thermals, RAM, disk partitions, and network throughput (#1)
- Add Tailcat WireGuard data-plane P2P tunneling with automatic DERP relay bootstrap (#1)
- Add instant 3-way pairing with pairing code, clickable link, and ANSI terminal QR code (#1)
- Add offline HTML pairing card export with embedded SVG QR code (#1)
- Add static GitHub Pages web dashboard with real-time SVG circular gauges and 60-second Canvas sparkline charts (#1)
- Add multi-server fleet management supporting simultaneous monitoring and live server switching in localStorage (#1)
- Add in-browser camera QR code scanner with BarcodeDetector API for instant mobile/webcam pairing (#1)
- Add dual connection mode with direct WebSocket and HTTP polling fallback (#1)

[1.3.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.3.0

[1.2.1]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.2.1

[1.2.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.2.0

[1.1.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.1.0

[1.0.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.0.0
