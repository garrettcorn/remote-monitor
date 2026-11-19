//go:build windows

package power

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modpowrprof         = windows.NewLazySystemDLL("powrprof.dll")
	procSetSuspendState = modpowrprof.NewProc("SetSuspendState")
)

// doSleep puts the PC to sleep (not hibernate) after a short delay so the
// HTTP response is delivered first. SetSuspendState needs SeShutdownPrivilege
// enabled on the process token.
func doSleep() error {
	if err := procSetSuspendState.Find(); err != nil {
		return fmt.Errorf("sleep is not available: %w", err)
	}
	if err := enableShutdownPrivilege(); err != nil {
		return fmt.Errorf("enable shutdown privilege: %w", err)
	}
	go func() {
		time.Sleep(2 * time.Second)
		// bHibernate=FALSE, bForce=FALSE, bWakeupEventsDisabled=FALSE
		procSetSuspendState.Call(0, 0, 0)
	}()
	return nil
}

func enableShutdownPrivilege() error {
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	var luid windows.LUID
	name, _ := windows.UTF16PtrFromString("SeShutdownPrivilege")
	if err := windows.LookupPrivilegeValue(nil, name, &luid); err != nil {
		return err
	}
	tp := windows.Tokenprivileges{PrivilegeCount: 1}
	tp.Privileges[0] = windows.LUIDAndAttributes{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}
	return windows.AdjustTokenPrivileges(token, false, &tp, uint32(unsafe.Sizeof(tp)), nil, nil)
}
