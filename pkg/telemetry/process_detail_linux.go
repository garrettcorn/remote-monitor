//go:build linux

package telemetry

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

// ProcessDetails reads /proc/<pid>. Another user's open files and exe path
// aren't readable without root; those fields stay empty.
func (c *Collector) ProcessDetails(pid int) (ProcessDetail, error) {
	d := ProcessDetail{PID: pid}
	dir := fmt.Sprintf("/proc/%d", pid)
	status, err := os.ReadFile(dir + "/status")
	if err != nil {
		return d, fmt.Errorf("process %d is not running", pid)
	}
	st := parseProcStatus(string(status))
	d.Name, d.ParentPID, d.Threads = st.Name, st.PPid, st.Threads
	d.MemoryBytes, d.PeakBytes = st.RSSKB*1024, st.HWMKB*1024
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", st.PPid)); err == nil {
		d.ParentName = strings.TrimSpace(string(b))
	}
	if u, err := user.LookupId(st.UID); err == nil {
		d.User = u.Username
	} else {
		d.User = st.UID
	}
	root := st.UID == "0"
	d.Elevated = &root
	if p, err := os.Readlink(dir + "/exe"); err == nil {
		d.Path = p
	}
	if fds, err := os.ReadDir(dir + "/fd"); err == nil {
		d.Handles = len(fds)
	}
	if data, err := os.ReadFile(dir + "/stat"); err == nil {
		if _, _, startTicks, _, ok := parseProcStat(string(data)); ok {
			if boot, ok := bootTime(); ok {
				t := boot.Add(time.Duration(startTicks) * time.Second / 100)
				d.Started = &t
			}
		}
	}
	return d, nil
}

// bootTime reads btime from /proc/stat.
func bootTime() (time.Time, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(line, "btime "); ok {
			if n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil {
				return time.Unix(n, 0), true
			}
		}
	}
	return time.Time{}, false
}
