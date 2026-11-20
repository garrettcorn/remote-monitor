# Changelog

All notable changes to this project will be documented in this file.

## [1.8.1] - 2026-11-20

### Changed
- Show the operating system's full name on the dashboard without the "windows" prefix, and its short name in the fleet overview ([`b77f644`](https://github.com/garrettcorn/remote-monitor/commit/b77f644))

### Fixed
- Fix Windows 11 being reported as Windows 10 ([`b77f644`](https://github.com/garrettcorn/remote-monitor/commit/b77f644))

## [1.8.0] - 2026-11-20

### Changed
- Group processes by app with combined CPU, memory and process count, and friendly program names ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Average process CPU over 5 seconds so the list is steady on a quiet computer ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Measure current process CPU on Linux instead of the average since the process started ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Rename Top Active Processes to Processes ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))

### Added
- Add sortable CPU, GPU, Memory and I/O columns to the process list ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Add process search and a Show all option ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Add process details: path, user, administrator, start time, parent, threads, handles and memory ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Add End task for an app or a single process, with system processes protected ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))
- Add app icons to the process list on Windows ([`7542531`](https://github.com/garrettcorn/remote-monitor/commit/7542531))

## [1.7.3] - 2026-11-20

### Changed
- Restart the tray icon automatically after an update so it runs the new version ([`cb7d788`](https://github.com/garrettcorn/remote-monitor/commit/cb7d788))

### Fixed
- Fix Windows updates failing with "Access is denied" while a tray icon from an earlier version is running ([`cb7d788`](https://github.com/garrettcorn/remote-monitor/commit/cb7d788))
- Fix a failed Windows update leaving the console and tray programs on different versions ([`cb7d788`](https://github.com/garrettcorn/remote-monitor/commit/cb7d788))

## [1.7.2] - 2026-11-20

### Changed
- Allow a single modifier key on its own (Win, Ctrl, Shift or Alt) in shortcuts and Control Panel buttons ([`c13398f`](https://github.com/garrettcorn/remote-monitor/commit/c13398f))

### Fixed
- Fix the Win button in the screen viewer being refused by the server ([`c13398f`](https://github.com/garrettcorn/remote-monitor/commit/c13398f))
- Keep remote control on when the host refuses a single key or click ([`c13398f`](https://github.com/garrettcorn/remote-monitor/commit/c13398f))

## [1.7.1] - 2026-11-20

### Added
- Add a full screen mode to the screen viewer with the Remote control switch, sharper live frames and, in Chrome and Edge, capture of keys like Esc and Alt+Tab ([`7a40eeb`](https://github.com/garrettcorn/remote-monitor/commit/7a40eeb))

### Fixed
- Fix the screen viewer being limited to 480 px wide ([`7a40eeb`](https://github.com/garrettcorn/remote-monitor/commit/7a40eeb))

## [1.7.0] - 2026-11-20

### Changed
- Check for updates every 6 hours instead of once a day ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))
- Make `-no-screen` also turn off remote control ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))
- Rename the remaining "Stream Deck" references to the Control Panel ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))

### Added
- Add remote control of the host's mouse and keyboard from the Displays viewer, with a live view (Windows; Linux X11 via `xdotool`) ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))
- Add saving a full-resolution screen capture as a JPEG ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))
- Add automatic updates, off by default: settings page, tray menu or `pc_monitor update --auto on` ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))
- Add a Check for updates button to the dashboard; after updating, it waits for the server to come back on the new version ([`addee2c`](https://github.com/garrettcorn/remote-monitor/commit/addee2c))

## [1.6.0] - 2026-11-20

### Changed
- Save settings in `~/.pc_monitor/config.json`; command-line flags now override saved settings and `--save` keeps them ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))
- Rename the files inside release archives to `pc_monitor`, `pc_monitor.exe` and `pc_monitor-app.exe` ([`b3f5eeb`](https://github.com/garrettcorn/remote-monitor/commit/b3f5eeb))
- Stop writing `pairing_info.json` and the pairing card to the current folder; pairing info now lives in `~/.pc_monitor` and the card is opt-in with `--export-html` ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))

### Added
- Add management commands: `start`, `stop`, `restart`, `status`, `pin`, `settings`, `config`, `autostart`, `update` and `version` ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))
- Add a local settings page with the pairing QR code, PIN, start-at-login, updates and all options ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))
- Add start at login (per-user, inside the desktop session) on Windows, Linux and macOS ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))
- Add signed self-updates from GitHub Releases, plus the server version and an update button in the dashboard ([`d64cc45`](https://github.com/garrettcorn/remote-monitor/commit/d64cc45))
- Add a tray / menu-bar app, including a windowless `pc_monitor-app.exe` on Windows ([`22b5808`](https://github.com/garrettcorn/remote-monitor/commit/22b5808))
- Add Windows installers, Linux AppImages with zsync updates, `.deb` and `.rpm` packages, and Homebrew, Scoop and winget manifests ([`b3f5eeb`](https://github.com/garrettcorn/remote-monitor/commit/b3f5eeb))

## [1.5.0] - 2026-11-19

### Changed
- Replace the button editor's action dropdown with App, Shortcut, Media & system, Text and Website tabs ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))
- Light up the mute and mic-mute buttons while muted ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))

### Added
- Add a searchable picker of apps installed on the host for launch buttons ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))
- Add a searchable library of common keyboard shortcuts that adapts to the host's OS ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))
- Add text-snippet buttons that type saved text on the host ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))
- Add Lock and Sleep to the power controls ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))
- Add a custom colour wheel for button colours ([`94cd3cb`](https://github.com/garrettcorn/remote-monitor/commit/94cd3cb))

## [1.4.0] - 2026-11-19

### Changed
- Remove the "Stream Deck" label from the Control Panel button grid ([`1f9113f`](https://github.com/garrettcorn/remote-monitor/commit/1f9113f))

### Added
- Add a Displays section listing connected displays with model name, resolution, refresh rate and scaling ([`1f9113f`](https://github.com/garrettcorn/remote-monitor/commit/1f9113f))
- Add on-demand screen preview with optional 10-second auto-refresh, off by default ([`1f9113f`](https://github.com/garrettcorn/remote-monitor/commit/1f9113f))
- Add `-no-screen` flag to disable screen previews ([`1f9113f`](https://github.com/garrettcorn/remote-monitor/commit/1f9113f))
- Add a README table of supported features per operating system and the technology behind each ([`1f9113f`](https://github.com/garrettcorn/remote-monitor/commit/1f9113f))

## [1.3.1] - 2026-11-19

### Changed
- Connect automatically once the 4-digit PIN is entered in the dashboard ([`a1cb90e`](https://github.com/garrettcorn/remote-monitor/commit/a1cb90e))

### Added
- Report whether the server runs as administrator/root (`server_elevated`) ([`2dd0ff2`](https://github.com/garrettcorn/remote-monitor/commit/2dd0ff2))

### Fixed
- Fix the drive health admin note showing on elevated servers whose drives don't report SMART details ([`2dd0ff2`](https://github.com/garrettcorn/remote-monitor/commit/2dd0ff2))

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

[1.8.1]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.8.1
[1.8.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.8.0
[1.7.3]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.7.3
[1.7.2]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.7.2
[1.7.1]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.7.1
[1.7.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.7.0
[1.6.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.6.0

[1.5.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.5.0

[1.4.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.4.0

[1.3.1]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.3.1

[1.3.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.3.0

[1.2.1]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.2.1

[1.2.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.2.0

[1.1.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.1.0

[1.0.0]: https://github.com/garrettcorn/remote-monitor/releases/tag/v1.0.0
