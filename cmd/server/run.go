package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"pc_monitor/pkg/api"
	"pc_monitor/pkg/buildinfo"
	"pc_monitor/pkg/config"
	"pc_monitor/pkg/control"
	"pc_monitor/pkg/pairing"
	"pc_monitor/pkg/service"
	"pc_monitor/pkg/tailcat"
	"pc_monitor/pkg/telemetry"
	"pc_monitor/pkg/update"
)

// guiBuild is set to "1" for the windowless Windows app (-H windowsgui).
var guiBuild = ""

func guiApp() bool { return guiBuild == "1" }

type runExtras struct {
	token        string
	resetPairing bool
	background   bool
	verbose      bool
	save         bool
}

// runFlags binds flags to cfg so they override saved settings.
func runFlags(cfg *config.Config) (*flag.FlagSet, *runExtras) {
	if cfg == nil {
		d := config.Defaults()
		cfg = &d
	}
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	x := &runExtras{}
	fs.StringVar(&cfg.Name, "name", cfg.Name, "Friendly name for this server in the dashboard")
	fs.IntVar(&cfg.Port, "port", cfg.Port, "Direct HTTP/WebSocket port for LAN/Tailscale IP access (0 to disable)")
	fs.IntVar(&cfg.TailcatPort, "tc-port", cfg.TailcatPort, "Tailcat tunnel port")
	fs.StringVar(&cfg.WebURL, "web-url", cfg.WebURL, "URL of the dashboard used in pairing links")
	fs.BoolVar(&cfg.NoTailcat, "no-tailcat", cfg.NoTailcat, "Disable the Tailcat tunnel (direct LAN mode only)")
	fs.StringVar(&cfg.ExportHTML, "export-html", cfg.ExportHTML, "Save a printable pairing card to this HTML file")
	fs.BoolVar(&cfg.ReadOnly, "read-only", cfg.ReadOnly, "Monitoring only: disable remote control; no PIN needed")
	fs.BoolVar(&cfg.NoScreen, "no-screen", cfg.NoScreen, "Disable screen capture previews")
	fs.IntVar(&cfg.SettingsPort, "settings-port", cfg.SettingsPort, "Port of the local settings page (127.0.0.1 only)")
	fs.StringVar(&x.token, "token", "", "Pairing token (default: saved in ~/.pc_monitor)")
	fs.BoolVar(&x.resetPairing, "reset-pairing", false, "Revoke all paired dashboards: new pairing token and PIN sessions")
	fs.BoolVar(&x.background, "background", false, "Run without console output (logs to ~/.pc_monitor/server.log)")
	fs.BoolVar(&x.verbose, "verbose", false, "Enable verbose debug output")
	fs.BoolVar(&x.save, "save", false, "Save the given flags as the new defaults")
	return fs, x
}

// server holds the running state shared with the settings page.
type server struct {
	cfg          config.Config
	token        string
	tailcatAddr  string
	localIPs     []string
	pairingURL   string
	svgQR        string
	pin          *control.PIN
	sessions     *control.Sessions
	updates      *update.Checker
	settingsKey  string
	closers      []func()
	stopOnce     sync.Once
	stopped      chan struct{}
	restartAfter bool
}

func runCommand(args []string) error {
	cfg, cfgErr := config.Load()
	fs, x := runFlags(&cfg)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if x.background {
		redirectToLog()
	}
	if cfgErr != nil {
		fmt.Printf("Warning: %v (using defaults)\n", cfgErr)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if x.save {
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save settings: %w", err)
		}
		fmt.Println("Settings saved to", config.Path())
	}
	update.CleanupOld()

	if running, ver := pingSettings(cfg.SettingsPort); running {
		return fmt.Errorf("PC Monitor %s is already running (use `pc_monitor stop` or `pc_monitor settings`)", ver)
	}

	s := &server{cfg: cfg, stopped: make(chan struct{})}
	return s.run(x)
}

// redirectToLog sends output to ~/.pc_monitor/server.log (rotated at 5 MB).
func redirectToLog() {
	_ = os.MkdirAll(config.Dir(), 0o700)
	if fi, err := os.Stat(service.LogPath()); err == nil && fi.Size() > 5<<20 {
		_ = os.Rename(service.LogPath(), service.LogPath()+".1")
	}
	f, err := os.OpenFile(service.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	os.Stdout, os.Stderr = f, f
	fmt.Printf("\n----- %s PC Monitor %s starting (background) -----\n", time.Now().Format(time.RFC3339), buildinfo.Version)
}

func (s *server) run(x *runExtras) error {
	cfg := s.cfg
	fmt.Println("\n========================================================================")
	fmt.Println("  PC MONITOR SERVER " + buildinfo.Version + " - System Telemetry & Remote Control")
	fmt.Println("========================================================================")

	stateDir := config.Dir()
	tokenFile := filepath.Join(stateDir, "pairing_token")
	sessionsFile := filepath.Join(stateDir, "sessions.json")
	if x.resetPairing {
		if err := resetPairingFiles(); err != nil {
			return err
		}
		fmt.Println("  Pairing reset: all previously paired dashboards have been revoked.")
	}

	// Pairing token: flag, else saved, else create and save.
	s.token = x.token
	if s.token == "" {
		var err error
		s.token, _, err = control.LoadOrCreateToken(tokenFile, func() (string, error) { return pairing.GenerateToken(16) })
		if err != nil {
			fmt.Printf("Warning: could not load/save pairing token (%v); using a temporary one\n", err)
			if s.token, err = pairing.GenerateToken(16); err != nil {
				s.token = fmt.Sprintf("tok-%d", time.Now().UnixNano())
			}
		}
	}
	// Local settings page key (127.0.0.1 only, owner-only file).
	var err error
	s.settingsKey, _, err = control.LoadOrCreateToken(filepath.Join(stateDir, "settings_key"), func() (string, error) { return pairing.GenerateToken(24) })
	if err != nil {
		return fmt.Errorf("settings key: %w", err)
	}

	s.localIPs = getLocalIPv4s()

	collector := telemetry.NewCollector(1200 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector.Start(ctx)

	s.updates = update.NewChecker()
	s.updates.SetAuto(cfg.AutoUpdate)
	s.updates.SetPeriodicChecks(cfg.UpdateCheck)
	if !buildinfo.IsDev() {
		// Every 6 h; skipped while both update_check and auto_update are off.
		s.updates.RunPeriodic(ctx, 6*time.Hour, s.autoInstalled)
	}

	opts := api.Options{ReadOnly: cfg.ReadOnly, NoScreen: cfg.NoScreen, Version: buildinfo.Version, Updates: s.updates, Restart: s.restart}
	if !cfg.ReadOnly {
		pin, err := control.NewPIN(func(newPIN, reason string) {
			fmt.Printf("\n  [pairing] %s. New control PIN: %s\n", reason, newPIN)
		})
		if err != nil {
			return fmt.Errorf("could not generate control PIN: %w", err)
		}
		s.pin = pin
		opts.PIN = pin
		sessions, err := control.LoadSessions(sessionsFile)
		if err != nil {
			fmt.Printf("Warning: could not load saved pairings (%v); dashboards will need a new PIN\n", err)
			sessions = nil
		}
		s.sessions = sessions
		opts.Sessions = sessions
		opts.ActionsFile = filepath.Join(stateDir, "actions.json")
	}
	handler := api.NewServer(cfg.Name, s.token, collector, opts).Handler()

	// Local settings page (loopback only) — started first so `status`,
	// `stop` and the tray can always reach the server.
	settingsLn, err := listenRetry("127.0.0.1", cfg.SettingsPort)
	if err != nil {
		return fmt.Errorf("settings page port %d: %w", cfg.SettingsPort, err)
	}
	s.closers = append(s.closers, func() { _ = settingsLn.Close() })
	go func() { _ = http.Serve(settingsLn, s.settingsHandler()) }()

	// Tailcat tunnel.
	if !cfg.NoTailcat {
		fmt.Print("Initializing Tailcat WireGuard data-plane...")
		startupCtx, startupCancel := context.WithTimeout(ctx, 30*time.Second)
		tunnel, err := tailcat.StartTunnel(startupCtx, tailcat.Config{
			Port:        cfg.TailcatPort,
			KeyFilePath: filepath.Join(stateDir, "server_key.json"),
			Verbose:     x.verbose,
		})
		startupCancel()
		if err != nil {
			fmt.Printf("\n[Tailcat Note] Direct tunnel bootstrap deferred or unavailable: %v\n", err)
			fmt.Println("  (Operating via Direct LAN/Tailscale IP fallback)")
		} else {
			s.tailcatAddr = tunnel.Addr()
			s.closers = append(s.closers, func() { _ = tunnel.Close() })
			fmt.Println(" Ready!")
			go func() { _ = http.Serve(tunnel.Listener(), handler) }()
		}
	}

	// Direct LAN listener.
	if cfg.Port > 0 {
		ln, err := listenRetry("", cfg.Port)
		if err != nil {
			fmt.Printf("Warning: Failed to bind direct port :%d: %v\n", cfg.Port, err)
		} else {
			s.closers = append(s.closers, func() { _ = ln.Close() })
			go func() { _ = http.Serve(ln, handler) }()
		}
	}

	// Pairing link + QR.
	s.pairingURL = pairing.BuildPairingURL(cfg.WebURL, cfg.Name, s.tailcatAddr, s.token, cfg.TailcatPort, cfg.Port, s.localIPs)
	matrix, qrErr := pairing.EncodeQR(s.pairingURL)
	if qrErr == nil {
		s.svgQR = pairing.GenerateSVG(matrix, 6)
	}
	info := pairing.Info{
		ServerName: cfg.Name, TailcatAddr: s.tailcatAddr, TailcatPort: cfg.TailcatPort, DirectPort: cfg.Port,
		LocalIPs: s.localIPs, PairingToken: s.token, WebClientURL: cfg.WebURL, FullPairingURL: s.pairingURL,
	}
	if infoJSON, err := json.MarshalIndent(info, "", "  "); err == nil {
		_ = control.WriteFileAtomic(filepath.Join(stateDir, "pairing_info.json"), infoJSON)
	}
	if cfg.ExportHTML != "" && s.svgQR != "" {
		_ = pairing.ExportPairingHTML(cfg.ExportHTML, info, s.svgQR)
	}

	s.printBanner(matrix, x.background)

	// Wait for Ctrl+C / SIGTERM or a stop/restart request.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	case <-s.stopped:
	}
	fmt.Println("\nShutting down PC Monitor server...")
	cancel()
	for _, c := range s.closers {
		c()
	}
	if s.restartAfter {
		if _, err := service.StartDetached(); err != nil {
			fmt.Println("Restart failed:", err)
		} else {
			fmt.Println("Restarted in the background.")
		}
	}
	fmt.Println("Server stopped cleanly.")
	return nil
}

func (s *server) printBanner(matrix [][]bool, background bool) {
	cfg := s.cfg
	fmt.Printf("\n  Server Name:    %s (%s/%s)\n", cfg.Name, runtime.GOOS, runtime.GOARCH)
	if s.tailcatAddr != "" {
		fmt.Printf("  Tailcat Addr:   %s\n", s.tailcatAddr)
	}
	if len(s.localIPs) > 0 && cfg.Port > 0 {
		fmt.Printf("  Direct LAN URL: http://%s:%d\n", s.localIPs[0], cfg.Port)
	}
	fmt.Printf("  Pairing Link:   %s\n", s.pairingURL)
	fmt.Printf("  Settings page:  run `pc_monitor settings` (http://127.0.0.1:%d)\n", cfg.SettingsPort)
	if len(matrix) > 0 && !background {
		fmt.Println("\n  Scan with your phone camera to pair:")
		fmt.Println(pairing.RenderTerminalQR(matrix))
	}
	fmt.Println("------------------------------------------------------------------------")
	if s.pin != nil {
		fmt.Println("  Remote control: ENABLED. Use --read-only to disable.")
		fmt.Printf("  Control PIN:    %s   (one-time; also shown by `pc_monitor pin`)\n", s.pin.Current())
		if s.sessions != nil {
			fmt.Printf("  Paired:         %d dashboard(s) remembered across restarts\n", s.sessions.Count())
		}
	} else {
		fmt.Println("  Remote control: disabled (read-only mode). No PIN required.")
	}
	if !background {
		fmt.Println("  Status: Monitoring active. Press Ctrl+C to stop.")
	}
	fmt.Println("========================================================================")
}

// stop ends the run loop; restart additionally relaunches in the background.
func (s *server) stop(restart bool) {
	s.stopOnce.Do(func() {
		s.restartAfter = restart
		close(s.stopped)
	})
}

func (s *server) restart() { go func() { time.Sleep(500 * time.Millisecond); s.stop(true) }() }

// listenRetry retries briefly so a restarted server can take over ports the
// previous process is still releasing.
func listenRetry(host string, port int) (net.Listener, error) {
	addr := net.JoinHostPort(host, fmt.Sprint(port))
	var err error
	for i := 0; i < 20; i++ {
		var ln net.Listener
		if ln, err = net.Listen("tcp", addr); err == nil {
			return ln, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, err
}

func resetPairingFiles() error {
	for _, f := range []string{"pairing_token", "sessions.json"} {
		if err := os.Remove(filepath.Join(config.Dir(), f)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("could not reset pairing (%s): %w", f, err)
		}
	}
	return nil
}

func getLocalIPv4s() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if (iface.Flags&net.FlagLoopback) != 0 || (iface.Flags&net.FlagUp) == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() && !strings.HasPrefix(ip.String(), "169.254.") {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

// autoInstalled restarts into an update that was installed automatically.
func (s *server) autoInstalled() {
	fmt.Printf("%s Installed v%s automatically; restarting\n", time.Now().Format(time.RFC3339), s.updates.Status().Latest)
	s.restart()
}
