//go:build !windows

package power

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// doSleep suspends the host after a short delay (untested on real hardware
// in this project's CI).
func doSleep() error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("systemctl", "suspend")
	case "darwin":
		cmd = exec.Command("pmset", "sleepnow")
	default:
		return fmt.Errorf("sleep is not supported on %s", runtime.GOOS)
	}
	go func() {
		time.Sleep(2 * time.Second)
		_ = cmd.Run()
	}()
	return nil
}
