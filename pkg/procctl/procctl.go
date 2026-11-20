// Package procctl ends processes on the host for the dashboard's End task.
//
// Safety: callers pass the program name they expect; a PID is only ended if
// it still belongs to that program (so a recycled PID can't hit something
// else), and system-critical processes and PC Monitor itself are refused.
package procctl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrProtected is returned for processes that must never be ended remotely.
var ErrProtected = errors.New("this process is protected and can't be ended from the dashboard")

// protected lists programs whose termination crashes or locks up the OS (or
// cuts off this dashboard). Compared case-insensitively, without ".exe".
var protected = map[string][]string{
	"windows": {
		"system", "registry", "secure system", "memory compression", "idle",
		"smss", "csrss", "wininit", "winlogon", "services", "lsass", "lsaiso",
		"svchost", "dwm", "fontdrvhost", "logonui", "sihost", "ctfmon",
	},
	"linux":  {"systemd", "init", "kthreadd", "Xorg", "Xwayland", "gnome-shell", "kwin_x11", "kwin_wayland", "dbus-daemon", "sshd"},
	"darwin": {"launchd", "kernel_task", "WindowServer", "loginwindow", "opendirectoryd", "sshd"},
}

// Protected reports whether name (e.g. "csrss.exe") must not be ended.
func Protected(pid int, name string) bool {
	if pid <= 4 || pid == os.Getpid() || pid == os.Getppid() {
		return true // idle/system/init, this server and whatever launched it
	}
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)))
	if base == "" {
		base = strings.ToLower(name)
	}
	if strings.HasPrefix(base, "pc_monitor") || strings.HasPrefix(base, "pc-monitor") {
		return true
	}
	for _, p := range protected[runtime.GOOS] {
		if strings.ToLower(p) == base {
			return true
		}
	}
	return false
}

// End ends pid if it is still running as want (a program name such as
// "notepad.exe"); current is the name the host reports for pid right now.
func End(pid int, want, current string) error {
	if current == "" {
		return fmt.Errorf("process %d is no longer running", pid)
	}
	if !strings.EqualFold(current, want) {
		return fmt.Errorf("process %d is now %s, not %s; refresh and try again", pid, current, want)
	}
	if Protected(pid, current) {
		return ErrProtected
	}
	return end(pid)
}
