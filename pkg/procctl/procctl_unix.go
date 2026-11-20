//go:build !windows

package procctl

import (
	"errors"
	"fmt"
	"syscall"
)

// end asks the process to quit (SIGTERM), like closing it normally.
func end(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.EPERM) {
			return errors.New("permission denied: the process belongs to another user")
		}
		return fmt.Errorf("can't end process %d: %w", pid, err)
	}
	return nil
}
