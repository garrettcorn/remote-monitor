//go:build windows

package procctl

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

// end terminates the process (like Task Manager's End task). Processes of
// another user or running elevated need the server to run as administrator.
func end(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return errors.New("access denied: the process runs as administrator or another user")
		}
		return fmt.Errorf("can't open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(h)
	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("can't end process %d: %w", pid, err)
	}
	return nil
}
