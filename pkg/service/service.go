// Package service starts the server in the background, opens the browser
// and manages "start at login".
//
// Start-at-login deliberately runs inside the user's desktop session (not
// as a system service): screen capture, audio control, button actions and
// device queries only work there.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"pc_monitor/pkg/config"
)

// Executable returns the program to launch at login or when restarting:
// the AppImage file if running from one, the windowless companion app on
// Windows if present, else this executable.
func Executable() (path string, gui bool) {
	if p := os.Getenv("APPIMAGE"); p != "" {
		return p, false
	}
	exe, err := os.Executable()
	if err != nil {
		return os.Args[0], false
	}
	exe, _ = filepath.EvalSymlinks(exe)
	if runtime.GOOS == "windows" {
		app := filepath.Join(filepath.Dir(exe), "pc_monitor-app.exe")
		if filepath.Base(exe) == "pc_monitor-app.exe" {
			return exe, true
		}
		if _, err := os.Stat(app); err == nil {
			return app, true
		}
	}
	return exe, false
}

// LogPath is where background runs write their output.
func LogPath() string { return filepath.Join(config.Dir(), "server.log") }

// StartDetached launches `<exe> run --background` without a console window
// and returns immediately. Output goes to LogPath().
func StartDetached() (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		return 0, err
	}
	logf, err := os.OpenFile(LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, err
	}
	defer logf.Close()
	cmd := exec.Command(exe, "run", "--background")
	if p := os.Getenv("APPIMAGE"); p != "" {
		cmd = exec.Command(p, "run", "--background")
	}
	cmd.Stdout, cmd.Stderr = logf, logf
	cmd.SysProcAttr = detachedAttr()
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}

// OpenBrowser opens a URL in the default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("couldn't open a browser (%v); visit %s", err, url)
	}
	return nil
}
