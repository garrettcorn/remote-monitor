# 🖥️ PC Monitor & Multi-Server Telemetry Dashboard

[![CI](https://github.com/garrettcorn/remote-monitor/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/garrettcorn/remote-monitor/actions/workflows/ci.yml)
[![Deploy Pages](https://github.com/garrettcorn/remote-monitor/actions/workflows/deploy-pages.yml/badge.svg?branch=main)](https://github.com/garrettcorn/remote-monitor/actions/workflows/deploy-pages.yml)
[![Release](https://img.shields.io/github/v/release/garrettcorn/remote-monitor?sort=semver)](https://github.com/garrettcorn/remote-monitor/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/garrettcorn/remote-monitor/total)](https://github.com/garrettcorn/remote-monitor/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/garrettcorn/remote-monitor)](go.mod)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Dashboard](https://img.shields.io/badge/dashboard-live-0a84ff?logo=githubpages&logoColor=white)](https://garrettcorn.github.io/remote-monitor/)

**🌐 Live dashboard: [garrettcorn.github.io/remote-monitor](https://garrettcorn.github.io/remote-monitor/)**

A lightweight, cross-platform system monitor and real-time dashboard powered by **Go**, **Tailcat** peer-to-point WireGuard networking, and a static web client designed to be hosted directly on **GitHub Pages**.

---

## 🌟 Key Features

### 1. Cross-Platform Server Daemon (Go)
- **Supported Platforms**: Windows (`amd64`, `arm64`), Linux (`amd64`, `arm64`), macOS (Apple Silicon `arm64` & Intel `amd64`), Android (via Termux), and iOS (Darwin).
- **Single Static Binary**: Ultra-low memory footprint (< 15 MB RAM) with zero runtime dependencies.
- **Tailcat Data-Plane**: Uses [`github.com/tailscale/tailcat`](https://github.com/tailscale/tailcat) to create instant, end-to-end encrypted WireGuard tunnels across firewalls and NATs without requiring any Tailscale account, login, or central control plane.
- **Dual Connection Mode**: Supports both Tailcat P2P tunneling and direct LAN/Tailscale IP connections.

### 2. Deep System Telemetry & Metrics
- **CPU**: Model name, architecture, physical & logical core count, overall CPU % load, per-core utilization bars, clock frequency (MHz/GHz), and core temperatures.
- **GPU**: NVIDIA, AMD, Intel, and Apple Silicon detection with core utilization %, VRAM used/total, power draw (watts), and GPU temperatures.
- **Thermals**: Live sensor readings for CPU package, GPU, motherboard, and ambient thermal zones.
- **Memory**: Total, used, free, and available RAM with usage %, plus swap / pagefile breakdown.
- **Storage**: Real-time disk capacity visualizer for every drive (`C:`, `/`, `/home`) showing total, used, and free space with filesystem type (NTFS, APFS, ext4, etc.).
- **Network Activity**: Interface list with MAC addresses, IPv4, IPv6, active link state, instantaneous download/upload speeds (KB/s, MB/s), and total RX/TX byte counters.
- **Operating System & Host**: Hostname, OS version/build, kernel, uptime, and top active processes by memory and CPU.

### 3. Instant 3-Way Pairing
When the server starts, it automatically outputs:
1. **Pairing Code**: The Tailcat address string (`tc...`) and session token.
2. **Clickable Pairing Link**: Formatted directly for your GitHub Pages deployment.
3. **Terminal QR Code**: An ANSI/Unicode QR code rendered directly inside your terminal screen — just point your smartphone camera or webcam at the terminal to pair in 1 second!
4. **Offline Pairing Card**: Automatically saves `pairing_card.html` with an SVG QR code for sharing.

### 4. GitHub Pages Multi-Server Client
- **100% Static Web App**: Modern HTML5, responsive CSS3, and ES Modules. Zero build steps, bundlers, or Node.js dependencies needed.
- **Multi-Server Management**: Pair multiple servers simultaneously! Switch between machines seamlessly or monitor your entire fleet on the **Fleet Overview** grid.
- **Real-Time Visualizations**: Smooth circular SVG gauge dials, per-core progress bars, and high-DPI 60-second Canvas sparkline charts for CPU, RAM, and Network throughput.
- **Built-in Camera QR Scanner**: Scan terminal QR codes directly within the web app using your mobile or laptop camera.
- **Persistent Storage**: All paired servers, tokens, and nicknames are stored safely in `localStorage`.

---

## 🧩 Platform Support

✅ supported · 🟡 partial or needs setup · ❌ not available. Windows is the primary platform and is tested on real hardware; Linux and macOS builds compile in CI and use the tools below, but haven't been verified on real machines yet.

| Feature | Windows | Linux | macOS |
|---|---|---|---|
| **CPU load & frequency** | ✅ `GetSystemTimes`; model + MHz from the registry | ✅ `/proc/stat`, `/proc/cpuinfo` | ✅ `top -l 1`, `sysctl machdep.cpu.brand_string` |
| **Per-core CPU** | ✅ `NtQuerySystemInformation` (processor performance counters) | ✅ `/proc/stat` per-CPU lines | 🟡 shows overall load for every core |
| **Memory & swap** | ✅ `GlobalMemoryStatusEx` | ✅ `/proc/meminfo` | ✅ `sysctl hw.memsize`, `vm_stat` |
| **Uptime** | ✅ `GetTickCount64` | ✅ `/proc/uptime` | ✅ `sysctl kern.boottime` |
| **Processes (top 10, count)** | ✅ Toolhelp snapshot + `GetProcessTimes` / `K32GetProcessMemoryInfo` | ✅ `ps -eo pid,%cpu,%mem,rss,comm` | ✅ `ps -arcwwwxo …` |
| **Signed-in users** | ✅ `WTSEnumerateSessions` (Remote Desktop Services API) | ✅ `who` | ✅ `who` |
| **Battery** | ✅ `GetSystemPowerStatus` | ✅ `/sys/class/power_supply/BAT*` | ✅ `pmset -g batt` |
| **GPU** | ✅ `nvidia-smi` (load, VRAM, temp, power, fan, clocks); name only for AMD/Intel via the registry | 🟡 `nvidia-smi` (load, VRAM, temp); NVIDIA only | 🟡 name only via `system_profiler SPDisplaysDataType` |
| **Temperatures** | 🟡 GPU via `nvidia-smi` + ACPI thermal zones via PDH `\Thermal Zone Information`; no CPU core temps without admin/driver | ✅ `/sys/class/thermal/thermal_zone*` + GPU | 🟡 needs the third-party `osx-cpu-temp` tool |
| **Network totals & rates** | ✅ `GetIfEntry2Ex` (`MIB_IF_ROW2`) per adapter, incl. link speed | ✅ `/proc/net/dev`; link speed from `/sys/class/net/*/speed` | 🟡 adapters and IPs only (Go `net.Interfaces`); no traffic counters |
| **Volumes (space)** | ✅ `GetDiskFreeSpaceEx` for every drive letter | ✅ `/proc/mounts` + `statfs` | 🟡 main volume (`/`) only via `statfs` |
| **Disk I/O (read/write, active %)** | ✅ PDH `\LogicalDisk(*)\…` performance counters | ✅ `/proc/diskstats` | ❌ |
| **Drive health & SMART** | ✅ health via PowerShell `Get-PhysicalDisk`; temperature, wear, power-on hours and errors via `Get-StorageReliabilityCounter` (**admin**) | 🟡 drive list from `/sys/block`; health/SMART via `smartctl -j` (**root**, smartmontools) | ❌ |
| **Connected devices** | ✅ PowerShell `Get-PnpDevice` + device-reported USB names (`DEVPKEY_Device_BusReportedDeviceDesc`); USB, Bluetooth, monitors | 🟡 USB only via `lsusb` | ❌ |
| **Displays (resolution, refresh, scale)** | ✅ `EnumDisplayMonitors`, `EnumDisplaySettingsW`, `GetDpiForMonitor`; model names from WMI `WmiMonitorID` | 🟡 `xrandr --query` (X11) | ✅ `system_profiler SPDisplaysDataType` |
| **Screen preview** | ✅ GDI `BitBlt` per monitor (DPI-aware), downscaled JPEG | ❌ | 🟡 built-in `screencapture` (needs Screen Recording permission) |
| **Audio: list, volume, mute** | ✅ Core Audio (`IMMDeviceEnumerator`, `IAudioEndpointVolume`) | 🟡 `pactl` (PulseAudio/PipeWire, pactl 16+) | 🟡 AppleScript `set volume`: system output/input only, output mute only |
| **Audio: switch default device** | ✅ `IPolicyConfig` (the interface behind the Sound control panel) | 🟡 `pactl set-default-sink/source` | ❌ |
| **Buttons: media & volume keys** | ✅ `SendInput` virtual keys | 🟡 `xdotool key XF86Audio…` (X11) | ❌ |
| **Buttons: keyboard shortcuts** | ✅ `SendInput` | 🟡 `xdotool key` (X11; Wayland blocks synthetic input) | 🟡 AppleScript System Events (needs Accessibility permission) |
| **Buttons: lock screen** | ✅ `LockWorkStation` | 🟡 `loginctl lock-session` | 🟡 `pmset displaysleepnow` |
| **Buttons: open URL / launch app** | ✅ `ShellExecute` | 🟡 `xdg-open` / direct exec | 🟡 `open` |
| **Mic mute toggle** | ✅ Core Audio | 🟡 `pactl` | ❌ |
| **Shutdown / restart** | ✅ `shutdown /s` / `/r` (5 s delay) | 🟡 `systemctl poweroff` / `reboot` | 🟡 AppleScript System Events |

The server is a single Go binary with no runtime dependencies on Windows. The Linux/macOS tools above are used only if they're installed; missing ones just leave that panel empty.

---

## 🚀 Quick Start Guide

### Step 1: Run the Server

#### On Windows:
```powershell
# Run with default settings (Tailcat + Direct LAN port 9120)
.\pc_monitor_server.exe

# Or specify a friendly name and your GitHub Pages client URL:
.\pc_monitor_server.exe -name "GamingPC" -web-url "https://garrettcorn.github.io/remote-monitor/"
```

#### On Linux:
```bash
chmod +x pc_monitor_linux_amd64
./pc_monitor_linux_amd64 -name "LinuxServer" -web-url "https://garrettcorn.github.io/remote-monitor/"
```

#### On macOS:
```bash
chmod +x pc_monitor_darwin_arm64
./pc_monitor_darwin_arm64 -name "MacBookPro" -web-url "https://garrettcorn.github.io/remote-monitor/"
```

#### On Android (Termux):
```bash
pkg install golang git
# Download or copy pc_monitor_linux_arm64:
chmod +x pc_monitor_linux_arm64
./pc_monitor_linux_arm64 -name "AndroidDevice"
```

---

### Step 2: Pair with the Client

Once the server starts, you will see a banner in your terminal:

```
========================================================================
  PC MONITOR SERVER - Cross-Platform System Telemetry & Pairing
========================================================================

  Server Name:    Workstation (windows/amd64)
  Tailcat Addr:   tc1...
  Pairing Token:  4a9f81bc20e98...
  Direct LAN URL: http://192.168.1.120:9120
  Pairing Link:   https://<user>.github.io/pc_monitor/#name=Workstation&tc=tc1...&token=...
  Pairing Card:   pairing_card.html (saved)

  SCAN WITH PHONE CAMERA TO PAIR INSTANTLY:
  [ ANSI QR CODE ]
```

You can pair using any of these methods:
- **Scan the Terminal QR Code**: Open your phone camera or tablet camera and tap the link to open the dashboard and pair automatically.
- **Click the Pairing Link**: Open the generated URL in any web browser.
- **In-App Scanner**: On your GitHub Pages web app, click **➕ Pair Server** ➔ **📷 Scan Terminal QR Code** to use your webcam or phone camera.
- **Manual Input**: Paste the Tailcat address (`tc...`) or direct IP address into the pairing dialog.

### Step 3: Enter the Control PIN

Remote control (volume, mute, audio device switching, shutdown and restart) is **on by default**, so the dashboard asks for a one-time **4-digit PIN** the first time it connects. The PIN is printed only in the server's console:

```
  Remote control: ENABLED (audio, shutdown/restart). Use -read-only to disable.
  Control PIN:    4821   (enter in the dashboard to connect; one-time use)
```

Each PIN works once. After it's used (or after 5 wrong attempts) the server prints a new PIN for the next dashboard:

```
  [pairing] PIN used. New control PIN: 0937
```

Pairings are remembered: the pairing token and paired sessions are saved in `~/.pc_monitor/` (`pairing_token`, `sessions.json`, readable only by your user; sessions are stored as hashes), so dashboards reconnect automatically after the server restarts. To revoke every paired dashboard, start the server once with `-reset-pairing`; it creates a new pairing link and QR code.

Start the server with `-read-only` for monitoring only: control is disabled and no PIN is needed.

### Control Panel & Stream Deck Buttons

The **Control Panel** at the top of the dashboard holds a Stream Deck-style button grid, audio controls (output/input device, volume, mute) and Restart / Shut down. Click **Edit** on the button grid to add, edit, delete or drag-to-reorder buttons. Available actions:

| Action | What it does on the host |
|---|---|
| Media & volume key | Play/pause, next, previous, stop, volume up/down, mute |
| Toggle microphone mute | Mutes/unmutes the default input device |
| Lock screen | Locks the session |
| Keyboard shortcut | Sends a key combination (e.g. `Ctrl+Shift+M` for Discord mute, OBS hotkeys) to the active window |
| Open web page | Opens an `http(s)://` address in the host's default browser |
| Launch app | Starts an app from the server's allowlist (below) |

Button layouts are saved per server in your browser. For safety, dashboards can only launch apps the server owner lists in `~/.pc_monitor/actions.json` (re-read on every press, no restart needed):

```json
{
  "launch": [
    { "id": "obs", "name": "OBS Studio", "path": "C:\\Program Files\\obs-studio\\bin\\64bit\\obs64.exe", "dir": "C:\\Program Files\\obs-studio\\bin\\64bit" },
    { "id": "notepad", "name": "Notepad", "path": "notepad.exe" }
  ]
}
```

Only `id` and `name` are sent to dashboards; paths and arguments stay on the server. Button actions need a PIN-paired dashboard and are disabled in `-read-only` mode. On Linux they use `xdotool`, `loginctl` and `xdg-open` (X11); on macOS, AppleScript (no media keys).

### Displays & Screen Preview

The **Displays** section lists each connected display with its model name, resolution, refresh rate, scaling and which one is primary. On PIN-paired servers you can also see what is on each screen:

- Nothing is captured until you ask: click **Capture** for a one-off snapshot.
- Turn on **Auto-refresh** to capture every 10 seconds (off by default, remembered in this browser). It pauses while the tab is hidden or the section is hidden.
- Click a preview to open it larger. Images are scaled to 1280 px wide JPEGs (about 100 KB each).

Screen preview is never available in `-read-only` mode, and `-no-screen` disables it entirely.

### Customize the Dashboard

Click the **sliders icon** in the header to show or hide any section (CPU, memory, Stream Deck, audio, power, drive health, processes, and so on). Hidden cards leave no gaps, and your choices are saved in this browser.

---

## 🚀 CI/CD Pipelines & Releases

This repository includes full GitHub Actions workflows configured in `.github/workflows/`:

1. **`ci.yml` (CI - Test & Build Matrix)**:
   - Validates `CHANGELOG.md` against [Common Changelog](https://common-changelog.org/) standards.
   - Runs the full Go test suite.
   - Cross-compiles binaries for a matrix of 6 targets: `windows/amd64`, `windows/arm64`, `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `darwin/amd64`.
2. **`deploy-pages.yml` (GitHub Pages Deployment)**:
   - Automatically builds and deploys the `/web` static dashboard to GitHub Pages upon pushing to `main`.
3. **`release.yml` (Automated GitHub Releases)**:
   - Triggered on tag push (`v*`) or manual `workflow_dispatch`.
   - Extracts release notes directly from `CHANGELOG.md` for that specific version.
   - Packages binaries into compressed `.zip` and `.tar.gz` archives with `README.md` and `CHANGELOG.md`.
   - Calculates SHA-256 hashes (`checksums.txt`).
   - Publishes the GitHub Release with attached assets and release notes.

---

## 📝 Common Changelog (https://common-changelog.org)

Project changes are curated strictly following the [Common Changelog](https://common-changelog.org/) specification:
- Curated for humans, using the imperative mood (`Add support for...`).
- Allowed change groups: `### Added`, `### Changed`, `### Removed`, `### Fixed`.
- Breaking changes are highlighted with `**Breaking:**`.
- No noisy `[Unreleased]` section. Every version section is explicitly dated `YYYY-MM-DD`.

### Changelog Helper CLI Tool
A Go tool is included in `cmd/changelog/` to manage releases and validation:

```bash
# Validate that CHANGELOG.md strictly adheres to common-changelog.org
go run ./cmd/changelog validate

# Extract release notes for a version (used by CI release workflow)
go run ./cmd/changelog extract 1.0.0

# Prepare a new version entry with today's date and template categories
go run ./cmd/changelog bump 1.1.0
```

### Performing a New Release

To release a new version:

#### 1. Bump the Changelog
```bash
go run ./cmd/changelog bump 1.1.0
# Edit CHANGELOG.md to add your release bullet points under Added/Changed/Fixed
```

#### 2. Run the Release Helper
**On Windows (PowerShell):**
```powershell
.\scripts\release.ps1 -Version 1.1.0 -Push
```

**On Linux / macOS (Bash):**
```bash
chmod +x scripts/release.sh
./scripts/release.sh 1.1.0 --push
```

The script validates the changelog, runs tests, creates the Git tag `v1.1.0`, and pushes to GitHub. The GitHub Actions release pipeline handles the rest automatically!

---

1. Push this repository to GitHub:
   ```bash
   git remote add origin https://github.com/garrettcorn/remote-monitor.git
   git branch -M main
   git push -u origin main --tags
   ```
2. Enable GitHub Pages:
   - In GitHub, go to repository **Settings** ➔ **Pages**.
   - Under **Build and deployment**, select **Source: GitHub Actions**.
3. The included workflow (`.github/workflows/deploy-pages.yml`) will automatically deploy the `/web` directory to:
   ```
   https://garrettcorn.github.io/remote-monitor/
   ```

---

## 🛠️ CLI Flags Reference

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-name` | Hostname | Friendly nickname for this server displayed in the client |
| `-port` | `9120` | Direct HTTP/WebSocket port for LAN or Tailscale IP connections (set `0` to disable) |
| `-tc-port` | `8080` | TCP port inside the Tailcat WireGuard tunnel |
| `-web-url` | `https://your-username.github.io/pc_monitor/` | Base URL of your deployed GitHub Pages client |
| `-token` | *(saved)* | Custom pairing token. If omitted, a random 32-character hex token is created once and reused from `~/.pc_monitor/pairing_token` |
| `-no-tailcat`| `false` | Run in direct LAN / Tailscale IP mode only without Tailcat |
| `-export-html`| `pairing_card.html`| Path to write a standalone HTML card with SVG QR code |
| `-read-only` | `false` | Monitoring only: disable remote control (audio, shutdown/restart); dashboards connect without a PIN |
| `-reset-pairing` | `false` | Revoke all paired dashboards: new pairing token and cleared PIN sessions |
| `-no-screen` | `false` | Disable screen capture previews (displays are still listed) |
| `-verbose` | `false` | Enable detailed debug logs for networking and telemetry |

---

## 🏗️ Project Architecture

```
pc_monitor/
├── .github/workflows/
│   └── deploy.yml          # GitHub Actions: Pages deployment & Cross-platform binary builds
├── cmd/server/
│   └── main.go             # Server daemon entrypoint & CLI interface
├── pkg/
│   ├── telemetry/          # System telemetry engine
│   │   ├── types.go        # JSON data models for CPU, GPU, memory, disks, network
│   │   ├── collector.go    # Thread-safe orchestrator & delta rate calculations
│   │   ├── collector_windows.go # Windows WMI, registry, and perf counters
│   │   ├── collector_linux.go   # Linux /proc, /sys/class/thermal, and sysfs
│   │   ├── collector_darwin.go  # macOS sysctl, vm_stat, and system_profiler
│   │   └── collector_fallback.go# Fallback OS collectors
│   ├── tailcat/
│   │   └── tunnel.go       # Tailcat P2P WireGuard tunnel manager
│   ├── pairing/
│   │   ├── pairing.go      # Pairing token, URL generator & HTML card exporter
│   │   ├── qrcode.go       # ANSI terminal & SVG QR code renderer
│   │   └── qr_encode.go    # Pure Go ISO/IEC 18004 QR code matrix encoder
│   └── api/
│       ├── server.go       # HTTP REST & WebSocket streaming server
│       └── server_test.go  # Unit test suite
├── web/                    # GitHub Pages static client
│   ├── index.html          # Dashboard interface & multi-server views
│   ├── style.css           # Modern dark cyberpunk UI styling
│   ├── favicon.svg         # SVG icon
│   └── js/
│       ├── app.js          # App orchestrator & hash router
│       ├── storage.js      # Multi-server localStorage registry
│       ├── client.js       # WebSocket, HTTP polling & Tailcat WASM client
│       ├── ui.js           # DOM renderer, SVG gauges, and process tables
│       ├── charts.js       # High-DPI Canvas sparkline charts
│       └── qr-scanner.js   # In-browser camera QR code reader
├── scripts/
│   ├── build.ps1           # Windows build & cross-compilation script
│   └── build.sh            # Unix build & cross-compilation script
├── go.mod
└── go.sum
```

---

## 🔒 Security & Privacy

- **No Third-Party Cloud Dependency**: Your telemetry is never routed through third-party servers. Connections are established point-to-point via WireGuard encryption over Tailcat's data-plane or direct LAN.
- **Session Authentication**: Every server session generates a cryptographically secure pairing token (`crypto/rand`). Requests without the token are rejected with `401 Unauthorized`.
- **Private by Default**: Storage credentials remain strictly in your browser's local `localStorage`.
- **PIN-Gated Control**: When remote control is enabled, telemetry and control both require a session that is only issued in exchange for a one-time PIN shown on the server console. Five wrong guesses replace the PIN, so a stolen pairing link alone cannot connect or power the machine off. Use `-read-only` to remove control entirely. Treat paired dashboards as fully trusted: keyboard-shortcut buttons can type on the host.
- **Plain HTTP on the LAN port**: The direct port (`-port`, default `9120`) is unencrypted HTTP. Prefer Tailscale/Tailcat or a trusted network when control is enabled.
- **Screen previews are sensitive**: they're only served to PIN-paired dashboards, never in `-read-only` mode, and are only captured on request or while auto-refresh is on. Use `-no-screen` to turn them off.
- **SMART details need elevation**: Drive health status works for any user; temperature, wear and error counters require running the server as administrator (Windows) or root with `smartctl` (Linux).
