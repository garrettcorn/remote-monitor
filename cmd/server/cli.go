package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"runtime"
	"strings"
	"time"

	"pc_monitor/pkg/buildinfo"
	"pc_monitor/pkg/config"
	"pc_monitor/pkg/pairing"
	"pc_monitor/pkg/service"
	"pc_monitor/pkg/update"
)

func runtimeOS() string { return runtime.GOOS }

func loadConfigOrDefaults() config.Config {
	c, err := config.Load()
	if err != nil {
		fmt.Println("Warning:", err)
	}
	return c
}

func waitFor(port int, running bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok, _ := pingSettings(port); ok == running {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func startCommand() error {
	cfg := loadConfigOrDefaults()
	if ok, ver := pingSettings(cfg.SettingsPort); ok {
		fmt.Printf("PC Monitor %s is already running. Open settings with: pc_monitor settings\n", ver)
		return nil
	}
	pid, err := service.StartDetached()
	if err != nil {
		return err
	}
	if !waitFor(cfg.SettingsPort, true, 20*time.Second) {
		return fmt.Errorf("server didn't start; see the log at %s", service.LogPath())
	}
	fmt.Printf("PC Monitor started in the background (pid %d).\n", pid)
	fmt.Println("  Pairing link and PIN: pc_monitor pin")
	fmt.Println("  Settings page:        pc_monitor settings")
	return nil
}

func stopCommand() error {
	cfg := loadConfigOrDefaults()
	if ok, _ := pingSettings(cfg.SettingsPort); !ok {
		fmt.Println("PC Monitor is not running.")
		return nil
	}
	if err := settingsCall(cfg.SettingsPort, "POST", "/api/stop", nil, nil); err != nil {
		return err
	}
	if !waitFor(cfg.SettingsPort, false, 15*time.Second) {
		return errors.New("server didn't stop in time")
	}
	fmt.Println("PC Monitor stopped.")
	return nil
}

type stateResp struct {
	Version   string `json:"version"`
	Autostart struct {
		Enabled bool   `json:"enabled"`
		Where   string `json:"where"`
	} `json:"autostart"`
	Update  update.Status `json:"update"`
	Pairing struct {
		URL       string `json:"url"`
		PIN       string `json:"pin"`
		Paired    int    `json:"paired"`
		Control   bool   `json:"control"`
		DirectURL string `json:"direct_url"`
	} `json:"pairing"`
	Log string `json:"log"`
}

func statusCommand() error {
	cfg := loadConfigOrDefaults()
	fmt.Printf("PC Monitor %s (%s/%s)\n", buildinfo.Version, runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Start at login: %s\n", onOff(service.AutostartEnabled()))
	ok, ver := pingSettings(cfg.SettingsPort)
	if !ok {
		fmt.Println("  Server:         not running (start it with: pc_monitor start)")
		return nil
	}
	var st stateResp
	if err := settingsCall(cfg.SettingsPort, "GET", "/api/state", nil, &st); err != nil {
		fmt.Printf("  Server:         running %s (details unavailable: %v)\n", ver, err)
		return nil
	}
	fmt.Printf("  Server:         running %s\n", st.Version)
	if st.Pairing.Control {
		fmt.Printf("  Remote control: on · %d paired dashboard(s)\n", st.Pairing.Paired)
	} else {
		fmt.Println("  Remote control: off (read-only)")
	}
	if st.Update.Available {
		fmt.Printf("  Update:         %s available → run: pc_monitor update\n", st.Update.Latest)
	} else if st.Update.Latest != "" {
		fmt.Println("  Update:         up to date")
	}
	fmt.Printf("  Log:            %s\n", st.Log)
	return nil
}

func settingsCommand() error {
	cfg := loadConfigOrDefaults()
	if ok, _ := pingSettings(cfg.SettingsPort); !ok {
		fmt.Println("Starting PC Monitor...")
		if err := startCommand(); err != nil {
			return err
		}
	}
	url := fmt.Sprintf("%s/#key=%s", settingsBase(cfg.SettingsPort), readSettingsKey())
	fmt.Printf("Opening settings: %s/\n", settingsBase(cfg.SettingsPort))
	return service.OpenBrowser(url)
}

func pinCommand() error {
	cfg := loadConfigOrDefaults()
	if ok, _ := pingSettings(cfg.SettingsPort); !ok {
		return errors.New("PC Monitor is not running (start it with: pc_monitor start)")
	}
	var st stateResp
	if err := settingsCall(cfg.SettingsPort, "GET", "/api/state", nil, &st); err != nil {
		return err
	}
	fmt.Println("Pairing link:", st.Pairing.URL)
	if m, err := pairing.EncodeQR(st.Pairing.URL); err == nil {
		fmt.Println(pairing.RenderTerminalQR(m))
	}
	if st.Pairing.Control {
		fmt.Println("Control PIN: ", st.Pairing.PIN, "(one-time; a new one appears after each use)")
	} else {
		fmt.Println("Read-only server: no PIN needed.")
	}
	return nil
}

func configCommand(args []string) error {
	if len(args) == 0 {
		args = []string{"list"}
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		for _, k := range config.Keys() {
			v, _ := cfg.Get(k)
			fmt.Printf("  %-14s %s\n", k, v)
		}
		fmt.Println("File:", config.Path())
	case "path":
		fmt.Println(config.Path())
	case "get":
		if len(args) != 2 {
			return errors.New("usage: pc_monitor config get <key>")
		}
		v, err := cfg.Get(args[1])
		if err != nil {
			return err
		}
		fmt.Println(v)
	case "set":
		if len(args) != 3 {
			return errors.New("usage: pc_monitor config set <key> <value>")
		}
		if err := cfg.Set(args[1], args[2]); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Printf("Saved %s = %s\n", args[1], args[2])
		if ok, _ := pingSettings(cfg.SettingsPort); ok {
			fmt.Println("Restart to apply: pc_monitor restart")
		}
	default:
		return fmt.Errorf("unknown config action %q (list, get, set, path)", args[0])
	}
	return nil
}

func autostartCommand(args []string) error {
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "on", "enable":
		if err := service.AutostartEnable(); err != nil {
			return err
		}
		fmt.Println("PC Monitor will start when you log in.")
		fmt.Println("  Entry:", service.AutostartDescription())
	case "off", "disable":
		if err := service.AutostartDisable(); err != nil {
			return err
		}
		fmt.Println("PC Monitor will no longer start at login.")
	case "status":
		fmt.Println("Start at login:", onOff(service.AutostartEnabled()))
	default:
		return fmt.Errorf("usage: pc_monitor autostart on|off|status")
	}
	return nil
}

func updateCommand(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	checkOnly := fs.Bool("check", false, "Only check for a new version")
	if err := fs.Parse(args); err != nil {
		return err
	}
	c := update.NewChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	st := c.Check(ctx)
	if st.Error != "" {
		return errors.New(st.Error)
	}
	fmt.Printf("Installed: %s · Latest: %s\n", st.Current, st.Latest)
	if !st.Available {
		fmt.Println("You're up to date.")
		return nil
	}
	if *checkOnly {
		fmt.Println("Update available:", st.URL)
		return nil
	}
	if !st.CanApply {
		return errors.New(strings.TrimSpace(st.Note + " "))
	}
	cfg := loadConfigOrDefaults()
	wasRunning, _ := pingSettings(cfg.SettingsPort)
	fmt.Printf("Downloading and verifying %s...\n", st.Latest)
	if err := c.Apply(ctx); err != nil {
		return err
	}
	fmt.Println("Installed", st.Latest)
	if wasRunning {
		fmt.Println("Restarting the background server...")
		if err := stopCommand(); err != nil {
			return err
		}
		return startCommand()
	}
	return nil
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
