package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"pc_monitor/pkg/api"
	"pc_monitor/pkg/control"
	"pc_monitor/pkg/pairing"
	"pc_monitor/pkg/tailcat"
	"pc_monitor/pkg/telemetry"
)

func main() {
	defaultHostname, _ := os.Hostname()
	if defaultHostname == "" {
		defaultHostname = "pc-server"
	}

	serverName := flag.String("name", defaultHostname, "Friendly name for this server in the dashboard")
	directPort := flag.Int("port", 9120, "Direct HTTP/WebSocket port for LAN/Tailscale IP access (0 to disable)")
	tcPort := flag.Int("tc-port", 8080, "Tailcat tunnel port")
	webURL := flag.String("web-url", "https://garrettcorn.github.io/remote-monitor/", "URL of the GitHub Pages dashboard")
	tokenFlag := flag.String("token", "", "Pairing authentication token (auto-generated if empty)")
	noTailcat := flag.Bool("no-tailcat", false, "Disable Tailcat tunnel (run in direct LAN mode only)")
	exportHTML := flag.String("export-html", "pairing_card.html", "Path to save scannable pairing HTML card")
	readOnly := flag.Bool("read-only", false, "Monitoring only: disable remote control (audio, shutdown/restart); no PIN needed")
	resetPairing := flag.Bool("reset-pairing", false, "Revoke all paired dashboards: new pairing token and PIN sessions")
	noScreen := flag.Bool("no-screen", false, "Disable screen capture previews in the dashboard")
	verbose := flag.Bool("verbose", false, "Enable verbose debug output")
	flag.Parse()

	fmt.Println("\n========================================================================")
	fmt.Println("  PC MONITOR SERVER - Cross-Platform System Telemetry & Pairing")
	fmt.Println("========================================================================")

	// Pairing state lives next to the Tailcat key so paired dashboards keep
	// working across restarts.
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".pc_monitor")
	tokenFile := filepath.Join(stateDir, "pairing_token")
	sessionsFile := filepath.Join(stateDir, "sessions.json")
	if *resetPairing {
		for _, f := range []string{tokenFile, sessionsFile} {
			if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
				fmt.Printf("Fatal: could not reset pairing (%s): %v\n", f, err)
				os.Exit(1)
			}
		}
		fmt.Println("  Pairing reset: all previously paired dashboards have been revoked.")
	}

	// 1. Use the -token flag, else the saved token, else create and save one.
	token := *tokenFlag
	if token == "" {
		var err error
		token, _, err = control.LoadOrCreateToken(tokenFile, func() (string, error) { return pairing.GenerateToken(16) })
		if err != nil {
			fmt.Printf("Warning: could not load/save pairing token (%v); using a temporary one\n", err)
			token, err = pairing.GenerateToken(16)
			if err != nil {
				token = fmt.Sprintf("tok-%d", time.Now().UnixNano())
			}
		}
	}

	// 2. Discover local IP addresses
	localIPs := getLocalIPv4s()

	// 3. Start Telemetry Collector
	collector := telemetry.NewCollector(1200 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector.Start(ctx)

	// 4. Create API Server. Unless read-only, dashboards must enter a
	// one-time PIN (shown only here on the console) to connect; each used
	// PIN is replaced by a new one for the next dashboard.
	opts := api.Options{ReadOnly: *readOnly, NoScreen: *noScreen}
	if !*readOnly {
		pin, err := control.NewPIN(func(newPIN, reason string) {
			fmt.Printf("\n  [pairing] %s. New control PIN: %s\n", reason, newPIN)
		})
		if err != nil {
			fmt.Printf("Fatal: could not generate control PIN: %v\n", err)
			os.Exit(1)
		}
		opts.PIN = pin
		sessions, err := control.LoadSessions(sessionsFile)
		if err != nil {
			fmt.Printf("Warning: could not load saved pairings (%v); dashboards will need a new PIN\n", err)
			sessions = nil
		}
		opts.Sessions = sessions
		opts.ActionsFile = filepath.Join(stateDir, "actions.json")
	}
	apiServer := api.NewServer(*serverName, token, collector, opts)
	httpHandler := apiServer.Handler()

	// 5. Start Tailcat Tunnel
	var tunnel *tailcat.Tunnel
	var tailcatAddr string
	if !*noTailcat {
		fmt.Print("Initializing Tailcat WireGuard data-plane...")
		homeDir, _ := os.UserHomeDir()
		keyFile := filepath.Join(homeDir, ".pc_monitor", "server_key.json")

		startupCtx, startupCancel := context.WithTimeout(ctx, 30*time.Second)
		var err error
		tunnel, err = tailcat.StartTunnel(startupCtx, tailcat.Config{
			Port:        *tcPort,
			KeyFilePath: keyFile,
			Verbose:     *verbose,
		})
		startupCancel()

		if err != nil {
			fmt.Printf("\n[Tailcat Note] Direct tunnel bootstrap deferred or unavailable: %v\n", err)
			fmt.Println("  (Operating seamlessly via Direct LAN/Tailscale IP fallback)")
		} else {
			tailcatAddr = tunnel.Addr()
			fmt.Println(" Ready!")

			// Serve API over Tailcat listener
			go func() {
				if err := http.Serve(tunnel.Listener(), httpHandler); err != nil && err != http.ErrServerClosed {
					if *verbose {
						fmt.Printf("[tailcat http error] %v\n", err)
					}
				}
			}()
		}
	}

	// 6. Start Direct LAN/Local Listener
	var directListener net.Listener
	if *directPort > 0 {
		var err error
		directListener, err = net.Listen("tcp", fmt.Sprintf(":%d", *directPort))
		if err != nil {
			fmt.Printf("Warning: Failed to bind direct port :%d: %v\n", *directPort, err)
		} else {
			go func() {
				if err := http.Serve(directListener, httpHandler); err != nil && err != http.ErrServerClosed {
					if *verbose {
						fmt.Printf("[direct http error] %v\n", err)
					}
				}
			}()
		}
	}

	// 7. Build Pairing URLs and QR Code
	pairingURL := pairing.BuildPairingURL(*webURL, *serverName, tailcatAddr, token, *tcPort, *directPort, localIPs)

	info := pairing.Info{
		ServerName:     *serverName,
		TailcatAddr:    tailcatAddr,
		TailcatPort:    *tcPort,
		DirectPort:     *directPort,
		LocalIPs:       localIPs,
		PairingToken:   token,
		WebClientURL:   *webURL,
		FullPairingURL: pairingURL,
	}

	// Save pairing_info.json
	infoJSON, _ := json.MarshalIndent(info, "", "  ")
	_ = os.WriteFile("pairing_info.json", infoJSON, 0644)

	// Generate QR Code
	var svgQR string
	matrix, err := pairing.EncodeQR(pairingURL)
	if err == nil {
		svgQR = pairing.GenerateSVG(matrix, 6)
		_ = pairing.ExportPairingHTML(*exportHTML, info, svgQR)
	}

	// Print pairing details to console
	fmt.Printf("\n  Server Name:    %s (%s/%s)\n", *serverName, runtime.GOOS, runtime.GOARCH)
	if tailcatAddr != "" {
		fmt.Printf("  Tailcat Addr:   %s\n", tailcatAddr)
		fmt.Printf("  Tailcat Port:   %d\n", *tcPort)
	}
	fmt.Printf("  Pairing Token:  %s\n", token)
	if len(localIPs) > 0 && *directPort > 0 {
		fmt.Printf("  Direct LAN URL: http://%s:%d\n", localIPs[0], *directPort)
	}
	fmt.Printf("  Pairing Link:   %s\n", pairingURL)
	if *exportHTML != "" {
		fmt.Printf("  Pairing Card:   %s (saved)\n", *exportHTML)
	}

	if len(matrix) > 0 {
		fmt.Println("\n  ┌─────────────────────────────────────────┐")
		fmt.Println("  │  Pairing QR Code                        │")
		fmt.Println("  └─────────────────────────────────────────┘")
		fmt.Println("  Scan with your phone camera to pair:")
		fmt.Println(pairing.RenderTerminalQR(matrix))
	}

	fmt.Println("------------------------------------------------------------------------")
	if opts.PIN != nil {
		fmt.Println("  Remote control: ENABLED (audio, shutdown/restart). Use -read-only to disable.")
		fmt.Printf("  Control PIN:    %s   (enter in the dashboard to connect; one-time use)\n", opts.PIN.Current())
		if opts.Sessions != nil {
			fmt.Printf("  Paired:         %d dashboard(s) remembered across restarts (-reset-pairing to revoke)\n", opts.Sessions.Count())
		}
		fmt.Printf("  Launch buttons: apps allowed for Stream Deck buttons are listed in %s\n", opts.ActionsFile)
		if *noScreen {
			fmt.Println("  Screen preview: disabled (-no-screen)")
		} else {
			fmt.Println("  Screen preview: available to PIN-paired dashboards (-no-screen to disable)")
		}
	} else {
		fmt.Println("  Remote control: disabled (read-only mode). No PIN required.")
	}
	fmt.Println("  Status: Monitoring active. Press Ctrl+C to terminate.")
	fmt.Println("========================================================================")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down PC Monitor server...")
	cancel()
	if directListener != nil {
		_ = directListener.Close()
	}
	if tunnel != nil {
		_ = tunnel.Close()
	}
	fmt.Println("Server stopped cleanly. Goodbye.")
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
			ip := ipnet.IP.To4()
			if ip != nil && !ip.IsLoopback() {
				// Prefer non-link-local (169.254.x.x)
				if !strings.HasPrefix(ip.String(), "169.254.") {
					ips = append(ips, ip.String())
				}
			}
		}
	}
	return ips
}
