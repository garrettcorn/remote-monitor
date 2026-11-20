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
- **Operating System & Host**: Hostname, OS version/build, kernel, uptime, and running apps grouped by program with CPU, GPU, memory and I/O, search, details and End task.

### 3. Easy Pairing
Each server has a **pairing link** and **QR code**, shown on its local settings page (tray icon → **Open settings…**), by `pc_monitor pin`, and in the terminal when run in the foreground. Scan the QR code with a phone camera or open the link in any browser, then enter the one-time **4-digit PIN** shown on that computer. See [Step 3: Pair the dashboard](#step-3-pair-the-dashboard).

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
| **Processes (grouped by app, CPU averaged over 5 s)** | ✅ Toolhelp snapshot + `GetProcessTimes` / `K32GetProcessMemoryInfo` / `GetProcessIoCounters`; names from each exe's version info | ✅ `/proc/<pid>/stat`, `io` (I/O for your own processes, or all as root) | 🟡 `ps -axo pid,time,rss,comm` (no I/O) |
| **Per-process GPU** | ✅ `GPU Engine` performance counters (busiest engine, like Task Manager) | ❌ | ❌ |
| **Process details** | ✅ Toolhelp, process token (user, administrator), handle count, private memory | ✅ `/proc/<pid>/status`, `fd`, `exe` | 🟡 `ps` (user, parent, start time) |
| **End task** | ✅ `TerminateProcess` | ✅ `SIGTERM` | ✅ `SIGTERM` |
| **App icons** | ✅ `ExtractIconExW` → PNG | ❌ letter badges | ❌ letter badges |
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
| **Remote control (mouse & keyboard)** | ✅ `SetCursorPos` + `SendInput` (mouse buttons, wheel, keys, Unicode text) | 🟡 `xdotool` (X11); no screen view yet | ❌ |
| **Audio: list, volume, mute** | ✅ Core Audio (`IMMDeviceEnumerator`, `IAudioEndpointVolume`) | 🟡 `pactl` (PulseAudio/PipeWire, pactl 16+) | 🟡 AppleScript `set volume`: system output/input only, output mute only |
| **Audio: switch default device** | ✅ `IPolicyConfig` (the interface behind the Sound control panel) | 🟡 `pactl set-default-sink/source` | ❌ |
| **Buttons: media & volume keys** | ✅ `SendInput` virtual keys | 🟡 `xdotool key XF86Audio…` (X11) | ❌ |
| **Buttons: keyboard shortcuts** | ✅ `SendInput` | 🟡 `xdotool key` (X11; Wayland blocks synthetic input) | 🟡 AppleScript System Events (needs Accessibility permission) |
| **Buttons: lock screen** | ✅ `LockWorkStation` | 🟡 `loginctl lock-session` | 🟡 `pmset displaysleepnow` |
| **Buttons: installed-app picker** | ✅ PowerShell `Get-StartApps` (Start-menu, Store and game shortcuts), launched via `shell:AppsFolder\<AppID>` | 🟡 freedesktop `.desktop` entries, launched with `gtk-launch` (or the entry's `Exec`) | 🟡 `.app` bundles in the Applications folders, launched with `open -a` |
| **Buttons: type text snippet** | ✅ `SendInput` with `KEYEVENTF_UNICODE` (any language/emoji) | 🟡 `xdotool type` (X11) | 🟡 AppleScript `keystroke` (needs Accessibility permission) |
| **Buttons: open URL / custom launch** | ✅ `ShellExecute` | 🟡 `xdg-open` / direct exec | 🟡 `open` |
| **Mic mute toggle** | ✅ Core Audio | 🟡 `pactl` | ❌ |
| **Shutdown / restart** | ✅ `shutdown /s` / `/r` (5 s delay) | 🟡 `systemctl poweroff` / `reboot` | 🟡 AppleScript System Events |
| **Tray / menu-bar icon** | ✅ `fyne.io/systray` in the windowless `pc_monitor-app.exe` | ✅ `fyne.io/systray` via D-Bus StatusNotifierItem (GNOME needs the AppIndicator extension) | ✅ `fyne.io/systray` (cgo build) |
| **Start at login** | ✅ per-user `HKCU\…\Run` entry (no admin) | ✅ XDG autostart entry (`~/.config/autostart`) | ✅ LaunchAgent (`~/Library/LaunchAgents`), or `brew services` |
| **Self-update** | ✅ signed download; renames the running `.exe` aside | ✅ replaces the binary or the AppImage file; also `.zsync` for AppImageUpdate | ✅ signed download |
| **Packages** | Inno Setup installer, `.zip` (Scoop planned) | AppImage, `.deb`, `.rpm`, `.tar.gz` | `.tar.gz` (Homebrew planned) |
| **Sleep** | ✅ `SetSuspendState` (powrprof, with `SeShutdownPrivilege`; sleeps rather than hibernates) | 🟡 `systemctl suspend` | 🟡 `pmset sleepnow` |

The server is a single Go binary with no runtime dependencies on Windows. The Linux/macOS tools above are used only if they're installed; missing ones just leave that panel empty.

---

## 🚀 Quick Start Guide

PC Monitor has two parts: the **server**, which you install on each computer you want to monitor, and the **dashboard**, which runs in any browser at [garrettcorn.github.io/remote-monitor](https://garrettcorn.github.io/remote-monitor/) (nothing to install). The dashboard's **Get started** page also offers a one-click download for your system.

### Step 1: Install the server

Download from the **[latest release](https://github.com/garrettcorn/remote-monitor/releases/latest)**. Pick the file for your computer (`<version>` is the release number, e.g. `1.8.1`):

| Computer | Download | How to install |
|---|---|---|
| **Windows** (most PCs) | [`pc_monitor_<version>_windows_amd64_setup.exe`](https://github.com/garrettcorn/remote-monitor/releases/latest) | Run it. It installs for your user only, no admin needed, and can start PC Monitor when you log in. If SmartScreen says "Windows protected your PC", click **More info → Run anyway** (the app isn't code-signed). |
| **Windows on ARM** (Surface Pro X, Snapdragon laptops) | [`pc_monitor_<version>_windows_arm64_setup.exe`](https://github.com/garrettcorn/remote-monitor/releases/latest) | Same as above. |
| **Mac with Apple Silicon** (M1 and later) | [`pc_monitor_<version>_darwin_arm64.tar.gz`](https://github.com/garrettcorn/remote-monitor/releases/latest) | Double-click to unpack, then see [macOS](#macos) below. |
| **Mac with Intel** | [`pc_monitor_<version>_darwin_amd64.tar.gz`](https://github.com/garrettcorn/remote-monitor/releases/latest) | Same as above. |
| **Linux** (most PCs) | [`pc_monitor-<version>-x86_64.AppImage`](https://github.com/garrettcorn/remote-monitor/releases/latest) | `chmod +x pc_monitor-*.AppImage`, then double-click it or run it. It updates itself. Some distributions need FUSE 2 (`sudo apt install libfuse2`), or run it with `--appimage-extract-and-run`. |
| **Linux on ARM** (Raspberry Pi 4/5, ARM laptops) | [`pc_monitor-<version>-aarch64.AppImage`](https://github.com/garrettcorn/remote-monitor/releases/latest) | Same as above. |
| **Debian / Ubuntu** (package) | [`pc-monitor_<version>_amd64.deb`](https://github.com/garrettcorn/remote-monitor/releases/latest) (or `_arm64`) | `sudo apt install ./pc-monitor_<version>_amd64.deb` |
| **Fedora / RHEL / openSUSE** (package) | [`pc-monitor-<version>-1.x86_64.rpm`](https://github.com/garrettcorn/remote-monitor/releases/latest) (or `.aarch64`) | `sudo dnf install ./pc-monitor-<version>-1.x86_64.rpm` |
| **Android (Termux)** | [`pc_monitor_<version>_linux_arm64.tar.gz`](https://github.com/garrettcorn/remote-monitor/releases/latest) | `tar xzf` it in Termux and run `./pc_monitor` |

Not sure which you have? On Windows, **Settings → System → About** shows "x64" or "ARM64". On a Mac, ** → About This Mac** shows "Apple M…" or "Intel". On Linux, `uname -m` prints `x86_64` or `aarch64`.

Every release also has plain `.zip` / `.tar.gz` archives with just the program, plus `checksums.txt` (signed) to verify downloads. Homebrew and Scoop packages are planned but not published yet.

#### macOS

Browsers mark downloaded programs as "quarantined", and PC Monitor isn't signed by Apple, so the first launch is blocked. The simplest way is to download and run it from **Terminal**, which skips the quarantine (use `darwin_amd64` on an Intel Mac):

```bash
cd ~ && mkdir -p pc_monitor && cd pc_monitor
curl -fsSL "$(curl -fsSL https://api.github.com/repos/garrettcorn/remote-monitor/releases/latest | grep -o 'https://[^"]*darwin_arm64.tar.gz')" | tar xz
./pc_monitor
```

If you unpacked it in Finder instead, run `xattr -d com.apple.quarantine pc_monitor` in that folder first, or allow it under **System Settings → Privacy & Security → Open Anyway**. To start it at login with a menu-bar icon, run `./pc_monitor autostart on` and then `./pc_monitor tray`.

### Step 2: Start the server

- **Windows and Linux (AppImage / package):** open **PC Monitor** from the Start menu or app launcher. A tray icon appears and the server runs in the background. Choose **Open settings…** from the tray icon to see the pairing QR code and PIN.
- **From a terminal (any system):** run `pc_monitor`. It prints the pairing link, a QR code and the PIN, and keeps running until you close the window. `pc_monitor start` runs it in the background instead, and `pc_monitor settings` opens the settings page.

The settings page opens in your browser but only works on that computer. It shows the QR code, the pairing link, the current PIN and every option.

### Step 3: Pair the dashboard

Pairing tells a browser how to reach the server. Use the QR code or pairing link from the settings page (or terminal), whichever is easiest:

1. **On a phone or tablet:** point the camera at the QR code and tap the link. The dashboard opens and adds the computer.
2. **On another computer:** copy the **pairing link** from the settings page and open it in the browser, or open the [dashboard](https://garrettcorn.github.io/remote-monitor/), click **Pair Server** and paste the link. **Scan QR Code with Camera** there uses a webcam instead.
3. **Enter the PIN.** The dashboard asks for the 4-digit PIN shown on the settings page, in the tray's settings, or by `pc_monitor pin`. It connects as soon as you type the fourth digit.

That's it. The dashboard remembers the computer (in your browser), so it reconnects on its own, even after the server restarts. Pair more computers the same way and switch between them, or see them all side by side in **Fleet Overview**.

**About the PIN:** each PIN works once. After it's used, or after 5 wrong tries, the server shows a new one for the next browser. The PIN only appears on the computer itself, so someone who gets hold of the pairing link still can't connect. To disconnect every paired browser, use **Revoke all dashboards** on the settings page (or start the server once with `-reset-pairing`); this also changes the pairing link and QR code. Start the server with `-read-only` (or turn on read-only mode in settings) for monitoring only: remote control is off and no PIN is needed.

**Connection tips:** the dashboard connects to the server directly over your network on port `9120`. Both devices need to be on the same network, or connected through a VPN such as Tailscale. If it can't connect, allow PC Monitor through the computer's firewall (Windows asks the first time it runs). Some browsers block a secure site (the dashboard) from talking to a plain-HTTP address on your network. If yours does, allow "insecure content" for the dashboard's site in the browser's site settings.

### Managing the server

| Command | What it does |
|---|---|
| `pc_monitor settings` | Opens the **local settings page** (only reachable from this computer): pairing QR, PIN, start at login, updates and all options |
| `pc_monitor start` / `stop` / `restart` / `status` | Control the background server |
| `pc_monitor pin` | Shows the pairing link, QR code and current PIN |
| `pc_monitor autostart on` / `off` | Start when you log in (inside your desktop session, so screen, audio and button features work) |
| `pc_monitor config list` / `set <key> <value>` | View or change saved settings (`~/.pc_monitor/config.json`) |
| `pc_monitor update` / `update --check` | Install or check for the latest release |
| `pc_monitor update --auto on` / `off` | Install new releases automatically (checks every 6 hours) |
| `pc_monitor tray` | Show the tray / menu-bar icon |

### Updates

The server checks GitHub for new releases every 6 hours (turn this off in settings). When one is available you can install it from the **settings page**, the **tray menu**, the **dashboard** or `pc_monitor update`. On PIN-paired dashboards, **Check for updates** next to the server version asks the server to look right away; if there's a newer release it becomes **Update to vX**, and the dashboard waits for the server to restart and reconnects on its own.

**Automatic updates** (off by default): turn on **Install updates automatically** in the settings page or tray menu, or run `pc_monitor update --auto on`. The server then installs new signed releases by itself and restarts (dashboards are offline for a few seconds and reconnect). Turning it on checks immediately. Dashboards show **Auto-update on** next to the version. Copies installed by a package manager (Homebrew, Scoop, `.deb`/`.rpm`) can't replace their own files, so use the package manager there.

Every release's `checksums.txt` is signed with the project's ed25519 key. The updater only installs a download whose signature verifies against the key built into PC Monitor **and** whose SHA-256 matches. Installs from a `.deb` or `.rpm` show where to download the new package.

---

### Control Panel Buttons

The **Control Panel** at the top of the dashboard holds a customizable button grid, audio controls (output/input device, volume, mute) and power controls (**Lock, Sleep, Restart, Shut down**). The mute and mic-mute buttons light up while muted.

Click **Edit** on the button grid to add, edit, delete or drag-to-reorder buttons. The editor has five tabs:

| Tab | What you pick | What it does on the host |
|---|---|---|
| **App** | Any app installed on the host, from a searchable list (Start-menu programs, Store apps and games such as Steam titles) | Launches it |
| **Shortcut** | A searchable library of ~60 common shortcuts (Windows, Xbox Game Bar, editing, browser, Discord, Teams, Zoom, Google Meet; macOS equivalents on Mac hosts), or record your own combination | Sends the keys to the active window |
| **Media & system** | Play/pause, previous, next, stop, volume up/down, mute speakers, mic mute, lock screen | Media keys / mute / lock |
| **Text** | Up to 1000 characters, e.g. an email address or a canned reply | Types the text into the active window |
| **Website** | An `http(s)://` address | Opens it in the host's default browser |

Labels are suggested automatically (you can change them), and button layouts are saved per server in your browser.

For safety, the dashboard never sends program paths. App buttons refer to an app by the ID the server discovered itself, and the server checks that ID against its own list before launching. To launch something that needs arguments or a working folder, list it in `~/.pc_monitor/actions.json` (re-read on every press, no restart needed); these appear under **Custom** in the App tab:

```json
{
  "launch": [
    { "id": "obs", "name": "OBS Studio", "path": "C:\\Program Files\\obs-studio\\bin\\64bit\\obs64.exe", "dir": "C:\\Program Files\\obs-studio\\bin\\64bit" },
    { "id": "notepad", "name": "Notepad", "path": "notepad.exe" }
  ]
}
```

Only `id` and `name` are sent to dashboards; paths and arguments stay on the server. Button actions need a PIN-paired dashboard and are disabled in `-read-only` mode. See [Platform Support](#-platform-support) for how each action works on Linux and macOS.

### Displays & Screen Preview

The **Displays** section lists each connected display with its model name, resolution, refresh rate, scaling and which one is primary. On PIN-paired servers you can also see what is on each screen:

- Nothing is captured until you ask: click **Capture** for a one-off snapshot.
- Turn on **Auto-refresh** to capture every 10 seconds (off by default, remembered in this browser). It pauses while the tab is hidden or the section is hidden.
- Click a preview to open it larger. Images are scaled to 1280 px wide JPEGs (about 100 KB each).
- **Save** (on a preview or in the viewer) downloads a fresh full-resolution capture as a JPEG named after the server, display and time.
- **Remote control**: in the viewer, switch on **Remote control** to use the host's mouse and keyboard. The view refreshes several times a second while it's on; clicks, right-clicks, dragging, scrolling and typing go to that display. Use the **Type to send** box on phones and tablets, and the **Win**, **Alt+Tab** and **Esc** buttons for keys your browser keeps for itself. Closing the viewer turns control off.
- **Full screen**: click **Full screen** in the viewer (or press **F**) to fill your screen; the Remote control switch, Refresh, Save and the key buttons stay at hand, and the live view gets sharper (up to 2560 px). In Chrome and Edge, full screen with Remote control on also captures keys the browser normally keeps, such as **Esc**, **Alt+Tab** and **Ctrl+W**; hold **Esc** to leave. Otherwise press **Esc** (with Remote control off) or **Exit full screen**. On iPhone, the viewer fills the browser window instead.

Screen preview and remote control are never available in `-read-only` mode, and `-no-screen` disables both. Remote control works on Windows (and Linux X11 with `xdotool`, where screen capture isn't available yet); it is not supported on macOS. A server that isn't running as administrator can't click or type into windows that are (Windows blocks it).

### Processes

The **Processes** section groups running processes by program, so a browser with 19 processes is one row showing its combined use. Each row shows the program's friendly name and icon (Windows), how many processes it runs, and its **CPU** (averaged over 5 seconds so a quiet PC shows a steady list), **GPU** (Windows), **Memory** and **I/O** (reads and writes per second).

- Click a column heading to sort; your choice is remembered. The top 10 apps are shown; **Show all** lists every app.
- **Search** by app name, file name or PID.
- Click an app to see its processes, then click a process for details: path, user, whether it runs as administrator, start time, parent process, threads, handles and memory. Command lines are never shown, since they can contain passwords.
- **End task** (PIN-paired dashboards only, never in `-read-only` mode) ends a whole app or a single process after you confirm. Windows system processes such as `csrss`, `lsass`, `winlogon` and `svchost`, and PC Monitor itself, are protected. A server that isn't running as administrator can't end elevated programs.

### Customize the Dashboard

Click the **sliders icon** in the header to show or hide any section (CPU, memory, buttons, audio, power, drive health, processes, and so on). Hidden cards leave no gaps, and your choices are saved in this browser.

---

## 🚀 CI/CD Pipelines & Releases

This repository includes full GitHub Actions workflows configured in `.github/workflows/`:

1. **`ci.yml` (CI - Test & Build Matrix)**:
   - Validates `CHANGELOG.md` against [Common Changelog](https://common-changelog.org/) standards.
   - Runs the full Go test suite.
   - Cross-compiles binaries for a matrix of 6 targets: `windows/amd64`, `windows/arm64`, `linux/amd64`, `linux/arm64`, `darwin/arm64`, and `darwin/amd64`.
2. **`deploy-pages.yml` (GitHub Pages Deployment)**:
   - Automatically builds and deploys the `/web` static dashboard to GitHub Pages upon pushing to `main`.
3. **`release.yml` (Automated GitHub Releases)**: triggered on tag push (`v*`) or manually (`workflow_dispatch`, with a **dry run** option that builds and signs everything without publishing). All jobs use free GitHub-hosted runners:
   - **Linux runner:** Windows console + windowless tray app (`.zip`), Linux `.tar.gz`, `.deb` and `.rpm` (amd64/arm64), x86_64 AppImage with zsync update info.
   - **Linux ARM runner:** aarch64 AppImage.
   - **macOS runner:** macOS builds with the menu-bar icon (needs cgo).
   - **Windows runner:** per-user Inno Setup installers.
   - **Publish:** stamps the version into every build, writes `checksums.txt`, signs it with the `UPDATE_SIGNING_KEY` secret (`checksums.txt.sig`; the release fails without it), renders Homebrew/Scoop/winget manifests, attaches `winget_manifests_<version>.zip`, and publishes the release with notes from `CHANGELOG.md`.
   - **Package managers:** if a `PACKAGES_TOKEN` secret exists, the [Homebrew tap](https://github.com/garrettcorn/homebrew-tap) and [Scoop bucket](https://github.com/garrettcorn/scoop-bucket) are updated automatically. Create a free fine-grained personal access token with **Contents: Read and write** on those two repositories and save it as the `PACKAGES_TOKEN` Actions secret. winget manifests are attached for manual submission to `microsoft/winget-pkgs`.

**Signing keys:** generate a key pair with `go run ./cmd/signtool genkey`, store the private key as the `UPDATE_SIGNING_KEY` secret and put the public key in `pkg/update/update.go`. Keep a private backup of the private key: if it's lost, installed copies can't verify updates signed with a new key until they're updated manually.

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

Flags apply to `pc_monitor run` (the default command). They override the saved settings in `~/.pc_monitor/config.json`; add `--save` to keep them.

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-name` | Hostname | Friendly nickname for this server displayed in the client |
| `-port` | `9120` | Direct HTTP/WebSocket port for LAN or Tailscale IP connections (set `0` to disable) |
| `-tc-port` | `8080` | TCP port inside the Tailcat WireGuard tunnel |
| `-web-url` | `https://garrettcorn.github.io/remote-monitor/` | Base URL of the dashboard used in pairing links |
| `-settings-port` | `9121` | Port of the local settings page (bound to 127.0.0.1 only) |
| `-token` | *(saved)* | Custom pairing token. If omitted, a random 32-character hex token is created once and reused from `~/.pc_monitor/pairing_token` |
| `-no-tailcat`| `false` | Run in direct LAN / Tailscale IP mode only without Tailcat |
| `-export-html`| *(off)* | Also write a standalone HTML pairing card with the QR code to this file |
| `-background` | `false` | No console output; log to `~/.pc_monitor/server.log` (used by `start`, the tray and start-at-login) |
| `-save` | `false` | Save the given flags as the new defaults |
| `-read-only` | `false` | Monitoring only: disable remote control (audio, shutdown/restart); dashboards connect without a PIN |
| `-reset-pairing` | `false` | Revoke all paired dashboards: new pairing token and cleared PIN sessions |
| `-no-screen` | `false` | Disable screen capture previews and remote control (displays are still listed) |
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
│       ├── ui.js           # DOM renderer and SVG gauges
│       ├── processes.js    # Processes section (apps, search, details, End task)
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
- **Local settings page**: served only on `127.0.0.1`, rejects requests whose `Host` isn't localhost (DNS-rebinding protection), and every action needs the key in `~/.pc_monitor/settings_key` (owner-only). `pc_monitor settings` and the tray pass the key in the URL fragment, which browsers never send to servers.
- **Signed updates**: the self-updater installs a release only if `checksums.txt.sig` verifies against the ed25519 public key compiled into PC Monitor and the download's SHA-256 matches.
- **Screen previews and remote control are sensitive**: they're only available to PIN-paired dashboards, never in `-read-only` mode; screens are only captured on request, while auto-refresh is on or while the viewer is in Remote control. Use `-no-screen` to turn both off.
- **SMART details need elevation**: Drive health status works for any user; temperature, wear and error counters require running the server as administrator (Windows) or root with `smartctl` (Linux).
