# Changelog

All notable changes to this project will be documented in this file.

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

[1.1.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.1.0

[1.0.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.0.0
