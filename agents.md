# Agent Handoff: Remote Monitor

> Last updated: 2026-11-19  
> Repo: https://github.com/garrettcorn/remote-monitor  
> Pages: https://garrettcorn.github.io/remote-monitor/  
> Local workspace: `c:\Users\jon\workspace\pc_monitor\`

---

## What This Project Is

A cross-platform **PC system monitor** with two components:

| Component | Tech | Where |
|---|---|---|
| **Server** | Go binary (single static executable) | Runs on Windows / Linux / macOS / Android (Termux) / iOS |
| **Client** | 100% static web app (no build step) | GitHub Pages: `https://garrettcorn.github.io/remote-monitor/` |

The server collects system telemetry (CPU, GPU, thermals, RAM, disk, network, hostname, OS, processes) and exposes it over HTTP REST + WebSocket. Clients connect via **Tailcat** P2P WireGuard tunnels using a pairing code/link/QR code. Clients can connect to multiple servers simultaneously (fleet view in localStorage).

---

## Repository Layout

```
pc_monitor/
├── cmd/
│   ├── server/main.go              # Server entrypoint — CLI flags, Tailcat tunnel, QR + pairing output
│   └── changelog/
│       ├── main.go                 # validate / extract / bump subcommands for CHANGELOG.md
│       └── changelog_test.go
├── pkg/
│   ├── telemetry/
│   │   ├── types.go                # All JSON types for SystemSnapshot
│   │   ├── collector.go            # GetSnapshot() — platform-agnostic shell
│   │   ├── collector_windows.go    # WMI, registry, Windows APIs
│   │   ├── collector_linux.go      # /proc, /sys/class/thermal
│   │   ├── collector_darwin.go     # sysctl, vm_stat, sw_vers (exec-based, cross-compiles)
│   │   └── collector_fallback.go   # build tag: !windows && !linux && !darwin
│   ├── tailcat/
│   │   └── tunnel.go               # StartTunnel() — key persistence, DERP bootstrap
│   ├── pairing/
│   │   ├── pairing.go              # Token gen, URL builder, HTML card export
│   │   ├── qrcode.go               # ANSI terminal QR + SVG renderer
│   │   ├── qr_encode.go            # Pure Go QR ISO 18004 matrix encoder
│   │   └── qrcode_test.go
│   └── api/
│       ├── server.go               # HTTP REST + WebSocket, CORS, token auth
│       └── server_test.go
├── web/                            # Static client — deployed to GitHub Pages AS-IS (no build step)
│   ├── index.html
│   ├── style.css
│   ├── favicon.svg
│   └── js/
│       ├── app.js                  # App bootstrap
│       ├── storage.js              # localStorage fleet persistence
│       ├── client.js               # WS + HTTP polling + Tailcat WASM
│       ├── ui.js                   # DOM updates, gauges
│       ├── charts.js               # Canvas sparklines (60-second history)
│       └── qr-scanner.js           # BarcodeDetector API — camera QR scan
├── scripts/
│   ├── build.ps1 / build.sh        # Local cross-compile scripts
│   └── release.ps1 / release.sh   # Local release packaging scripts
├── .github/workflows/
│   ├── ci.yml                      # Runs tests on push/PR to main
│   ├── deploy-pages.yml            # Deploys web/ to GitHub Pages on push to main
│   └── release.yml                 # Builds 6 binaries + packages + publishes GitHub Release
├── go.mod                          # module: pc_monitor, go 1.27.1
├── go.sum
├── CHANGELOG.md                    # Common Changelog format (https://common-changelog.org/)
├── README.md
└── agents.md                       # ← this file
```

> [!NOTE]
> `dist/`, `pairing_card.html`, `pairing_info.json`, `pc_monitor_server.exe`, `main`, and `v1.0.0` in the root are local build artifacts — not committed (covered by `.gitignore`).

---

## Key Technologies

| Technology | Version / Detail |
|---|---|
| Go | 1.27.1 (`go.mod`), CI uses `go-version: '1.27'` with `check-latest: true` |
| Tailcat (P2P tunneling) | `github.com/tailscale/tailcat v0.7.0` — in local module cache at `C:\Users\jon\go\pkg\mod\github.com\tailscale\tailcat@v0.7.0` |
| `tailscale.com` | `v1.103.0-pre.0.20260916030321-a2263542f260` (indirect, pulled by tailcat) |
| `github.com/coder/websocket` | `v1.8.14` — WebSocket server |
| `golang.org/x/sys` | `v0.48.0` — low-level OS syscalls |
| GitHub Actions | ci.yml, deploy-pages.yml, release.yml |
| GitHub Pages | Served from `web/` directory via Actions |
| Common Changelog | https://common-changelog.org/ — `Added/Changed/Removed/Fixed` only, imperative mood, dated `YYYY-MM-DD` |

---

## Git State (as of 2026-11-19)

```
ef38c06  Enable automatic Pages enablement in deploy-pages workflow  ← HEAD / main
75ecd12  Configure remote repository URL and GitHub Pages endpoint
7d9a865  Add GitHub CI/CD pipelines, release automation, and Common Changelog tooling
81fe331  Initial commit of PC Monitor client and server
f0525ac  Initial commit (GitHub auto-init)
```

- **Remote:** `https://github.com/garrettcorn/remote-monitor.git`
- **Local tag `v1.0.0`** — re-created at HEAD (`ef38c06`) on 2026-11-19 after deleting the old one that pointed to `75ecd12`

---

## CI/CD Workflows

### `ci.yml` — runs on push/PR to `main`
- Sets up Go, runs `go test ./...`

### `deploy-pages.yml` — runs on push to `main`
- Uploads `web/` as the Pages artifact
- Uses `actions/configure-pages@v5` with `enablement: true` (required — without it the action fails with "Not Found" if Pages wasn't already set to Actions mode)

### `release.yml` — runs on `push` to a `v*` tag OR `workflow_dispatch`
- `workflow_dispatch` input: `tag` (string), `draft` (bool), `prerelease` (bool)
- Steps: validate CHANGELOG → extract release notes → cross-compile 6 targets → package archives → sha256 checksums → `softprops/action-gh-release@v2`

**6 build targets:**

| Binary name | GOOS | GOARCH |
|---|---|---|
| `pc_monitor_windows_amd64.exe` | windows | amd64 |
| `pc_monitor_windows_arm64.exe` | windows | arm64 |
| `pc_monitor_linux_amd64` | linux | amd64 |
| `pc_monitor_linux_arm64` | linux | arm64 |
| `pc_monitor_darwin_amd64` | darwin | amd64 |
| `pc_monitor_darwin_arm64` | darwin | arm64 |

---

## Outstanding Items (as of 2026-11-19)

### 🔴 Must Do — v1.0.0 Release Binaries

The `v1.0.0` release on GitHub only has source code zips. The release workflow was not present when the tag was originally pushed. The local tag has been re-created at HEAD. **You still need to push the tag:**

```powershell
cd c:\Users\jon\workspace\pc_monitor

# Delete the old remote tag
git push origin :refs/tags/v1.0.0

# Push the corrected tag (triggers release.yml → builds binaries)
git push origin v1.0.0
```

After ~5 min the release at https://github.com/garrettcorn/remote-monitor/releases/tag/v1.0.0 should have 7 assets (6 archives + `checksums.txt`).

**Alternative:** Manually trigger via GitHub UI → Actions → Release → Run workflow → enter `v1.0.0`.

### 🟡 Nice to Have

| Item | Detail |
|---|---|
| Rename Go module | `go.mod` says `module pc_monitor`; could be `github.com/garrettcorn/remote-monitor` for consistency. Requires updating all internal import paths. |
| Verify CI runs | Check the Actions tab to confirm `ci.yml` ran green for `ef38c06` |
| Android / iOS docs | The binary runs in Termux (Android) with `linux/arm64` build; iOS is not yet documented |
| GPU support depth | Windows uses WMI (`Win32_VideoController`); Linux reads `/sys/class/drm`; macOS uses `system_profiler`. NVIDIA/AMD NVML integration not yet added. |
| Process list in UI | `pkg/telemetry` collects top processes but the web UI doesn't yet show them in a table |
| WASM Tailcat client | `web/js/client.js` references a Tailcat WASM module for in-browser P2P — this is a stub and needs the actual WASM artifact |

---

## How to Work on This Locally

### Run tests
```powershell
cd c:\Users\jon\workspace\pc_monitor
go test ./...
```

### Build server for current platform
```powershell
go build -o pc_monitor_server.exe ./cmd/server
```

### Cross-compile all targets
```powershell
.\scripts\build.ps1
# or on Linux/macOS:
bash scripts/build.sh
```

### Run the server
```powershell
.\pc_monitor_server.exe -port 8080 -name "My PC"
# Outputs: pairing code, URL, and ANSI QR code
```

### Add a new release version
1. Add an entry to `CHANGELOG.md` following Common Changelog format
2. Validate: `go run ./cmd/changelog validate`
3. Commit, tag, push:
   ```powershell
   git tag -a v1.x.x -m "Release v1.x.x"
   git push origin v1.x.x
   ```
4. GitHub Actions builds and publishes automatically

---

## Architecture Notes

### Pairing Flow
```
Server starts
  → generates random token + node keypair
  → registers with Tailcat DERP relay
  → prints: pairing code (8 chars), URL (https://garrettcorn.github.io/remote-monitor/#pair=<token>&addr=<tailcat-addr>), ANSI QR
Client opens URL
  → reads token + addr from URL hash fragment (never sent to server)
  → connects to server over Tailcat WireGuard tunnel using token as auth header
  → begins WebSocket stream of SystemSnapshot JSON
```

### API Endpoints (pkg/api/server.go)
| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/snapshot` | Bearer token | Full SystemSnapshot JSON |
| `GET` | `/api/ws` | Bearer token (query param) | WebSocket — streams snapshots |
| `GET` | `/api/health` | None | `{"status":"ok"}` |
| `GET` | `/api/pair` | None | Pairing info (token redacted after first use) |

### SystemSnapshot JSON shape (pkg/telemetry/types.go)
```json
{
  "hostname": "...", "os": "...", "platform": "...", "uptime": 0,
  "cpu": { "model": "...", "cores": 0, "threads": 0, "usage_pct": [], "freq_mhz": [] },
  "gpu": [{ "name": "...", "usage_pct": 0, "vram_total_mb": 0, "vram_used_mb": 0 }],
  "thermals": [{ "name": "...", "temp_c": 0 }],
  "ram": { "total_mb": 0, "used_mb": 0, "free_mb": 0 },
  "disks": [{ "mount": "/", "total_gb": 0, "used_gb": 0, "free_gb": 0, "fs": "..." }],
  "network": [{ "interface": "...", "ip4": "...", "ip6": "...", "rx_bps": 0, "tx_bps": 0 }],
  "processes": [{ "pid": 0, "name": "...", "cpu_pct": 0, "mem_mb": 0 }],
  "collected_at": "2026-11-19T00:00:00Z"
}
```

### Web Client Storage (web/js/storage.js)
Servers are persisted in `localStorage` under key `remote_monitor_servers` as a JSON array:
```json
[{ "id": "uuid", "name": "My PC", "url": "https://...", "token": "...", "addedAt": "..." }]
```

---

## Known Gotchas

| Gotcha | Detail |
|---|---|
| `go.sum` tailcat hashes | Were hand-computed from the local module cache (`h1:` hashes). If `go mod tidy` removes them, re-add from the error output of `go build`. |
| `collector_darwin.go` cross-compile | Uses `exec.Command("sysctl", ...)` instead of `syscall.Uname` — `syscall.Utsname` is not defined when cross-compiling from Windows. Do not revert to syscall. |
| `configure-pages@v5` | Requires `enablement: true` in `deploy-pages.yml` or it fails with HTTP 404 "Not Found" on first run. Already set. |
| `workflow_dispatch` release | When triggering manually, `softprops/action-gh-release@v2` will **update** (not duplicate) an existing release with the same tag. Safe to re-run. |
| Go version in CI | `go-version: '1.27'` with `check-latest: true` — `go.mod` says `go 1.27.1`. If Go 1.27 isn't yet on the runner, `check-latest` pulls the latest 1.27.x patch. |
| Module name mismatch | `go.mod` says `module pc_monitor` (not `github.com/garrettcorn/remote-monitor`). Internal imports use `pc_monitor/pkg/...`. Changing this requires a find-replace across all `.go` files. |
