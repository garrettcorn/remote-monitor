# Agent Handoff: Remote Monitor

> Last updated: 2026-11-19 (after release **v1.2.1**)  
> Repo: https://github.com/garrettcorn/remote-monitor  
> Pages: https://garrettcorn.github.io/remote-monitor/  
> Local workspace: `c:\Users\jon\workspace\pc_monitor\`

---

## What This Project Is

A cross-platform **PC system monitor and remote control** with two components:

| Component | Tech | Where |
|---|---|---|
| **Server** | Go binary (single static executable) | Windows / Linux / macOS (Android via Termux `linux/arm64`) |
| **Client** | 100% static web app (no build step) | GitHub Pages: `https://garrettcorn.github.io/remote-monitor/` |

The server collects telemetry (CPU incl. per-core, GPU, thermals, RAM, drives + I/O, physical drive health/SMART, network + per-adapter rates, processes, connected devices, audio devices, battery, signed-in users) and serves it over HTTP REST + WebSocket. By default it also allows **remote control** (audio volume/mute/default device, Stream Deck-style button actions, shutdown/restart), gated by a **one-time PIN** shown on the server console. `-read-only` disables control. The dashboard has a **Control Panel** (button grid + audio + power) and a **Customize** dialog to show/hide sections.

Dashboards pair via a link/QR code containing the server's pairing token and direct URL. The browser stores paired servers in `localStorage` (fleet view supports multiple servers).

**Platform depth:** Windows is the primary, fully tested platform. Linux and macOS compile in CI and have best-effort implementations that have **not** been run on real hardware.

---

## Repository Layout

```
pc_monitor/
├── cmd/
│   ├── server/main.go              # Entrypoint: flags, pairing state (~/.pc_monitor), PIN, Tailcat, QR output
│   └── changelog/                  # validate / extract / bump for CHANGELOG.md (+ tests)
├── pkg/
│   ├── api/
│   │   ├── server.go               # REST + WebSocket, CORS (+ Private Network Access), token/PIN-session auth, control endpoints
│   │   └── server_test.go          # read-only mode, PIN pairing, control auth, PIN rotation (power is stubbed)
│   ├── control/
│   │   ├── control.go              # One-time PIN, persisted session hashes, token persistence, atomic file writes
│   │   └── control_test.go
│   ├── audio/                      # Audio endpoints: list/volume/mute/default device
│   │   ├── audio.go                # Shared types + API
│   │   ├── audio_windows.go        # Core Audio via raw COM vtables; IPolicyConfig for default device
│   │   ├── volume_windows_amd64.go # SetMasterVolumeLevelScalar (float arg works on amd64)
│   │   ├── volume_windows_other.go # arm64: volume via VolumeStepUp/Down (no float args in SyscallN)
│   │   ├── audio_linux.go          # pactl (JSON output, pactl 16+)
│   │   ├── audio_darwin.go         # osascript: system output/input volume, output mute only
│   │   └── audio_other.go          # unsupported
│   ├── power/power.go              # shutdown/restart (Windows `shutdown /t 5`; systemctl; osascript)
│   ├── deck/                       # Stream Deck button actions
│   │   ├── deck.go                 # Action types, hotkey parsing, URL validation, launch allowlist (actions.json)
│   │   ├── input_windows.go        # SendInput (media keys, hotkeys), LockWorkStation, ShellExecute
│   │   ├── input_linux.go          # xdotool / loginctl / xdg-open (X11)
│   │   ├── input_darwin.go         # osascript System Events / pmset / open (no media keys)
│   │   └── deck_test.go
│   ├── telemetry/
│   │   ├── types.go                # SystemSnapshot and all JSON types
│   │   ├── collector.go            # Sampling loop, audio refresh, slow background refresh (30 s)
│   │   ├── collector_<os>.go       # CPU, memory, drives (+I/O), network, GPU, temps, processes per OS
│   │   ├── extra_<os>.go           # battery, physical disks/SMART, connected devices, users per OS
│   │   ├── pdh_windows.go          # Reusable PDH perf-counter query (disk I/O)
│   │   └── ps_unix.go              # Shared `ps` / `who` parsing (linux, darwin)
│   ├── pairing/                    # Token gen, pairing URL, HTML card, pure-Go QR encoder + ANSI/SVG render
│   └── tailcat/tunnel.go           # StartTunnel(): key persistence, DERP bootstrap
├── web/                            # Static client, deployed to GitHub Pages as-is
│   ├── index.html                  # Asset URLs carry ?v=N cache-busters (currently v=10); sections tagged data-section="…"
│   ├── style.css                   # "Liquid Glass" theme, light + dark
│   └── js/ app.js, client.js, ui.js, panel.js (Customize + Stream Deck), storage.js, charts.js, qr-scanner.js
├── scripts/                        # build.ps1/.sh (cross-compile), release.ps1/.sh (local packaging)
├── .github/workflows/              # ci.yml, deploy-pages.yml, release.yml
├── CHANGELOG.md                    # Common Changelog format
├── README.md
└── agents.md                       # ← this file
```

> [!NOTE]
> Local leftovers in the repo root (`main`, `v1.0.0`, `FETCH_HEAD`, `release_notes.md`) are untracked and **not** in `.gitignore`. Never `git add .`; stage files explicitly. `.gitignore` covers `*.exe`, `dist/`, `pairing_info.json`, `pairing_card.html`, `*.key`, `*.log`, `*.tmp`.

---

## Key Technologies

| Technology | Version / Detail |
|---|---|
| Go | 1.27.1 (`go.mod`); CI uses `go-version: '1.27'` with `check-latest: true` |
| Tailcat (P2P tunneling) | `github.com/tailscale/tailcat v0.7.0` |
| `github.com/coder/websocket` | `v1.8.14` |
| `golang.org/x/sys` | `v0.48.0` (Windows APIs: `GetIfEntry2Ex`, `NtQuerySystemInformation`, Toolhelp, WTS, COM init) |
| GitHub Actions | `ci.yml`, `deploy-pages.yml`, `release.yml` (`softprops/action-gh-release@v3`) |
| Common Changelog | `Changed/Added/Removed/Fixed` only, imperative mood, `## [x.y.z] - YYYY-MM-DD`, link refs at bottom |

No new third-party Go dependencies were added for audio/SMART/devices: Windows uses raw syscalls/COM, PDH, and PowerShell (`Get-PhysicalDisk`, `Get-PnpDevice`).

---

## Release History

| Tag | Highlights |
|---|---|
| `v1.0.0` | Initial release (assets built and published) |
| `v1.1.0` | QR encoder fixed (Reed-Solomon ECC, v1–40), dashboard render crash fixed, real Windows network counters, real per-core CPU / uptime / process stats / ACPI thermal zone, fake temperatures removed |
| `v1.2.0` | Liquid Glass UI; **PIN-gated remote control** (audio, shutdown/restart), `-read-only`; drive health/SMART; connected devices; disk I/O, adapter rates/link speed, GPU fan/clocks/power limit, battery, users; HTML escaping of host-provided names |
| `v1.2.1` | **Pairing persists across restarts** (token + session hashes in `~/.pc_monitor`), `-reset-pairing`, single pairing QR code |
| `v1.3.0` | Control Panel with Stream Deck-style buttons (`pkg/deck`, `/api/deck/*`), Customize dialog to show/hide sections |

All released tags have 7 release assets.

---

## CI/CD Workflows

### `ci.yml` (push/PR to `main`)
- Job 1: validate CHANGELOG + `go test ./...`
- Job 2: matrix cross-build of the 6 targets

### `deploy-pages.yml` (push to `main`)
- Uploads `web/` as the Pages artifact; `actions/configure-pages@v5` with `enablement: true`
- **Has failed transiently once** with "No artifacts named github-pages were found" (GitHub-side). Fix: `gh run rerun <id>`. After any web change, confirm the live site serves the new `?v=N` (e.g. `curl -s https://garrettcorn.github.io/remote-monitor/ | grep app.js`).

### `release.yml` (push of a `v*` tag, or `workflow_dispatch`)
- Validate CHANGELOG → extract notes for `${TAG#v}` → cross-compile → package → checksums → publish
- Assets: `pc_monitor_<ver>_{windows_amd64,windows_arm64}.zip`, `pc_monitor_<ver>_{linux,darwin}_{amd64,arm64}.tar.gz`, `checksums.txt` (binaries inside are named `pc_monitor_<os>_<arch>[.exe]`)

---

## How to Work on This Locally

```powershell
cd c:\Users\jon\workspace\pc_monitor
go test ./...                                   # unit tests (api, control, deck, pairing, changelog)
go vet ./...                                    # also run with GOOS=linux/darwin, GOARCH=arm64
go build -o pc_monitor_server.exe ./cmd/server
.\pc_monitor_server.exe -name "My PC"          # prints pairing link, QR code and control PIN
```

### Testing safely
- **Never trigger shutdown/restart on the dev machine.** `pkg/api/server_test.go` stubs `Server.doPower`; keep it that way.
- Audio tests should **restore** volume/mute/default device afterwards; prefer the microphone over the speakers.
- The server writes pairing state to `~/.pc_monitor`. To test without touching the real state, start it with `USERPROFILE` (Windows) / `HOME` pointed at a scratch folder.
- To preview the dashboard against a local server, serve `web/` over `http://127.0.0.1` (not `file://`), and open `/#direct=http%3A%2F%2F127.0.0.1%3A<port>&name=x&token=<token>`.
- The HTTPS Pages site calling a plain-HTTP server is blocked as mixed content in default Chromium settings; the user's own browser allows it. Don't "fix" this unless asked.

### Release a new version
1. Add `## [x.y.z] - YYYY-MM-DD` to `CHANGELOG.md` (+ `[x.y.z]: …/releases/tag/vx.y.z` link ref)
2. `go run ./cmd/changelog validate` and `go run ./cmd/changelog extract -o <tmp> x.y.z`
3. Commit, then `git tag -a vx.y.z -m "Release vx.y.z"` and `git push origin main` + `git push origin vx.y.z`
4. Watch: `gh run list --repo garrettcorn/remote-monitor --limit 3`, then `gh release view vx.y.z`

---

## Architecture Notes

### Server CLI flags (`cmd/server/main.go`)
| Flag | Default | Notes |
|---|---|---|
| `-name` | hostname | Display name |
| `-port` | `9120` | Direct HTTP/WS port (plain HTTP); `0` disables |
| `-tc-port` | `8080` | Port inside the Tailcat tunnel |
| `-web-url` | Pages URL | Base of the pairing link |
| `-token` | saved | Overrides the persisted pairing token |
| `-no-tailcat` | `false` | Direct mode only |
| `-export-html` | `pairing_card.html` | Pairing card with SVG QR (`""` to skip) |
| `-read-only` | `false` | Disable control; no PIN needed |
| `-reset-pairing` | `false` | Delete saved token + sessions (revokes all dashboards) |
| `-verbose` | `false` | Debug logging |

### Pairing & access control
```
Server start
  → token: -token, else ~/.pc_monitor/pairing_token, else generate + save (0600, atomic write)
  → sessions: ~/.pc_monitor/sessions.json (SHA-256 hashes only)
  → prints pairing link + ONE terminal QR + (unless -read-only) the control PIN
Dashboard opens link (#direct=…&name=…&port=…&tc=…&tcport=…&token=…)
  → GET /api/health → control_enabled / pin_required
  → if pin_required and no saved session: PIN dialog → POST /api/pair {pin} → {session}
  → WS /ws?token=…&session=…  (falls back to HTTP polling /api/telemetry)
PIN: 4 digits (crypto/rand), constant-time compare, one-time; rotated after use or 5 failures;
     new PIN printed on the server console only. 1 s delay after a wrong PIN.
```
- `401 {"error":"pin_required"}` → dashboard reopens the PIN dialog.
- `401 {"error":"unauthorized"}` (token rejected, e.g. after `-reset-pairing`) → dashboard shows "Pairing revoked" and stops retrying.
- Read-only: telemetry needs only the token; control endpoints return `403 {"error":"read_only"}`.

### API endpoints (`pkg/api/server.go`)
| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/health` | none | `status`, `server_name`, `control_enabled`, `pin_required` |
| `POST` | `/api/pair` | token | `{pin}` → `{session}` |
| `GET` | `/api/telemetry` | token (+ session if control) | Full `SystemSnapshot` |
| `GET` | `/ws` | token (+ session) via query | Streams snapshots every 1.2 s |
| `POST` | `/api/audio/volume` | token + session | `{id, volume 0-100}` → fresh audio state |
| `POST` | `/api/audio/mute` | token + session | `{id, muted}` → fresh audio state |
| `POST` | `/api/audio/default` | token + session | `{id}` → fresh audio state |
| `POST` | `/api/power` | token + session | `{action: "shutdown"\|"restart"}` |
| `GET` | `/api/deck/actions` | token + session | Launch allowlist (`id`, `name` only), `actions_file`, `media_keys` |
| `POST` | `/api/deck/press` | token + session | `{type: media\|hotkey\|lock\|mic_mute\|url\|launch, key?, keys?, url?, id?}` |

Token: `?token=`, `X-Pairing-Token`, or `Authorization: Bearer`. Session: `X-Session` or `?session=`.

### SystemSnapshot (`pkg/telemetry/types.go`, abridged)
```json
{
  "timestamp": "…",
  "host": { "hostname", "os", "os_version", "architecture", "kernel", "uptime_sec", "boot_time", "process_count", "server_time" },
  "cpu": { "model_name", "vendor", "physical_cores", "logical_cores", "usage_percent", "core_usages": [], "frequency_mhz" },
  "memory": { "total_bytes", "used_bytes", "available_bytes", "usage_percent", "swap_*" },
  "gpus": [{ "name", "vendor", "utilization_percent", "vram_*", "temperature_c", "power_watts", "power_limit_watts", "fan_speed_percent", "core_clock_mhz", "mem_clock_mhz" }],
  "temperatures": [{ "label", "sensor_type", "temperature_c" }],
  "drives": [{ "device", "mountpoint", "filesystem", "total_bytes", "free_bytes", "usage_percent", "read_speed_bps", "write_speed_bps", "busy_percent" }],
  "network": { "interfaces": [{ "name", "mac", "ipv4", "ipv6", "is_up", "speed_recv_bps", "speed_sent_bps", "link_speed_mbps" }], "download_speed_bps", "upload_speed_bps", "total_bytes_*" },
  "top_processes": [{ "pid", "name", "cpu_percent", "memory_percent", "memory_bytes" }],
  "physical_disks": [{ "name", "media_type", "bus_type", "size_bytes", "health", "temperature_c?", "wear_percent?", "power_on_hours?", "read_errors?", "write_errors?" }],
  "smart_details_available": false,
  "devices": [{ "name", "class", "connection", "status" }],
  "audio": { "outputs": [{ "id", "name", "default", "volume", "muted" }], "inputs": [], "can_switch": true },
  "battery": { "percent", "charging", "plugged_in", "seconds_left", "state" },
  "users": [{ "user", "session", "state" }]
}
```

### Windows data sources
| Data | Source |
|---|---|
| Per-core CPU | `NtQuerySystemInformation(SystemProcessorPerformanceInformation)` deltas |
| Uptime | `GetTickCount64` |
| Processes | Toolhelp snapshot + `GetProcessTimes` deltas + `K32GetProcessMemoryInfo` |
| Network bytes / link speed | `GetIfEntry2Ex` (`MIB_IF_ROW2`) |
| Disk I/O | PDH `\LogicalDisk(*)\Disk Read/Write Bytes/sec`, `% Idle Time` |
| Thermals | GPU (nvidia-smi) + PDH `\Thermal Zone Information(*)\Temperature` (Kelvin) |
| GPU | `nvidia-smi --query-gpu=…` (registry fallback for non-NVIDIA, name only) |
| Drive health / SMART | PowerShell `Get-PhysicalDisk` (+ `Get-StorageReliabilityCounter`, admin only) |
| Devices | PowerShell `Get-PnpDevice` + per-device `DEVPKEY_Device_BusReportedDeviceDesc` |
| Users | `WTSEnumerateSessions` + `WTSQuerySessionInformationW` |
| Battery | `GetSystemPowerStatus` |
| Audio | Core Audio COM (`IMMDeviceEnumerator`, `IAudioEndpointVolume`, `IPolicyConfig`) |

PowerShell sources are slow (~1–4 s) and run in a background refresh every 30 s (`Collector.refreshSlow`), never on the 1.2 s sampling path.

### Stream Deck buttons (`pkg/deck`, `web/js/panel.js`)
- Action types: `media` (play_pause/next/previous/stop/volume_up/volume_down/volume_mute), `hotkey` (`ctrl+shift+m`: modifiers ctrl/shift/alt/win + one key a–z, 0–9, f1–f24 or named), `lock`, `mic_mute`, `url` (http/https only), `launch` (id from `~/.pc_monitor/actions.json`; paths never leave the server).
- Button layouts are per server in the browser; defaults are 8 media/volume/mic/lock keys.
- Hotkeys can type arbitrary input on the host, so they are as trusted as shutdown: PIN session required, disabled in read-only.

### Customize dialog (`Layout` in `web/js/panel.js`)
- Sections are `[data-section]` elements; `SECTION_GROUPS` defines the dialog. Hidden ids live in `localStorage['pc_monitor_hidden_sections']`.
- `Layout.apply(controlEnabled)` also hides `.control-only` parts on read-only servers and reflows rows (`grid-single`, `cp-single`) so hidden cards leave no gaps. Call it after any layout change.

### Web client storage
- `localStorage['pc_monitor_servers_v1']` = array of `{ id, name, directURL, directPort, tailcatAddr, tailcatPort, token, session, controlEnabled, status, lastSnapshot (trimmed) }`; active server id in `pc_monitor_active_server_id`.
- `pc_monitor_deck_<serverId>` = button layout; `pc_monitor_hidden_sections` = hidden section ids.

---

## Outstanding Items / Ideas

| Item | Detail |
|---|---|
| Linux/macOS verification | Audio (pactl/osascript), SMART (smartctl), devices (lsusb), battery, users and disk I/O are untested on real machines |
| macOS gaps | No drive health or device list; no audio device switching |
| Non-NVIDIA GPUs | Windows registry fallback gives name only; no AMD/Intel utilisation |
| CPU core temperature (Windows) | Needs admin or a driver (e.g. LibreHardwareMonitor); only ACPI zone + GPU today |
| Fallback platform | `collector_fallback.go` still reports a fixed 10% CPU |
| Plain HTTP direct port | PIN/session cross the LAN unencrypted; Tailscale/Tailcat recommended. Consider TLS. |
| WASM Tailcat client | `window.tailcatDial` is never defined; the Tailcat path in `client.js` is a stub, so connections use the direct URL |
| Rename Go module | `module pc_monitor` → `github.com/garrettcorn/remote-monitor` (find-replace imports) |
| Node.js 20 deprecation | Actions warn that checkout/configure-pages/deploy-pages/upload-artifact target Node 20 (forced to 24); bump when newer majors exist |

---

## Known Gotchas

| Gotcha | Detail |
|---|---|
| BOM in JSON config | PowerShell `Set-Content -Encoding utf8` and some editors add a UTF-8 BOM; `deck.LoadLaunchActions` strips it. Do the same for any new config file |
| `SendInput` struct | `INPUT` must be 40 bytes on 64-bit (union sized by `MOUSEINPUT`); `deck_test.go` guards the layout |
| Testing button actions | Only press harmless keys on the dev machine (e.g. a hotkey for `F24`); never media keys, launch or URL buttons while the user is working |
| Float args in Windows syscalls | Go's amd64 Windows call path mirrors int args into XMM regs (floats OK); **arm64 does not** → use the step-based volume path |
| COM out-params | Allocate with `//go:noinline` helpers (heap) so Go stack moves can't invalidate pointers passed to COM |
| `Get-PnpDeviceProperty` arrays | Passing `-InstanceId` as an array returns mismatched IDs; query one device at a time |
| `% Idle Time` | Sleeping/stalled drives report 0% idle with no transfers; busy % is only reported while bytes move |
| Dashboard cache | GitHub Pages sends `max-age=600`. Bump the `?v=N` on `style.css`, `js/app.js` **and** the `client.js`/`ui.js` imports in `app.js` when changing web files |
| Backdrop blur flicker | Don't put `backdrop-filter` on the live-updating cards or animate the background; it flickers in Chromium. Blur is limited to header, dialog and toast |
| Snapshot render errors | A throw in `UI.renderSnapshot` stops all later sections; `client.js` logs render errors as `console.error` "Failed to render snapshot" |
| Re-renders and refs | Cards re-render every ~1.2 s; audio re-render pauses ~2.5 s after user interaction (`audioHoldUntil`) |
| `go.sum` tailcat hashes | Hand-computed originally; if `go mod tidy` drops them, re-add from `go build` errors |
| `collector_darwin.go` | Uses `exec.Command("sysctl", …)`, not `syscall.Uname` (doesn't cross-compile from Windows) |
| `configure-pages@v5` | Needs `enablement: true` (already set) |
| `gofmt` | `cmd/changelog/main.go` has a pre-existing formatting nit; avoid unrelated reformat churn in commits |
| Module name | `module pc_monitor`; imports use `pc_monitor/pkg/...` |
