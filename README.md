# 🖥️ PC Monitor & Multi-Server Telemetry Dashboard

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

## 🚀 Quick Start Guide

### Step 1: Run the Server

#### On Windows:
```powershell
# Run with default settings (Tailcat + Direct LAN port 9120)
.\pc_monitor_server.exe

# Or specify a friendly name and your GitHub Pages client URL:
.\pc_monitor_server.exe -name "GamingPC" -web-url "https://<your-username>.github.io/pc_monitor/"
```

#### On Linux:
```bash
chmod +x pc_monitor_linux_amd64
./pc_monitor_linux_amd64 -name "LinuxServer" -web-url "https://<your-username>.github.io/pc_monitor/"
```

#### On macOS:
```bash
chmod +x pc_monitor_darwin_arm64
./pc_monitor_darwin_arm64 -name "MacBookPro" -web-url "https://<your-username>.github.io/pc_monitor/"
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

---

## 🌐 Hosting on GitHub Pages

1. Push this repository to GitHub:
   ```bash
   git init
   git add .
   git commit -m "Initial commit of PC Monitor"
   git remote add origin https://github.com/<your-username>/pc_monitor.git
   git push -u origin main
   ```
2. Enable GitHub Pages:
   - Go to your repository **Settings** ➔ **Pages**.
   - Under **Build and deployment**, select **Source: GitHub Actions**.
3. The included GitHub Actions workflow (`.github/workflows/deploy.yml`) will automatically deploy the `/web` directory to:
   ```
   https://<your-username>.github.io/pc_monitor/
   ```

---

## 🛠️ CLI Flags Reference

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-name` | Hostname | Friendly nickname for this server displayed in the client |
| `-port` | `9120` | Direct HTTP/WebSocket port for LAN or Tailscale IP connections (set `0` to disable) |
| `-tc-port` | `8080` | TCP port inside the Tailcat WireGuard tunnel |
| `-web-url` | `https://your-username.github.io/pc_monitor/` | Base URL of your deployed GitHub Pages client |
| `-token` | *(auto-generated)* | Custom session token (random 32-character hex generated if omitted) |
| `-no-tailcat`| `false` | Run in direct LAN / Tailscale IP mode only without Tailcat |
| `-export-html`| `pairing_card.html`| Path to write a standalone HTML card with SVG QR code |
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
