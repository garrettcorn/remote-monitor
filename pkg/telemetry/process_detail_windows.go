//go:build windows

package telemetry

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetProcessHandleCount = modkernel32.NewProc("GetProcessHandleCount")

// processMemoryCountersEx mirrors PROCESS_MEMORY_COUNTERS_EX.
type processMemoryCountersEx struct {
	processMemoryCounters
	PrivateUsage uintptr
}

// ProcessDetails returns details for one process. Fields the server isn't
// allowed to read (other users' or protected processes) are left empty.
func (c *Collector) ProcessDetails(pid int) (ProcessDetail, error) {
	d := ProcessDetail{PID: pid}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return d, err
	}
	defer windows.CloseHandle(snapshot)
	names := map[uint32]string{}
	found := false
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		names[entry.ProcessID] = windows.UTF16ToString(entry.ExeFile[:])
		if int(entry.ProcessID) == pid {
			found = true
			d.Name = names[entry.ProcessID]
			d.Threads = int(entry.Threads)
			d.ParentPID = int(entry.ParentProcessID)
		}
	}
	if !found {
		return d, fmt.Errorf("process %d is not running", pid)
	}
	d.ParentName = names[uint32(d.ParentPID)]

	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return d, nil // protected: name, parent and threads only
	}
	defer windows.CloseHandle(h)
	d.Path = processPath(h)
	d.Title = c.titleFor(d.Path, d.Name)
	var created, exited, kernel, user windows.Filetime
	if windows.GetProcessTimes(h, &created, &exited, &kernel, &user) == nil {
		t := time.Unix(0, created.Nanoseconds())
		d.Started = &t
	}
	var n uint32
	if r, _, _ := procGetProcessHandleCount.Call(uintptr(h), uintptr(unsafe.Pointer(&n))); r != 0 {
		d.Handles = int(n)
	}
	var pmc processMemoryCountersEx
	pmc.cb = uint32(unsafe.Sizeof(pmc))
	if r, _, _ := procK32GetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&pmc)), uintptr(pmc.cb)); r != 0 {
		d.MemoryBytes = uint64(pmc.WorkingSetSize)
		d.PeakBytes = uint64(pmc.PeakWorkingSetSize)
		d.PrivateBytes = uint64(pmc.PrivateUsage)
	}
	var tok windows.Token
	if windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok) == nil {
		defer tok.Close()
		if tu, err := tok.GetTokenUser(); err == nil {
			if acct, dom, _, err := tu.User.Sid.LookupAccount(""); err == nil {
				d.User = acct
				if dom != "" {
					d.User = dom + `\` + acct
				}
			}
		}
		elevated := tok.IsElevated()
		d.Elevated = &elevated
	}
	return d, nil
}
