//go:build !windows

package service

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"pc_monitor/pkg/control"
)

const launchAgentLabel = "com.garrettcorn.pc-monitor"

func detachedAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Setsid: true} }

// autostartFile: XDG autostart entry on Linux (runs inside the desktop
// session, so screen/audio/input features work), LaunchAgent on macOS.
func autostartFile() string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(home, ".config")
	}
	return filepath.Join(cfg, "autostart", "pc-monitor.desktop")
}

// AutostartEnable registers the server to start when the user logs in.
func AutostartEnable() error {
	exe, _ := Executable()
	var content string
	if runtime.GOOS == "darwin" {
		content = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>%s</string><string>run</string><string>--background</string></array>
  <key>RunAtLoad</key><true/>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>
`, launchAgentLabel, html.EscapeString(exe), html.EscapeString(LogPath()), html.EscapeString(LogPath()))
	} else {
		// Desktop Entry Exec quoting: wrap in quotes, escape \ " ` $.
		q := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`).Replace(exe)
		content = fmt.Sprintf("[Desktop Entry]\nType=Application\nName=PC Monitor\nComment=PC Monitor server\nExec=\"%s\" start\nX-GNOME-Autostart-enabled=true\nTerminal=false\n", q)
	}
	if err := os.MkdirAll(filepath.Dir(autostartFile()), 0o755); err != nil {
		return err
	}
	return control.WriteFileAtomic(autostartFile(), []byte(content))
}

// AutostartDisable removes the login entry.
func AutostartDisable() error {
	if err := os.Remove(autostartFile()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AutostartEnabled reports whether a login entry exists.
func AutostartEnabled() bool {
	_, err := os.Stat(autostartFile())
	return err == nil
}

// AutostartDescription says where the entry lives.
func AutostartDescription() string { return autostartFile() }
