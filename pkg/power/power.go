// Package power shuts down, restarts or sleeps the host.
package power

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// Action is a power action.
type Action string

const (
	Shutdown Action = "shutdown"
	Restart  Action = "restart"
	Sleep    Action = "sleep"
)

// Do schedules the action a few seconds out so the API response can be
// delivered before the machine goes down.
func Do(a Action) error {
	if a == Sleep {
		return doSleep()
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		flag := map[Action]string{Shutdown: "/s", Restart: "/r"}[a]
		if flag == "" {
			return fmt.Errorf("unknown action %q", a)
		}
		cmd = exec.Command("shutdown", flag, "/t", "5", "/c", "Requested from PC Monitor dashboard")
	case "linux":
		verb := map[Action]string{Shutdown: "poweroff", Restart: "reboot"}[a]
		if verb == "" {
			return fmt.Errorf("unknown action %q", a)
		}
		cmd = exec.Command("systemctl", verb)
	case "darwin":
		verb := map[Action]string{Shutdown: "shut down", Restart: "restart"}[a]
		if verb == "" {
			return fmt.Errorf("unknown action %q", a)
		}
		cmd = exec.Command("osascript", "-e", `tell application "System Events" to `+verb)
	default:
		return fmt.Errorf("power control is not supported on %s", runtime.GOOS)
	}
	if runtime.GOOS != "windows" {
		// These act immediately; delay so the HTTP response gets out first.
		go func() {
			time.Sleep(2 * time.Second)
			_ = cmd.Run()
		}()
		return nil
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}
